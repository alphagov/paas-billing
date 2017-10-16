package db_test

import (
	"database/sql"
	"encoding/json"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	. "github.com/alphagov/paas-usage-events-collector/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	DBName = "usage_events"
)

var _ = Describe("Db", func() {

	var sqlClient SQLClient

	BeforeEach(func() {
		sqlClient = NewSQLClient()
		createDB(sqlClient)
	})

	AfterEach(func() {
		dropDB(sqlClient)
	})

	TestUsageEvents := func(tableName string) {
		var (
			testData   cf.UsageEventList
			event1GUID string
			event2GUID string
			sampleData json.RawMessage
		)

		BeforeEach(func() {
			createAppEventsTable(sqlClient, tableName)

			event1GUID = "2C5D3E72-2082-43C1-9262-814EAE7E65AA"
			event2GUID = "968437F2-CCEE-4B8E-B29B-34EA701BA196"
			sampleData = json.RawMessage(`{"field":"value"}`)
			testData = cf.UsageEventList{
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
			dropAppEventsTable(sqlClient, tableName)
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
				usageEvents, err := sqlClient.SelectUsageEvents(tableName)
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

				var dataThatFailsToInsert cf.UsageEventList

				BeforeEach(func() {
					dataThatFailsToInsert = cf.UsageEventList{
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
					usageEvents, err := sqlClient.SelectUsageEvents(tableName)
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

func createDB(sqlClient SQLClient) {
	db, err := sql.Open("postgres", sqlClient.ConnectionString)
	defer db.Close()
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE DATABASE " + DBName)
	Expect(err).ToNot(HaveOccurred())
}

func createAppEventsTable(sqlClient SQLClient, tableName string) {
	db, err := sql.Open("postgres", sqlClient.ConnectionString)
	defer db.Close()
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE TABLE " + tableName + " (id SERIAL, guid CHAR(36) UNIQUE, created_at timestamp, raw_message jsonb)")
	Expect(err).ToNot(HaveOccurred())
}

func dropAppEventsTable(sqlClient SQLClient, tableName string) {
	db, err := sql.Open("postgres", sqlClient.ConnectionString)
	defer db.Close()
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("DROP TABLE " + tableName + " CASCADE")
	Expect(err).ToNot(HaveOccurred())
}

func dropDB(sqlClient SQLClient) {
	db, err := sql.Open("postgres", sqlClient.ConnectionString)
	defer db.Close()
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("DROP DATABASE " + DBName)
	Expect(err).ToNot(HaveOccurred())
}
