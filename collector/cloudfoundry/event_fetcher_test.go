package cloudfoundry_test

import (
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	cf "github.com/alphagov/paas-billing/cloudfoundry"
	cffakes "github.com/alphagov/paas-billing/cloudfoundry/fakes"
	"github.com/alphagov/paas-billing/store"

	. "github.com/alphagov/paas-billing/collector/cloudfoundry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
// event3GUID = "0C586A5D-BFB7-4B31-91A2-7A7D06962D50"
// event4GUID = "F2133349-E9D5-47B6-AA0C-00A5AC96B703"
)

var _ = Describe("UsageEvent Fetcher", func() {
	var (
		fetcher     *EventFetcher
		fakeClient  *cffakes.FakeUsageEventsAPI
		eventKind   = "app"
		usageEvent1 = cf.UsageEvent{
			MetaData:  cf.MetaData{GUID: "2C5D3E72-2082-43C1-9262-814EAE7E65AA", CreatedAt: time.Date(2001, 1, 1, 1, 1, 1, 1, time.UTC)},
			EntityRaw: json.RawMessage(`{"field":"value1"}`),
		}
		usageEvent2 = cf.UsageEvent{
			MetaData:  cf.MetaData{GUID: "968437F2-CCEE-4B8E-B29B-34EA701BA196", CreatedAt: time.Date(2002, 2, 2, 2, 2, 2, 2, time.UTC)},
			EntityRaw: json.RawMessage(`{"field":"value2"}`),
		}
		rawEvent1 = store.RawEvent{
			GUID:       usageEvent1.MetaData.GUID,
			Kind:       eventKind,
			CreatedAt:  usageEvent1.MetaData.CreatedAt,
			RawMessage: json.RawMessage(`{"field":"value1"}`),
		}
		rawEvent2 = store.RawEvent{
			GUID:       usageEvent2.MetaData.GUID,
			Kind:       eventKind,
			CreatedAt:  usageEvent2.MetaData.CreatedAt,
			RawMessage: json.RawMessage(`{"field":"value2"}`),
		}
	)

	BeforeEach(func() {
		fakeClient = &cffakes.FakeUsageEventsAPI{}
		fakeClient.TypeReturns(eventKind)
		fetcher = &EventFetcher{
			Logger: lager.NewLogger("test"),
			Client: fakeClient,
		}
	})

	It("should fetch usage events without using after_guid when no lastEvent is set", func() {
		fakeClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				usageEvent1,
				usageEvent2,
			},
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]store.RawEvent{
			rawEvent1,
			rawEvent2,
		}))

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		afterGUID, fetchLimit, minAge := fakeClient.GetArgsForCall(0)
		Expect(afterGUID).To(Equal(cf.GUIDNil))
		Expect(fetchLimit).To(Equal(DefaultFetchLimit))
		Expect(minAge).To(Equal(DefaultRecordMinAge))
	})

	It("should request events using after_guid when when a lastEvent is set", func() {
		fakeClient.GetReturnsOnCall(0, &cf.UsageEventList{
			Resources: []cf.UsageEvent{
				usageEvent2,
			},
		}, nil)

		lastEvent := &store.RawEvent{
			GUID: usageEvent1.MetaData.GUID,
		}
		fetchedEvents, err := fetcher.FetchEvents(lastEvent)
		Expect(fetchedEvents).To(Equal([]store.RawEvent{
			rawEvent2,
		}))

		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(1))

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		afterGUID, fetchLimit, minAge := fakeClient.GetArgsForCall(0)
		Expect(afterGUID).To(Equal(usageEvent1.MetaData.GUID))
		Expect(fetchLimit).To(Equal(DefaultFetchLimit))
		Expect(minAge).To(Equal(DefaultRecordMinAge))

	})

	It("should request FetchLimit number of events", func() {
		fetcher.FetchLimit = 100
		_, err := fetcher.FetchEvents(nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		_, fetchLimit, _ := fakeClient.GetArgsForCall(0)
		Expect(fetchLimit).To(Equal(fetcher.FetchLimit))
	})

	It("should request events with RecordMinAge offset", func() {
		fetcher.RecordMinAge = 15 * time.Minute
		_, err := fetcher.FetchEvents(nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		_, _, minAge := fakeClient.GetArgsForCall(0)
		Expect(minAge).To(Equal(fetcher.RecordMinAge))
	})

	It("should return an error if the lastEvent given has no GUID", func() {
		badLastEvent := &store.RawEvent{}

		_, err := fetcher.FetchEvents(badLastEvent)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("invalid GUID for lastEvent"))
	})

	It("should return error if it can't fetch new events", func() {
		fetchErr := errors.New("some error")
		fakeClient.GetReturnsOnCall(0, nil, fetchErr)

		_, err := fetcher.FetchEvents(nil)
		Expect(err).To(MatchError(fetchErr))
	})

})
