package cffetcher_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/fakes"

	. "github.com/alphagov/paas-billing/eventfetchers/cffetcher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UsageEvent Fetcher", func() {
	var (
		ctx         = context.Background()
		fakeClient  *fakes.FakeUsageEventsAPI
		eventKind   = "app"
		usageEvent1 = UsageEvent{
			MetaData:  MetaData{GUID: "2C5D3E72-2082-43C1-9262-814EAE7E65AA", CreatedAt: time.Date(2001, 1, 1, 1, 1, 1, 1, time.UTC)},
			EntityRaw: json.RawMessage(`{"field":"value1"}`),
		}
		usageEvent2 = UsageEvent{
			MetaData:  MetaData{GUID: "968437F2-CCEE-4B8E-B29B-34EA701BA196", CreatedAt: time.Date(2002, 2, 2, 2, 2, 2, 2, time.UTC)},
			EntityRaw: json.RawMessage(`{"field":"value2"}`),
		}
		rawEvent1 = eventio.RawEvent{
			GUID:       usageEvent1.MetaData.GUID,
			Kind:       eventKind,
			CreatedAt:  usageEvent1.MetaData.CreatedAt,
			RawMessage: json.RawMessage(`{"field":"value1"}`),
		}
		rawEvent2 = eventio.RawEvent{
			GUID:       usageEvent2.MetaData.GUID,
			Kind:       eventKind,
			CreatedAt:  usageEvent2.MetaData.CreatedAt,
			RawMessage: json.RawMessage(`{"field":"value2"}`),
		}
	)

	BeforeEach(func() {
		fakeClient = &fakes.FakeUsageEventsAPI{}
		fakeClient.TypeReturns(eventKind)
	})

	It("should fetch usage events without using after_guid when no lastEvent is set", func() {
		fakeClient.GetReturnsOnCall(0, &UsageEventList{
			Resources: []UsageEvent{
				usageEvent1,
				usageEvent2,
			},
		}, nil)

		fetcher, err := New(Config{
			Logger: lager.NewLogger("test"),
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fetchedEvents, err := fetcher.FetchEvents(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{
			rawEvent1,
			rawEvent2,
		}))

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		afterGUID, fetchLimit, minAge := fakeClient.GetArgsForCall(0)
		Expect(afterGUID).To(Equal(GUIDNil))
		Expect(fetchLimit).To(Equal(DefaultFetchLimit))
		Expect(minAge).To(Equal(DefaultRecordMinAge))
	})

	It("should request events using after_guid when when a lastEvent is set", func() {
		fakeClient.GetReturnsOnCall(0, &UsageEventList{
			Resources: []UsageEvent{
				usageEvent2,
			},
		}, nil)

		lastEvent := &eventio.RawEvent{
			GUID: usageEvent1.MetaData.GUID,
		}

		fetcher, err := New(Config{
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fetchedEvents, err := fetcher.FetchEvents(ctx, lastEvent)
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{
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
		fetcher, err := New(Config{
			Client:     fakeClient,
			FetchLimit: 100,
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = fetcher.FetchEvents(ctx, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		_, fetchLimit, _ := fakeClient.GetArgsForCall(0)
		Expect(fetchLimit).To(Equal(100))
	})

	It("should request events with RecordMinAge offset", func() {
		fetcher, err := New(Config{
			Client:       fakeClient,
			RecordMinAge: 15 * time.Minute,
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = fetcher.FetchEvents(ctx, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetCallCount()).To(Equal(1))
		_, _, minAge := fakeClient.GetArgsForCall(0)
		Expect(minAge).To(Equal(15 * time.Minute))
	})

	It("should return an error if the lastEvent given has no GUID", func() {
		badLastEvent := &eventio.RawEvent{}

		fetcher, err := New(Config{
			Client:       fakeClient,
			RecordMinAge: 15 * time.Minute,
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = fetcher.FetchEvents(ctx, badLastEvent)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("invalid GUID for lastEvent"))
	})

	It("should return error if it can't fetch new events", func() {
		fetchErr := errors.New("some error")
		fakeClient.GetReturnsOnCall(0, nil, fetchErr)

		fetcher, err := New(Config{
			Client:       fakeClient,
			RecordMinAge: 15 * time.Minute,
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = fetcher.FetchEvents(ctx, nil)
		Expect(err).To(MatchError(fetchErr))
	})

})
