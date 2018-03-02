package cloudfoundry_test

import (
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	cf "github.com/alphagov/paas-billing/cloudfoundry"
	cffakes "github.com/alphagov/paas-billing/cloudfoundry/fakes"
	"github.com/alphagov/paas-billing/db"
	dbfakes "github.com/alphagov/paas-billing/db/fakes"

	. "github.com/alphagov/paas-billing/collector/cloudfoundry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	event1GUID = "2C5D3E72-2082-43C1-9262-814EAE7E65AA"
	event2GUID = "968437F2-CCEE-4B8E-B29B-34EA701BA196"
	event3GUID = "0C586A5D-BFB7-4B31-91A2-7A7D06962D50"
	event4GUID = "F2133349-E9D5-47B6-AA0C-00A5AC96B703"
)

var _ = Describe("Collector", func() {
	var (
		logger              lager.Logger
		fakeClient          *cffakes.FakeUsageEventsAPI
		fakeSQLCLient       *dbfakes.FakeSQLClient
		emptyUsageEvents    *cf.UsageEventList
		nonEmptyUsageEvents *cf.UsageEventList
	)

	BeforeEach(func() {
		fakeClient = &cffakes.FakeUsageEventsAPI{}
		fakeClient.TypeReturns("app")

		fakeSQLCLient = &dbfakes.FakeSQLClient{}

		logger = lager.NewLogger("test")

		emptyUsageEvents = &cf.UsageEventList{Resources: []cf.UsageEvent{}}
		nonEmptyUsageEvents = &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				{
					MetaData:  cf.MetaData{GUID: event1GUID},
					EntityRaw: json.RawMessage(`{"field":"value1"}`),
				},
				{
					MetaData:  cf.MetaData{GUID: event2GUID},
					EntityRaw: json.RawMessage(`{"field":"value2"}`),
				},
			},
		}
	})

	It("should fetch the latest events and insert into the database", func() {
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, "LAST-GUID", nil)
		fakeClient.GetReturnsOnCall(0, nonEmptyUsageEvents, nil)
		fakeSQLCLient.InsertUsageEventListReturnsOnCall(0, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(2))

		Expect(fakeSQLCLient.FetchLastGUIDCallCount()).To(Equal(1))
		tableName := fakeSQLCLient.FetchLastGUIDArgsForCall(0)
		Expect(tableName).To(Equal(db.AppUsageTableName))

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		afterGUID, count, minAge := fakeClient.GetArgsForCall(0)
		Expect(afterGUID).To(Equal("LAST-GUID"))
		Expect(count).To(Equal(10))
		Expect(minAge).To(Equal(time.Minute))

		Expect(fakeSQLCLient.InsertUsageEventListCallCount()).To(Equal(1))
		usageEvents, tableName := fakeSQLCLient.InsertUsageEventListArgsForCall(0)
		Expect(usageEvents).To(Equal(nonEmptyUsageEvents))
		Expect(tableName).To(Equal(db.AppUsageTableName))
	})

	It("should handle if there is no last guid in the database", func() {
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, cf.GUIDNil, nil)
		fakeClient.GetReturnsOnCall(0, nonEmptyUsageEvents, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		afterGUID, _, _ := fakeClient.GetArgsForCall(0)
		Expect(afterGUID).To(Equal(cf.GUIDNil))
	})

	It("should not insert an empty event list into the database", func() {
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, cf.GUIDNil, nil)
		fakeClient.GetReturnsOnCall(0, emptyUsageEvents, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.InsertUsageEventListCallCount()).To(BeZero())
	})

	It("should return error if it can't fetch the last guid", func() {
		guidErr := errors.New("some error")
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, "", guidErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(guidErr))
		Expect(cnt).To(BeZero())

		Expect(fakeClient.GetCallCount()).To(BeZero())
		Expect(fakeSQLCLient.InsertUsageEventListCallCount()).To(BeZero())
	})

	It("should return error if it can't fetch new events", func() {
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, cf.GUIDNil, nil)
		fetchErr := errors.New("some error")
		fakeClient.GetReturnsOnCall(0, nil, fetchErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(fetchErr))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.InsertUsageEventListCallCount()).To(BeZero())
	})

	It("should return error if it can't insert the events into the database", func() {
		fakeSQLCLient.FetchLastGUIDReturnsOnCall(0, cf.GUIDNil, nil)
		fakeClient.GetReturnsOnCall(0, nonEmptyUsageEvents, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.InsertUsageEventListReturnsOnCall(0, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())
	})

})
