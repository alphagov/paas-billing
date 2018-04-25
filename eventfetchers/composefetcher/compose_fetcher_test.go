package composefetcher_test

import (
	"context"
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/fakes"
	composeapi "github.com/compose/gocomposeapi"

	. "github.com/alphagov/paas-billing/eventfetchers/composefetcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComposeEventFetcher", func() {
	var (
		ctx         = context.Background()
		logger      = lager.NewLogger("test")
		fakeClient  *fakes.FakeComposeClient
		eventKind   = "compose"
		auditEvent1 = composeapi.AuditEvent{ID: "e1", CreatedAt: time.Date(2004, 4, 4, 4, 4, 4, 4, time.UTC), DeploymentID: "d1", Event: "deployment.scale.members"}
		auditEvent2 = composeapi.AuditEvent{ID: "e2", CreatedAt: time.Date(2003, 3, 3, 3, 3, 3, 3, time.UTC), DeploymentID: "d1", Event: "deployment.scale.members"}
		auditEvent3 = composeapi.AuditEvent{ID: "e3", CreatedAt: time.Date(2002, 2, 2, 2, 2, 2, 2, time.UTC), DeploymentID: "d2", Event: "deployment.scale.members"}
		auditEvent4 = composeapi.AuditEvent{ID: "e4", CreatedAt: time.Date(2001, 1, 1, 1, 1, 1, 1, time.UTC), DeploymentID: "d1", Event: "other"}
		rawEvent1   = eventio.RawEvent{
			GUID:       auditEvent1.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent1.CreatedAt,
			RawMessage: toJson(auditEvent1),
		}
		rawEvent2 = eventio.RawEvent{
			GUID:       auditEvent2.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent2.CreatedAt,
			RawMessage: toJson(auditEvent2),
		}
		rawEvent3 = eventio.RawEvent{
			GUID:       auditEvent3.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent3.CreatedAt,
			RawMessage: toJson(auditEvent3),
		}
	)

	BeforeEach(func() {
		fakeClient = &fakes.FakeComposeClient{}
	})

	It("should fetch all events when no lastEvent given", func() {
		fetcher, err := New(Config{
			Logger: logger,
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(ctx, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit: DefaultFetchLimit,
		}))

		Expect(len(fetchedEvents)).To(Equal(3))
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{
			rawEvent3,
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should request events newer than the given lastEvent", func() {
		fetcher, err := New(Config{
			Logger: logger,
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
		}, nil)

		lastEvent := rawEvent3
		fetchedEvents, err := fetcher.FetchEvents(ctx, &lastEvent)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)

		newerThanLastEvent := lastEvent.CreatedAt.Add(TimePrecisionOffset)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit:     DefaultFetchLimit,
			NewerThan: &newerThanLastEvent,
		}))

		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should request in batches of FetchLimit from the client", func() {
		fetcher, err := New(Config{
			Logger:     logger,
			Client:     fakeClient,
			FetchLimit: 1,
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = fetcher.FetchEvents(ctx, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit: 1,
		}))
	})

	It("should only return events up to last known guid when lastEvent is given", func() {
		fetcher, err := New(Config{
			Logger: logger,
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(ctx, &rawEvent3)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should return empty slice if no new events", func() {
		fetcher, err := New(Config{
			Logger: logger,
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())

		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(ctx, &rawEvent1)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(0))
		Expect(fetchedEvents).To(Equal([]eventio.RawEvent{}))
	})

	It("should stop fetching when context is canceled", func() {
		ctx, cancel := context.WithCancel(ctx)

		fetcher, err := New(Config{
			Logger: logger,
			Client: fakeClient,
		})
		Expect(err).ToNot(HaveOccurred())
		fakeClient.GetAuditEventsReturns(&[]composeapi.AuditEvent{}, nil)

		errs := make(chan error)
		go func() {
			for {
				_, err := fetcher.FetchEvents(ctx, &rawEvent1)
				if err != nil {
					errs <- err
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		Eventually(errs).Should(Receive(MatchError("interupted by context cancelation")))
		Expect(fakeClient.GetAuditEventsCallCount()).To(BeNumerically(">", 1))
	})

})

func toJson(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}
