package db_test

import (
	"encoding/json"
	"time"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	. "github.com/alphagov/paas-usage-events-collector/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	postgresTimeFormat = "2006-01-02T15:04:05.000000Z"
	dbName             = "usage_events"
)

var _ = Describe("Db", func() {

	var sqlClient PostgresClient

	BeforeEach(func() {
		sqlClient = NewPostgresClient("postgres://postgres:@localhost:5432/?sslmode=disable")
		createDB(sqlClient)
	})

	AfterEach(func() {
		dropDB(sqlClient)
	})

	TestUsageEvents := func(tableName string) {
		var (
			testData   *cf.UsageEventList
			event1GUID string
			event2GUID string
			sampleData json.RawMessage
		)

		BeforeEach(func() {
			sqlClient.InitSchema()

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

		AfterEach(func() {
			dropUsageEventsTable(sqlClient, tableName)
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
})

// SelectUsageEvents returns with all the usage events stored in the database
func selectUsageEvents(pc PostgresClient, tableName string) (*cf.UsageEventList, error) {
	usageEvents := &cf.UsageEventList{}

	db, openErr := pc.Open()
	if openErr != nil {
		return nil, openErr
	}
	defer db.Close()

	rows, queryErr := db.Query("SELECT guid, created_at, raw_message FROM " + tableName)
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

func createDB(sqlClient PostgresClient) {
	db, openErr := sqlClient.Open()
	defer db.Close()
	Expect(openErr).ToNot(HaveOccurred())

	_, execErr := db.Exec("CREATE DATABASE " + dbName)
	Expect(execErr).ToNot(HaveOccurred())
}

func dropUsageEventsTable(sqlClient PostgresClient, tableName string) {
	db, openErr := sqlClient.Open()
	defer db.Close()
	Expect(openErr).ToNot(HaveOccurred())

	_, execErr := db.Exec("DROP TABLE " + tableName + " CASCADE")
	Expect(execErr).ToNot(HaveOccurred())
}

func dropDB(sqlClient PostgresClient) {
	db, openErr := sqlClient.Open()
	defer db.Close()
	Expect(openErr).ToNot(HaveOccurred())

	_, execErr := db.Exec("DROP DATABASE " + dbName)
	Expect(execErr).ToNot(HaveOccurred())
}
