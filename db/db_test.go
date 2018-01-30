package db_test

import (
	"encoding/json"
	"math"
	"time"

	cf "github.com/alphagov/paas-billing/cloudfoundry"
	. "github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	postgresTimeFormat = "2006-01-02T15:04:05.000000Z"
)

var _ = Describe("Db", func() {

	var (
		sqlClient *PostgresClient
		connstr   string
	)

	BeforeEach(func() {
		var err error
		connstr, err = dbhelper.CreateDB()
		Expect(err).ToNot(HaveOccurred())
		sqlClient, err = NewPostgresClient(connstr)
		Expect(err).ToNot(HaveOccurred())
		err = sqlClient.InitSchema()
		Expect(err).ToNot(HaveOccurred())

		// time.Sleep(1 * time.Hour)
		// sig := make(chan os.Signal, 1)
		// signal.Notify(sig, os.Interrupt)
		// <-sig
		// fmt.Fprintln(GinkgoWriter, "shutdown")
	})

	AfterEach(func() {
		err := sqlClient.Close()
		Expect(err).ToNot(HaveOccurred())
		err = dbhelper.DropDB(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	Specify("schema application is idempotent", func() {
		Expect(sqlClient.InitSchema()).To(Succeed())
	})

	TestUsageEvents := func(tableName string) {
		var (
			testData   *cf.UsageEventList
			event1GUID string
			event2GUID string
			sampleData json.RawMessage
		)

		BeforeEach(func() {
			event1GUID = "2C5D3E72-2082-43C1-9262-814EAE7E65AA"
			event2GUID = "968437F2-CCEE-4B8E-B29B-34EA701BA196"
			sampleData = json.RawMessage(`{"field":"value"}`)
			testData = &cf.UsageEventList{
				Resources: []cf.UsageEvent{
					cf.UsageEvent{
						MetaData:  cf.MetaData{GUID: event1GUID},
						EntityRaw: sampleData,
					},
					cf.UsageEvent{
						MetaData:  cf.MetaData{GUID: event2GUID},
						EntityRaw: sampleData,
					},
				},
			}
		})

		Context("given an empty table", func() {
			It("returns a nil GUID", func() {
				guid, err := sqlClient.FetchLastGUID(tableName)
				Expect(err).ToNot(HaveOccurred())
				Expect(guid).To(Equal(cf.GUIDNil))
			})

			It("can store events", func() {
				By("inserting data")
				err := sqlClient.InsertUsageEventList(testData, tableName)
				Expect(err).ToNot(HaveOccurred())
				By("reading it back in the correct order")
				usageEvents, err := selectUsageEvents(sqlClient, tableName)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(usageEvents.Resources)).To(Equal(2))
				Expect(usageEvents.Resources[0].MetaData.GUID).To(Equal(event1GUID))
				Expect(usageEvents.Resources[1].MetaData.GUID).To(Equal(event2GUID))
			})
		})

		Context("given a table of data", func() {
			BeforeEach(func() {
				err := sqlClient.InsertUsageEventList(testData, tableName)
				Expect(err).ToNot(HaveOccurred())
			})

			It("can retrieve the GUID of the latest event", func() {
				guid, err := sqlClient.FetchLastGUID(tableName)
				Expect(err).ToNot(HaveOccurred())
				Expect(guid).To(Equal(event2GUID))
			})

			Context("when transactions fail", func() {

				var dataThatFailsToInsert *cf.UsageEventList

				BeforeEach(func() {
					dataThatFailsToInsert = &cf.UsageEventList{
						Resources: []cf.UsageEvent{
							cf.UsageEvent{
								MetaData:  cf.MetaData{GUID: "new-guid"},
								EntityRaw: sampleData,
							},
							cf.UsageEvent{
								// We cause transaction failure by entering an existing GUID into a UNIQUE column
								MetaData:  cf.MetaData{GUID: "968437F2-CCEE-4B8E-B29B-34EA701BA196"},
								EntityRaw: sampleData,
							},
						},
					}
				})

				It("it will rollback insertions of data", func() {
					err := sqlClient.InsertUsageEventList(dataThatFailsToInsert, tableName)
					Expect(err).To(HaveOccurred())
					usageEvents, err := selectUsageEvents(sqlClient, tableName)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(usageEvents.Resources)).To(Equal(2))
				})
			})
		})
	}

	Describe("App usage events", func() {
		TestUsageEvents(AppUsageTableName)
	})

	Describe("Service usage events", func() {
		TestUsageEvents(ServiceUsageTableName)
	})

	Describe("Pricing Formulae", func() {

		var insert = func(formula string, out interface{}) error {
			return sqlClient.Conn.QueryRow(`
				insert into pricing_plans(name, valid_from, plan_guid, formula) values (
					'FormulaTestPlan',
					'-infinity',
					$1,
					$2
				) returning eval_formula(64, tstzrange(now(), now() + '60 seconds'), formula) as result
			`, uuid.NewV4().String(), formula).Scan(out)
		}

		It("Should allow basic integer formulae", func() {
			var out int
			err := insert("((2 * 2::integer) + 1 - 1) / 1", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(((2 * 2) + 1 - 1) / 1))
		})

		It("Should allow basic bigint formulae", func() {
			var out int64
			err := insert("12147483647 * (2)::bigint", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(int64(12147483647 * 2)))
		})

		It("Should allow basic numeric formulae", func() {
			var out float64
			err := insert("1.5 * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(float64(1.5 * 2)))
		})

		It("Should allow $time_in_seconds variable", func() {
			var out int
			err := insert("$time_in_seconds * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(60 * 2))
		})

		It("Should allow $memory_in_mb variable", func() {
			var out int
			err := insert("$memory_in_mb * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(64 * 2))
		})

		It("Should allow power of operator", func() {
			var out float64
			err := insert("2^2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(math.Pow(2, 2)))
		})

		It("Should not allow `;`", func() {
			var out interface{}
			err := insert("1+1;", &out)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp(`illegal token in formula: ;`))
		})

		It("Should not allow `select`", func() {
			var out interface{}
			err := insert("select", &out)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp(`illegal token in formula: select`))
		})

		It("Should not allow `$unknown variable`", func() {
			var out interface{}
			err := insert("$unknown", &out)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(MatchRegexp(`illegal token in formula: \$unknown`))
		})
	})

	Context("pricing_plans", func() {

		It("should ensure unique valid_from + plan_guid", func() {
			t := time.Now()
			guid := uuid.NewV4().String()
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plans (name, valid_from, plan_guid, formula) values (
					$1,
					$2,
					$3,
					'1+1'
				)
			`, "PlanA", t.Format(time.RFC3339), guid)
			Expect(err).ToNot(HaveOccurred())
			_, err = sqlClient.Conn.Exec(`
				insert into pricing_plans (name, valid_from, plan_guid, formula) values (
					$1,
					$2,
					$3,
					'1+1'
				)
			`, "PlanB", t.Format(time.RFC3339), guid)
			Expect(err).To(HaveOccurred())
		})

	})
})

// SelectUsageEvents returns with all the usage events stored in the database
func selectUsageEvents(pc *PostgresClient, tableName string) (*cf.UsageEventList, error) {
	usageEvents := &cf.UsageEventList{}

	rows, queryErr := pc.Conn.Query("SELECT guid, created_at, raw_message FROM " + tableName)
	if queryErr != nil {
		return nil, queryErr
	}
	defer rows.Close()
	for rows.Next() {
		var guid string
		var createdAt string
		var rawMessage json.RawMessage
		if err := rows.Scan(&guid, &createdAt, &rawMessage); err != nil {
			return nil, err
		}
		usageEvents.Resources = append(usageEvents.Resources, cf.UsageEvent{
			MetaData:  cf.MetaData{GUID: guid, CreatedAt: decodePostgresTimestamp(createdAt)},
			EntityRaw: rawMessage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return usageEvents, nil
}

func decodePostgresTimestamp(timestamp string) time.Time {
	t, _ := time.Parse(postgresTimeFormat, timestamp)
	return t
}
