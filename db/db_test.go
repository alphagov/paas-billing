package db_test

import (
	"encoding/json"
	"math"
	"time"

	cf "github.com/alphagov/paas-billing/cloudfoundry"
	. "github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"
	composeapi "github.com/compose/gocomposeapi"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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

	Describe("Compose audit events", func() {
		var composeAuditEvents []composeapi.AuditEvent

		t1, _ := time.Parse(time.RFC3339Nano, "2018-01-01T16:51:14.148Z")
		t2, _ := time.Parse(time.RFC3339Nano, "2018-01-02T16:51:14.148Z")
		BeforeEach(func() {
			composeAuditEvents = []composeapi.AuditEvent{
				composeapi.AuditEvent{
					ID:           "a30a8030345702347b2ebc9b",
					DeploymentID: "d1",
					Event:        "e1",
					CreatedAt:    t1,
				},
				composeapi.AuditEvent{
					ID:           "0823a3089f13171fc0579715",
					DeploymentID: "d2",
					Event:        "e2",
					CreatedAt:    t2,
				},
			}
		})

		Context("given no previous events table", func() {
			It("returns an empty last event id", func() {
				latestEventID, err := sqlClient.FetchComposeLatestEventID()
				Expect(err).ToNot(HaveOccurred())
				Expect(latestEventID).To(BeNil())
			})

			It("returns an empty cursor", func() {
				cursor, err := sqlClient.FetchComposeCursor()
				Expect(err).ToNot(HaveOccurred())
				Expect(cursor).To(BeNil())
			})

			It("can store and retrieve the last event id", func() {
				eventID := "0f28d542e94dc76dc10b75ee"
				err := sqlClient.InsertComposeLatestEventID(eventID)
				Expect(err).ToNot(HaveOccurred())

				res, err := sqlClient.FetchComposeLatestEventID()
				Expect(err).ToNot(HaveOccurred())
				Expect(*res).To(Equal(eventID))
			})

			It("can store and retrieve the cursor", func() {
				cursor := "0f28d542e94dc76dc10b75ee"
				err := sqlClient.InsertComposeCursor(&cursor)
				Expect(err).ToNot(HaveOccurred())

				res, err := sqlClient.FetchComposeCursor()
				Expect(err).ToNot(HaveOccurred())
				Expect(*res).To(Equal(cursor))
			})

			It("can set cursor to nil", func() {
				err := sqlClient.InsertComposeCursor(nil)
				Expect(err).ToNot(HaveOccurred())

				res, err := sqlClient.FetchComposeCursor()
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(BeNil())
			})

			It("can store events", func() {
				By("inserting data")
				err := sqlClient.InsertComposeAuditEvents(composeAuditEvents)
				Expect(err).ToNot(HaveOccurred())

				By("reading it back in the correct order")
				auditEvents, err := selectComposeAuditEvents(sqlClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(auditEvents).To(Equal([]map[string]interface{}{
					map[string]interface{}{
						"id":          1,
						"event_id":    "a30a8030345702347b2ebc9b",
						"created_at":  "2018-01-01T16:51:14.148Z",
						"raw_message": `{"id": "a30a8030345702347b2ebc9b", "ip": "", "data": null, "event": "e1", "_links": {"alerts": {"href": "", "templated": false}, "backups": {"href": "", "templated": false}, "cluster": {"href": "", "templated": false}, "scalings": {"href": "", "templated": false}, "portal_users": {"href": "", "templated": false}, "compose_web_ui": {"href": "", "templated": false}}, "user_id": "", "account_id": "", "created_at": "2018-01-01T16:51:14.148Z", "user_agent": "", "deployment_id": "d1"}`,
					},
					map[string]interface{}{
						"id":          2,
						"event_id":    "0823a3089f13171fc0579715",
						"created_at":  "2018-01-02T16:51:14.148Z",
						"raw_message": `{"id": "0823a3089f13171fc0579715", "ip": "", "data": null, "event": "e2", "_links": {"alerts": {"href": "", "templated": false}, "backups": {"href": "", "templated": false}, "cluster": {"href": "", "templated": false}, "scalings": {"href": "", "templated": false}, "portal_users": {"href": "", "templated": false}, "compose_web_ui": {"href": "", "templated": false}}, "user_id": "", "account_id": "", "created_at": "2018-01-02T16:51:14.148Z", "user_agent": "", "deployment_id": "d2"}`,
					},
				}))
			})
		})

		Context("given previous audit events", func() {

			BeforeEach(func() {
				err := sqlClient.InsertComposeAuditEvents(composeAuditEvents)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when event id already exist in the table", func() {
				It("it will rollback insertions of data", func() {
					// the unique id constraint
					err := sqlClient.InsertComposeAuditEvents([]composeapi.AuditEvent{
						composeapi.AuditEvent{ID: "a30a8030345702347b2ebc9b"},
					})
					Expect(err).To(HaveOccurred())
					existingEvents, err := selectComposeAuditEvents(sqlClient)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(existingEvents)).To(Equal(2))
				})
			})

		})

	})

	Describe("App usage events", func() {
		TestUsageEvents(AppUsageTableName)
	})

	Describe("Service usage events", func() {
		TestUsageEvents(ServiceUsageTableName)
	})

	Describe("transactions", func() {
		Context("in a transaction", func() {
			var txDB SQLClient

			BeforeEach(func() {
				var err error
				txDB, err = sqlClient.BeginTx()
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(txDB.Rollback()).To(Succeed())
			})

			It("should error if trying to create a new transaction", func() {
				txDB2, err := txDB.BeginTx()
				Expect(err).To(MatchError("cannot create a transaction within a transaction"))
				Expect(txDB2).To(BeNil())
			})
		})

		Context("not in a transaction", func() {
			It("should error if calling commit", func() {
				Expect(sqlClient.Commit()).To(MatchError("cannot commit unless in a transaction"))
			})

			It("should error if calling rollback", func() {
				Expect(sqlClient.Rollback()).To(MatchError("cannot rollback unless in a transaction"))
			})
		})
	})

	Describe("Pricing Formulae", func() {

		var insert = func(formula string, out interface{}) error {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plans(name, valid_from, plan_guid) values (
					'FormulaTestPlan',
					'2000-01-01',
					$1
				);
			`, uuid.NewV4().String())
			if err != nil {
				return err
			}

			return sqlClient.Conn.QueryRow(`
				insert into pricing_plan_components(pricing_plan_id, name, formula, vat_rate_id, currency) values (
					1,
					'FormulaTestPlan/1',
					$1,
					1,
					'GBP'
				) returning eval_formula(64, 128, 2, tstzrange(now(), now() + '60 seconds'), formula) as result
			`, formula).Scan(out)
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

		It("Should not truncate the result of a division of $time_in_seconds", func() {
			var out float64
			err := insert("$time_in_seconds / 3600 * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(60 * 2 / 3600.0))
		})

		It("Should allow $memory_in_mb variable", func() {
			var out int
			err := insert("$memory_in_mb * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(64 * 2))
		})

		It("Should not truncate the result of a division of $memory_in_mb", func() {
			var out float64
			err := insert("$memory_in_mb / 1024 * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(64 / 1024.0 * 2))
		})

		It("Should allow $storage_in_mb variable", func() {
			var out int
			err := insert("$storage_in_mb * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(128 * 2))
		})

		It("Should not truncate the result of a division of $storage_in_mb", func() {
			var out float64
			err := insert("$storage_in_mb / 1024 * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(128 / 1024.0 * 2))
		})

		It("Should allow $number_of_nodes variable", func() {
			var out int
			err := insert("$number_of_nodes * 2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(2 * 2))
		})

		It("Should allow power of operator", func() {
			var out float64
			err := insert("2^2", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(math.Pow(2, 2)))
		})

		It("Should allow ceil function", func() {
			var out float64
			err := insert("ceil(5.0/3.0)", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeNumerically("==", 2))
		})

		It("Should allow ceil function with a variable", func() {
			var out float64
			err := insert("ceil($time_in_seconds / 3600) * 10", &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeNumerically("==", 10))
		})

		It("Should throw error if ceil is used wrongly", func() {
			var out float64
			err := insert("ceil(5", &out)
			Expect(err).To(HaveOccurred())
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
			timeFirstOfMonth := "2017-04-01T00:00:00Z"
			guid := uuid.NewV4().String()
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plans (name, valid_from, plan_guid) values (
					$1,
					$2,
					$3
				)
			`, "PlanA", timeFirstOfMonth, guid)
			Expect(err).ToNot(HaveOccurred())
			_, err = sqlClient.Conn.Exec(`
				insert into pricing_plans (name, valid_from, plan_guid) values (
					$1,
					$2,
					$3
				)
			`, "PlanB", timeFirstOfMonth, guid)
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("reject placing plans with valid_from that isn't the start of the month",
			func(timestamp string) {
				guid := uuid.NewV4().String()
				_, err := sqlClient.Conn.Exec(`
					insert into pricing_plans (name, valid_from, plan_guid) values (
						$1,
						$2,
						$3
					)
				`, "PlanC", timestamp, guid)
				Expect(err).To(MatchError(`pq: new row for relation "pricing_plans" violates check constraint "valid_from_start_of_month"`))
			},
			Entry("not first day of month", "2017-04-04T00:00:00Z"),
			Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
			Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
			Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
			Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
		)

	})

	Context("pricing_plan_components", func() {

		BeforeEach(func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plans (id, name, valid_from, plan_guid) values (
					1,
					'PlanA',
					$1,
					'GUID'
				)
				`, "2017-12-01T00:00:00Z")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ensure I can insert a valid record", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1+1", 1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ensure the pricing_plan_id belongs to an existing plan", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 2, "PlanB 1", "1+1", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
		})

		It("should ensure the vat_rate_id belongs to an existing vat_rate", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1+1", 999)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
		})

		It("should ensure the currency belongs to a valid currency code", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'ISK'
				)
			`, 1, "PlanA 1", "1+1", 999)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
		})

		It("should ensure name is not empty", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "", "1+1", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
		})

		It("should ensure formula is not empty", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("formula can not be empty"))
		})

		It("should ensure formula is valid", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1 + foo", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("illegal token in formula"))
		})

		It("should ensure unique pricing_plan_id + name", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1+1", 1)
			Expect(err).ToNot(HaveOccurred())

			_, err = sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1+2", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate key"))
		})

	})

	Context("vat_rates", func() {
		It("should ensure I can insert a valid record", func() {
			_, err := sqlClient.Conn.Exec(`insert into vat_rates (name, rate) values ('test', 0.25)`)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ensure name is not empty", func() {
			_, err := sqlClient.Conn.Exec(`insert into vat_rates (name, rate) values ('', 0.25)`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
		})

		It("should ensure rate is non-negative", func() {
			_, err := sqlClient.Conn.Exec(`insert into vat_rates (name, rate) values ('test', -0.1)`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
		})

		It("should ensure I can't delete a referenced vat_rates record", func() {
			_, err := sqlClient.Conn.Exec(`
				insert into pricing_plans (id, name, valid_from, plan_guid) values (
					1,
					'PlanA',
					$1,
					'GUID'
				)
				`, "2017-12-01T00:00:00Z")
			Expect(err).ToNot(HaveOccurred())

			_, err = sqlClient.Conn.Exec(`
				insert into pricing_plan_components (pricing_plan_id, name, formula, vat_rate_id, currency) values (
					$1,
					$2,
					$3,
					$4,
					'GBP'
				)
			`, 1, "PlanA 1", "1+1", 1)
			Expect(err).ToNot(HaveOccurred())

			_, err = sqlClient.Conn.Exec(`delete from vat_rates WHERE id = 1`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
		})
	})

	Context("currency_rates", func() {
		It("should ensure I can insert a valid currencies", func() {
			var err error
			_, err = sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('GBP', '2000-01-01T00:00:00', 0.25)`)
			Expect(err).ToNot(HaveOccurred())
			_, err = sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('USD', '2000-01-01T00:00:00', 0.25)`)
			Expect(err).ToNot(HaveOccurred())
			_, err = sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('EUR', '2000-01-01T00:00:00', 0.25)`)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ensure code is not an invalid currency", func() {
			var err error
			_, err = sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('ISK', '2000-01-01T00:00:00', 0.25)`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
			_, err = sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('', '2000-01-01T00:00:00', 0.25)`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
		})

		It("should ensure rate is non-negative", func() {
			_, err := sqlClient.Conn.Exec(`insert into currency_rates (code, valid_from, rate) values ('USD', '2000-01-01T00:00:00', -0.25)`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("violates check constraint"))
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

func selectComposeAuditEvents(pc *PostgresClient) ([]map[string]interface{}, error) {
	rows, queryErr := pc.Conn.Query("SELECT id, event_id, created_at, raw_message FROM compose_audit_events")
	if queryErr != nil {
		return nil, queryErr
	}
	res := make([]map[string]interface{}, 0)
	defer rows.Close()
	for rows.Next() {
		var id int
		var eventId string
		var createdAt string
		var rawMessage json.RawMessage
		if err := rows.Scan(&id, &eventId, &createdAt, &rawMessage); err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}{"id": id, "event_id": eventId, "created_at": createdAt, "raw_message": string(rawMessage)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func decodePostgresTimestamp(timestamp string) time.Time {
	t, _ := time.Parse(postgresTimeFormat, timestamp)
	return t
}
