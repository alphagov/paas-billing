package compose_test

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	composefakes "github.com/alphagov/paas-billing/compose/fakes"
	composeapi "github.com/compose/gocomposeapi"

	. "github.com/alphagov/paas-billing/collector/compose"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {

	var (
		testEvent1 = composeapi.AuditEvent{ID: "e1", DeploymentID: "d1", Event: "deployment.scale.members"}
		testEvent2 = composeapi.AuditEvent{ID: "e2", DeploymentID: "d1", Event: "deployment.scale.members"}
		testEvent3 = composeapi.AuditEvent{ID: "e3", DeploymentID: "d2", Event: "deployment.scale.members"}
		testEvent4 = composeapi.AuditEvent{ID: "e4", DeploymentID: "d1", Event: "other"}
	)

	var (
		logger = lager.NewLogger("test")
		limit  = 4
		age    = 1 * time.Minute
	)

	var (
		composeClient *composefakes.FakeClient
		db            *testenv.TempDB
	)

	BeforeEach(func() {
		db = testenv.MustOpen(testenv.BasicConfig)
		composeClient = &composefakes.FakeClient{}
	})

	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	It("should fetch the latest events and insert the billing events into the database", func() {
		fetcher := NewEventFetcher(db.Conn, composeClient)

		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
			testEvent3,
			testEvent4,
		}, nil)
		cnt, err := fetcher.FetchEvents(logger, limit, age)
		Expect(err).ToNot(HaveOccurred())
		Expect(cnt).To(Equal(4))

		By("fetching the events from the API")
		Expect(composeClient.GetAuditEventsCallCount()).To(Equal(1))
		params := composeClient.GetAuditEventsArgsForCall(0)
		Expect(params).To(Equal(composeapi.AuditEventsParams{
			Cursor: "",
			Limit:  4,
		}))

		By("storing only scaling events to the database")
		storedEvents := db.Query(`select * from compose_audit_events`)
		Expect(storedEvents).To(MatchJSON(testenv.Rows{
			{
				"created_at": "0001-01-01T00:00:00+00:00",
				"event_id":   "e1",
				"id":         1,
				"raw_message": json.RawMessage(`{
					"account_id": "",
					"created_at": "0001-01-01T00:00:00Z",
					"data": null,
					"deployment_id": "d1",
					"event": "deployment.scale.members",
					"id": "e1",
					"ip": "",
					"user_agent": "",
					"user_id": ""
				}`),
			},
			{
				"created_at": "0001-01-01T00:00:00+00:00",
				"event_id":   "e2",
				"id":         2,
				"raw_message": json.RawMessage(`{
					"account_id": "",
					"created_at": "0001-01-01T00:00:00Z",
					"data": null,
					"deployment_id": "d1",
					"event": "deployment.scale.members",
					"id": "e2",
					"ip": "",
					"user_agent": "",
					"user_id": ""
				}`),
			},
			{
				"created_at": "0001-01-01T00:00:00+00:00",
				"event_id":   "e3",
				"id":         3,
				"raw_message": json.RawMessage(`{
					"account_id": "",
					"created_at": "0001-01-01T00:00:00Z",
					"data": null,
					"deployment_id": "d2",
					"event": "deployment.scale.members",
					"id": "e3",
					"ip": "",
					"user_agent": "",
					"user_id": ""
				}`),
			},
		}))

		By("storing the last event id to the database")
		storedLastID := db.Get(`SELECT value FROM compose_audit_events_cursor WHERE name = 'latest_event_id'`)
		Expect(storedLastID).To(Equal("e1"))

		By("storing the cursor to the database")
		storedCursor := db.Get(`SELECT value FROM compose_audit_events_cursor WHERE name = 'cursor'`)
		Expect(storedCursor).To(Equal("e4"))
	})

	It("should set the cursor to nil on last page", func() {
		fetcher := NewEventFetcher(db.Conn, composeClient)

		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
			testEvent3,
			testEvent4,
		}, nil)
		_, err := fetcher.FetchEvents(logger, 10, age)
		Expect(err).ToNot(HaveOccurred())

		storedCursor := db.Get(`SELECT value FROM compose_audit_events_cursor WHERE name = 'cursor'`)
		Expect(storedCursor).To(BeNil())
	})

	It("it will rollback insertions of data when an event id already exist", func() {
		fetcher := NewEventFetcher(db.Conn, composeClient)

		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			testEvent1,
		}, nil)
		_, err := fetcher.FetchEvents(logger, 10, age)
		Expect(err).ToNot(HaveOccurred())

		composeClient.GetAuditEventsReturnsOnCall(1, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
		}, nil)
		_, err = fetcher.FetchEvents(logger, 10, age)
		Expect(err).ToNot(HaveOccurred())

		storedEvents := db.Query(`select * from compose_audit_events`)
		Expect(storedEvents).To(MatchJSON(testenv.Rows{
			{
				"created_at": "0001-01-01T00:00:00+00:00",
				"event_id":   "e1",
				"id":         1,
				"raw_message": json.RawMessage(`{
					"account_id": "",
					"created_at": "0001-01-01T00:00:00Z",
					"data": null,
					"deployment_id": "d1",
					"event": "deployment.scale.members",
					"id": "e1",
					"ip": "",
					"user_agent": "",
					"user_id": ""
				}`),
			},
		}))
	})

	It("should get the next page using the cursor when cursor is not nil", func() {
		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
		}, nil)
		fetcher := NewEventFetcher(db.Conn, composeClient)
		_, err := fetcher.FetchEvents(logger, 2, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		composeClient.GetAuditEventsReturnsOnCall(1, &[]composeapi.AuditEvent{
			testEvent3,
		}, nil)
		_, err = fetcher.FetchEvents(logger, 2, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		Expect(composeClient.GetAuditEventsCallCount()).To(Equal(2))
		params := composeClient.GetAuditEventsArgsForCall(1)
		Expect(params.Cursor).To(Equal(testEvent2.ID))
	})

	It("should set the cursor to nil when there are no events", func() {
		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{}, nil)

		fetcher := NewEventFetcher(db.Conn, composeClient)
		_, err := fetcher.FetchEvents(logger, 10, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		storedCursor := db.Get(`SELECT value FROM compose_audit_events_cursor WHERE name = 'cursor'`)
		Expect(storedCursor).To(BeNil())
	})

	It("should only insert events up to latest_event_id", func() {
		composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
		}, nil)
		fetcher := NewEventFetcher(db.Conn, composeClient)
		_, err := fetcher.FetchEvents(logger, 2, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		composeClient.GetAuditEventsReturnsOnCall(1, &[]composeapi.AuditEvent{
			testEvent1,
			testEvent2,
			testEvent3,
			testEvent4,
		}, nil)
		_, err = fetcher.FetchEvents(logger, 2, time.Minute)
		Expect(err).ToNot(HaveOccurred())

		Expect(composeClient.GetAuditEventsCallCount()).To(Equal(2))
		params := composeClient.GetAuditEventsArgsForCall(1)
		Expect(params.Cursor).To(Equal(testEvent2.ID))
		storedCursor := db.Get(`SELECT value FROM compose_audit_events_cursor WHERE name = 'cursor'`)
		Expect(storedCursor).To(BeNil())
	})

	PIt("should handle if there is no event in the database", func() {
		// fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, nil, nil)
		// composeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		// fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// _, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).ToNot(HaveOccurred())

		// params := composeClient.GetAuditEventsArgsForCall(0)
		// Expect(params.Cursor).To(Equal(""))
	})

	PIt("should not do anything if there are no new events from the api", func() {
		// fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		// composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{}, nil)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).ToNot(HaveOccurred())
		// Expect(cnt).To(Equal(0))

		// Expect(fakeSQLCLient.BeginTxCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
		// Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(0))
	})

	PIt("should still update the last event id if all events are filtered out", func() {
		// fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, &eventID1, nil)
		// composeClient.GetAuditEventsReturnsOnCall(0, &[]composeapi.AuditEvent{
		// 	composeapi.AuditEvent{ID: "e4", DeploymentID: "d1", Event: "other"},
		// }, nil)
		// fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).ToNot(HaveOccurred())
		// Expect(cnt).To(Equal(1))

		// Expect(fakeSQLCLient.BeginTxCallCount()).To(Equal(1))
		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
		// Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(1))
	})

	PIt("should return error if it can't fetch the last event id", func() {
		// dbError := errors.New("some error")
		// fakeSQLCLient.FetchComposeLatestEventIDReturnsOnCall(0, nil, dbError)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError(dbError))
		// Expect(cnt).To(BeZero())

		// Expect(composeClient.GetAuditEventsCallCount()).To(BeZero())
		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	PIt("should return error if it can't fetch the cursor", func() {
		// dbError := errors.New("some error")
		// fakeSQLCLient.FetchComposeCursorReturnsOnCall(0, nil, dbError)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError(dbError))
		// Expect(cnt).To(BeZero())

		// Expect(composeClient.GetAuditEventsCallCount()).To(BeZero())
		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	PIt("should return error if it can't fetch new events", func() {
		// composeErr := []error{
		// 	errors.New("error 1"),
		// 	errors.New("error 2"),
		// }
		// composeClient.GetAuditEventsReturnsOnCall(0, nil, composeErr)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError("error 1; error 2"))
		// Expect(cnt).To(BeZero())

		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(BeZero())
	})

	PIt("should return error if it fails to start a transaction", func() {
		// composeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		// dbErr := errors.New("some error")
		// fakeSQLCLient.BeginTxReturnsOnCall(0, nil, dbErr)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError(dbErr))
		// Expect(cnt).To(BeZero())

		// Expect(fakeSQLCLient.InsertComposeAuditEventsCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(0))
	})

	PIt("should rollback transaction if it can't insert the events into the database", func() {
		// composeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		// fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		// dbErr := errors.New("some error")
		// fakeSQLCLient.InsertComposeAuditEventsReturnsOnCall(0, dbErr)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError(dbErr))
		// Expect(cnt).To(BeZero())

		// Expect(fakeSQLCLient.InsertComposeLatestEventIDCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

	PIt("should rollback transaction if it can't insert the last event id to the database", func() {
		// composeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		// fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		// dbErr := errors.New("some error")
		// fakeSQLCLient.InsertComposeLatestEventIDReturnsOnCall(0, dbErr)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 10, time.Minute)
		// Expect(err).To(MatchError(dbErr))
		// Expect(cnt).To(BeZero())

		// Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

	PIt("should rollback transaction if it can't insert the cursor to the database", func() {
		// composeClient.GetAuditEventsReturnsOnCall(0, auditEvents, nil)
		// fakeSQLCLient.BeginTxReturnsOnCall(0, fakeSQLCLient, nil)
		// dbErr := errors.New("some error")
		// fakeSQLCLient.InsertComposeCursorReturnsOnCall(0, dbErr)

		// fetcher := NewEventFetcher(db.Conn, composeClient)
		// cnt, err := fetcher.FetchEvents(logger, 4, time.Minute)
		// Expect(err).To(MatchError(dbErr))
		// Expect(cnt).To(BeZero())

		// Expect(fakeSQLCLient.CommitCallCount()).To(Equal(0))
		// Expect(fakeSQLCLient.RollbackCallCount()).To(Equal(1))
	})

})
