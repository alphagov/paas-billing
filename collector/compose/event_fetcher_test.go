package compose_test

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	composefakes "github.com/alphagov/paas-billing/compose/fakes"
	"github.com/alphagov/paas-billing/store"
	composeapi "github.com/compose/gocomposeapi"

	. "github.com/alphagov/paas-billing/collector/compose"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {
	var (
		logger     = lager.NewLogger("test")
		fakeClient *composefakes.FakeClient
		fetcher    *EventFetcher
		eventKind  = "compose"
		// eventID1    string
		// cursor1     string
		auditEvent1 = composeapi.AuditEvent{ID: "e1", CreatedAt: time.Date(2004, 4, 4, 4, 4, 4, 4, time.UTC), DeploymentID: "d1", Event: "deployment.scale.members"}
		auditEvent2 = composeapi.AuditEvent{ID: "e2", CreatedAt: time.Date(2003, 3, 3, 3, 3, 3, 3, time.UTC), DeploymentID: "d1", Event: "deployment.scale.members"}
		auditEvent3 = composeapi.AuditEvent{ID: "e3", CreatedAt: time.Date(2002, 2, 2, 2, 2, 2, 2, time.UTC), DeploymentID: "d2", Event: "deployment.scale.members"}
		auditEvent4 = composeapi.AuditEvent{ID: "e4", CreatedAt: time.Date(2001, 1, 1, 1, 1, 1, 1, time.UTC), DeploymentID: "d1", Event: "other"}
		rawEvent1   = store.RawEvent{
			GUID:       auditEvent1.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent1.CreatedAt,
			RawMessage: toJson(auditEvent1),
		}
		rawEvent2 = store.RawEvent{
			GUID:       auditEvent2.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent2.CreatedAt,
			RawMessage: toJson(auditEvent2),
		}
		rawEvent3 = store.RawEvent{
			GUID:       auditEvent3.ID,
			Kind:       eventKind,
			CreatedAt:  auditEvent3.CreatedAt,
			RawMessage: toJson(auditEvent3),
		}
	)

	BeforeEach(func() {
		fakeClient = &composefakes.FakeClient{}

		// eventID1 = "e3"
		// cursor1 = "event-2"

		fetcher = &EventFetcher{
			Logger:  logger,
			Compose: fakeClient,
		}
	})

	It("should fetch all events when no lastEvent given", func() {
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit: DefaultFetchLimit,
		}))

		Expect(len(fetchedEvents)).To(Equal(3))
		Expect(fetchedEvents).To(Equal([]store.RawEvent{
			rawEvent3,
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should request events newer than the given lastEvent", func() {
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
		}, nil)

		lastEvent := rawEvent3
		fetchedEvents, err := fetcher.FetchEvents(&lastEvent)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)

		newerThanLastEvent := lastEvent.CreatedAt.Add(TimePrecisionOffset)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit:     DefaultFetchLimit,
			NewerThan: &newerThanLastEvent,
		}))

		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]store.RawEvent{
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should request in batches of FetchLimit from the client", func() {
		fetcher.FetchLimit = 1
		_, err := fetcher.FetchEvents(nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Limit: 1,
		}))
	})

	It("should only return events up to last known guid when lastEvent is given", func() {
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(&rawEvent3)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(2))
		Expect(fetchedEvents).To(Equal([]store.RawEvent{
			rawEvent2,
			rawEvent1,
		}))
	})

	It("should return empty slice if no new events", func() {
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			auditEvent1,
			auditEvent2,
			auditEvent3,
			auditEvent4,
		}, nil)

		fetchedEvents, err := fetcher.FetchEvents(&rawEvent1)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fetchedEvents)).To(Equal(0))
		Expect(fetchedEvents).To(Equal([]store.RawEvent{}))
	})

})

func toJson(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}
