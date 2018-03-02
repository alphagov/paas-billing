package compose_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	composefakes "github.com/alphagov/paas-billing/compose/fakes"
	dbfakes "github.com/alphagov/paas-billing/db/fakes"
	composeapi "github.com/compose/gocomposeapi"

	. "github.com/alphagov/paas-billing/collector/compose"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {
	var (
		logger        lager.Logger
		fakeClient    *composefakes.FakeClient
		fakeSQLCLient *dbfakes.FakeSQLClient
		auditEvents   *[]composeapi.AuditEvent
		eventID1      string
		cursor1       string
	)

	BeforeEach(func() {
		fakeClient = &composefakes.FakeClient{}
		fakeSQLCLient = &dbfakes.FakeSQLClient{}

		logger = lager.NewLogger("test")
		eventID1 = "e3"
		cursor1 = "event-2"

		auditEvents = &[]composeapi.AuditEvent{
			composeapi.AuditEvent{ID: "e1", DeploymentID: "d1", Event: "deployment.scale.members"},
			composeapi.AuditEvent{ID: "e2", DeploymentID: "d1", Event: "deployment.scale.members"},
			composeapi.AuditEvent{ID: "e3", DeploymentID: "d2", Event: "deployment.scale.members"},
			composeapi.AuditEvent{ID: "e4", DeploymentID: "d1", Event: "other"},
		}
	})

	It("should fetch the latest events and insert the billing events into the database", func() {
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 4, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(4))

		By("fetching the cursors")
		Expect(fakeSQLCLient.FetchComposeCursorCallCount()).To(Equal(1))
		Expect(fakeSQLCLient.FetchComposeLatestEventIDCallCount()).To(Equal(1))

		By("fetching the events from the API")
		Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Cursor: "",
			Limit:  4,
		}))

		By("starting a transaction")
		Expect(fakeSQLCLient.BeginTxCallCount()).To(Equal(1))

		By("saving the billing events to the database")
		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(Equal(1))
		passedAuditEvents := fakeSQLCLient.InsertComposeAuditEventsArgsForCall(0)
		Expect(passedAuditEvents).To(Equal([]composeapi.AuditEvent{
			composeapi.AuditEvent{ID: "e1", DeploymentID: "d1", Event: "deployment.scale.members"},
			composeapi.AuditEvent{ID: "e2", DeploymentID: "d1", Event: "deployment.scale.members"},
			composeapi.AuditEvent{ID: "e3", DeploymentID: "d2", Event: "deployment.scale.members"},
		}))

		By("saving the last event id to the database")
		Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(1))
		passedLastID := fakeSQLCLient.InsertComposeLatestEventIDArgsForCall(0)
		Expect(passedLastID).To(Equal("e1"))

		By("saving the cursor to the database")
		Expect(fakeSQLCLient.InsertComposeCursorCallCount()).To(Equal(1))
		passedCursor := fakeSQLCLient.InsertComposeCursorArgsForCall(0)
		Expect(*passedCursor).To(Equal("e4"))

		By("committing the transaction")
		Expect(fakeSQLCLient.CommitCallCount()).To(Equal(1))
	})

	Context("when we encounter the last page", func() {
		It("should set the cursor to nil", func() {
			fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
			fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

			fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
			_, err := fetcher.FetchEvents(logger, 10, time.Minute)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeSQLCLient.InsertComposeCursorCallCount()).To(Equal(1))
			passedCursor := fakeSQLCLient.InsertComposeCursorArgsForCall(0)
			Expect(passedCursor).To(BeNil())
		})
	})

	Context("when cursor is not nil", func() {
		It("should get the next page using the cursor", func() {
			fakeSQLCLient.FetchComposeCursorReturnsOnCall(0, &cursor1, nil)
			fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
			fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

			fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
			_, err := fetcher.FetchEvents(logger, 10, time.Minute)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeClient.GetAuditEventsCallCount()).To(Equal(1))
			params := fakeClient.GetAuditEventsArgsForCall(0)
			Expect(params.Cursor).To(Equal(cursor1))

			Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(0))
		})

		It("should set the cursor to nil when there are no events", func() {
			fakeSQLCLient.FetchComposeCursorReturnsOnCall(0, &cursor1, nil)
			fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{}, nil)
			fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

			fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
			_, err := fetcher.FetchEvents(logger, 10, time.Minute)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeSQLCLient.InsertComposeCursorCallCount()).To(Equal(1))
			passedCursor := fakeSQLCLient.InsertComposeCursorArgsForCall(0)
			Expect(passedCursor).To(BeNil())
		})
	})

	Context("when we encounter the latest_event_id", func() {
		It("should only insert events up to latest_event_id", func() {
			fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
			fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
			fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

			fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
			cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(cnt).To(Equal(2))

			Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(Equal(1))
			passedAuditEvents := fakeSQLCLient.InsertComposeAuditEventsArgsForCall(0)
			Expect(passedAuditEvents).To(Equal([]composeapi.AuditEvent{
				composeapi.AuditEvent{ID: "e1", DeploymentID: "d1", Event: "deployment.scale.members"},
				composeapi.AuditEvent{ID: "e2", DeploymentID: "d1", Event: "deployment.scale.members"},
			}))

			By("setting the cursor to nil")
			Expect(fakeSQLCLient.InsertComposeCursorCallCount()).To(Equal(1))
			passedCursor := fakeSQLCLient.InsertComposeCursorArgsForCall(0)
			Expect(passedCursor).To(BeNil())
		})
	})

	It("should handle if there is no event in the database", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, nil, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		params := fakeClient.GetAuditEventsArgsForCall(0)
		Expect(params.Cursor).To(Equal(""))
	})

	It("should not do anything if there are no new events from the api", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{}, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(0))

		Expect(fakeSQLCLient.BeginTxCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
		Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(0))
	})

	It("should still update the last event id if all events are filtered out", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			composeapi.AuditEvent{ID: "e4", DeploymentID: "d1", Event: "other"},
		}, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(1))

		Expect(fakeSQLCLient.BeginTxCallCount()).To(Equal(1))
		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
		Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(1))
	})

	It("should return error if it can't fetch the last event id", func() {
		dbError := errors.New("some error")
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, nil, dbError)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbError))
		Expect(cnt).To(BeZero())

		Expect(fakeClient.GetAuditEventsCallCount()).To(BeZero())
		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	It("should return error if it can't fetch the cursor", func() {
		dbError := errors.New("some error")
		fakeSQLCLient.FetchComposeCursorReturnsOnCall(0, nil, dbError)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbError))
		Expect(cnt).To(BeZero())

		Expect(fakeClient.GetAuditEventsCallCount()).To(BeZero())
		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	It("should return error if it can't fetch new events", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)

		composeErr := []error{
			errors.New("error 1"),
			errors.New("error 2"),
		}
		fakeClient.GetAuditEventsReturnsOnCall(0, nil, composeErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError("error 1; error 2"))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	It("should return error if it fails to start a transaction", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.BeginTxReturnsOnCall(0, nil, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(0))
	})

	It("should rollback transaction if it can't insert the events into the database", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.InsertComposeAuditEventsReturnsOnCall(0, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

	It("should rollback transaction if it can't insert the last event id to the database", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.InsertComposeLatestEventIDReturnsOnCall(0, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

	It("should rollback transaction if it can't insert the cursor to the database", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.InsertComposeCursorReturnsOnCall(0, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 4, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())

		Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

	It("should return error if it fails to commit transaction", func() {
		fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		fakeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		dbErr := errors.New("some error")
		fakeSQLCLient.CommitReturnsOnCall(0, dbErr)

		fetcher := NewEventFetcher(fakeSQLCLient, fakeClient)
		cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).To(MatchError(dbErr))
		Expect(cnt).To(BeZero())
	})

})
