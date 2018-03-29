package cloudfoundry_test

import (
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	cf "github.com/alphagov/paas-billing/cloudfoundry"
	cffakes "github.com/alphagov/paas-billing/cloudfoundry/fakes"
	"github.com/alphagov/paas-billing/testenv"

	. "github.com/alphagov/paas-billing/collector/cloudfoundry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {

	var (
		testEvent1 = cf.UsageEvent{
			MetaData: cf.MetaData{
				GUID:      "2c5d3e72-2082-43c1-9262-814eae7e65aa",
				CreatedAt: testenv.Time("2001-01-01T00:00:00+00:00"),
			},
			EntityRaw: json.RawMessage(`{"name":"testEvent1"}`),
		}
		testEvent2 = cf.UsageEvent{
			MetaData: cf.MetaData{
				GUID:      "968437f2-ccee-4b8e-b29b-34ea701ba196",
				CreatedAt: testenv.Time("2002-02-02T00:00:00+00:00"),
			},
			EntityRaw: json.RawMessage(`{"name":"testEvent2"}`),
		}
	)

	var (
		db     *testenv.TempDB
		logger = lager.NewLogger("test")
	)

	BeforeEach(func() {
		var err error
		db, err = testenv.Open(testenv.BasicConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	It("should fetch the latest events and insert into the database", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
				testEvent2,
			},
		}, nil)

		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(2))

		Expect(
			db.Query(`select * from app_usage_events`),
		).To(MatchJSON(testenv.Rows{
			{
				"id":          1,
				"guid":        testEvent1.MetaData.GUID,
				"created_at":  testEvent1.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent1.EntityRaw,
			},
			{
				"id":          2,
				"guid":        testEvent2.MetaData.GUID,
				"created_at":  testEvent2.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent2.EntityRaw,
			},
		}))
	})

	It("should append multiple batches of events into the database", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
			},
		}, nil)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		fakeCFClient.GetReturnsOnCall(1, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent2,
			},
		}, nil)
		_, err = fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		Expect(
			db.Query(`select * from app_usage_events`),
		).To(MatchJSON(testenv.Rows{
			{
				"id":          1,
				"guid":        testEvent1.MetaData.GUID,
				"created_at":  testEvent1.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent1.EntityRaw,
			},
			{
				"id":          2,
				"guid":        testEvent2.MetaData.GUID,
				"created_at":  testEvent2.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent2.EntityRaw,
			},
		}))
	})

	It("should fail to insert duplicate events", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
				testEvent1,
			},
		}, nil)

		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("duplicate key value")))
		Expect(db.Get(`select count(*) from app_usage_events`)).To(BeZero())
	})

	It("should set the last guid", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
			},
		}, nil)

		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		last, err := fetcher.LastGUID()
		Expect(err).ToNot(HaveOccurred())
		Expect(last).To(Equal(testEvent1.MetaData.GUID))
	})

	It("should use a nil afterGUID if no events present", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
			},
		}, nil)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		afterGuid, _, _ := fakeCFClient.GetArgsForCall(0)
		Expect(afterGuid).To(Equal("GUID_NIL"))
	})

	It("should use correct afterGUID for subsequent fetches", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent1,
			},
		}, nil)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		fakeCFClient.GetReturnsOnCall(1, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				testEvent2,
			},
		}, nil)
		_, err = fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		afterGuid, _, _ := fakeCFClient.GetArgsForCall(1)
		Expect(afterGuid).To(Equal(testEvent1.MetaData.GUID))

		Expect(
			db.Query(`select * from app_usage_events`),
		).To(MatchJSON(testenv.Rows{
			{
				"id":          1,
				"guid":        testEvent1.MetaData.GUID,
				"created_at":  testEvent1.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent1.EntityRaw,
			},
			{
				"id":          2,
				"guid":        testEvent2.MetaData.GUID,
				"created_at":  testEvent2.MetaData.CreatedAt.Format(testenv.ISO8601),
				"raw_message": testEvent2.EntityRaw,
			},
		}))
	})

	It("should not insert an empty event list into the database", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{},
		}, nil)

		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(BeZero())

		Expect(db.Get(`select count(*) from app_usage_events`)).To(BeZero())
	})

	It("should return error if it can't fetch the last guid", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		db.Conn.Close()

		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("database is closed")))
		Expect(cnt).To(BeZero())
	})

	It("should return error if it can't fetch new events", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fetchErr := errors.New("some error")
		fakeCFClient.GetReturnsOnCall(0, nil, fetchErr)

		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(fetchErr))
		Expect(cnt).To(BeZero())

		Expect(db.Get(`select count(*) from app_usage_events`)).To(BeZero())
	})

	It("should return error if event has invalid entity json", func() {
		fakeCFClient := &cffakes.FakeUsageEventsAPI{}
		fakeCFClient.TypeReturns("app")

		fetcher := NewEventFetcher(db.Conn, fakeCFClient)

		fakeCFClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				{
					MetaData: cf.MetaData{
						GUID:      "968437f2-ccee-4b8e-b29b-34ea701ba196",
						CreatedAt: testenv.Time("2001-01-01T00:00:00Z"),
					},
					EntityRaw: json.RawMessage(`{"bad-json"}`),
				},
			},
		}, nil)

		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(ContainSubstring("invalid input syntax for type json")))
		Expect(cnt).To(BeZero())
	})

})
