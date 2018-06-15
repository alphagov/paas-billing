package eventserver_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/eventserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BillableEventsHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *fakes.FakeAuthenticator
		fakeAuthorizer    *fakes.FakeAuthorizer
		fakeStore         *fakes.FakeEventStore
		token             = "ACCESS_GRANTED_TOKEN"
		orgGUID1          = "f5f32499-db32-4ab7-a314-20cbe3e49080"
	)

	BeforeEach(func() {
		fakeStore = &fakes.FakeEventStore{}
		fakeAuthenticator = &fakes.FakeAuthenticator{}
		fakeAuthorizer = &fakes.FakeAuthorizer{}
		cfg = Config{
			Authenticator: fakeAuthenticator,
			Logger:        lager.NewLogger("test"),
			Store:         fakeStore,
			EnablePanic:   true,
		}
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		defer cancel()
	})

	It("should return error if no token in request", func() {
		fakeAuthenticator.NewAuthorizerReturns(nil, nil)
		req := httptest.NewRequest(echo.GET, "/billable_events?orgGUID="+orgGUID1, nil)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"error": "no access_token in request"
		}`))
		Expect(res.Code).To(Equal(401))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should return error on authentication error", func() {
		authErr := errors.New("auth-error")
		fakeAuthenticator.NewAuthorizerReturns(nil, authErr)
		req := httptest.NewRequest(echo.GET, "/billable_events?orgGUID="+orgGUID1, nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"error": "auth-error"
		}`))
		Expect(res.Code).To(Equal(401))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should return error on malformed Authorization header", func() {
		fakeAuthenticator.NewAuthorizerReturns(nil, nil)
		req := httptest.NewRequest(echo.GET, "/billable_events?orgGUID="+orgGUID1, nil)
		req.Header.Set("Authorization", token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"error": "invalid Authorization header"
		}`))
		Expect(res.Code).To(Equal(401))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should require admin scope or org billing permissions", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
		fakeAuthorizer.OrganisationsReturns(false, nil)
		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"error": "you need to be billing_manager or an administrator to retreive the billing data"
		}`))
		Expect(res.Code).To(Equal(401))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should fetch BillableEvents from the store when admin", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(true, nil)
		fakeAuthorizer.OrganisationsReturns(false, nil)
		fakeRows := &fakes.FakeBillableEventRows{}
		fakeRows.CloseReturns(nil)
		fakeRows.NextReturnsOnCall(0, true)
		fakeRows.NextReturnsOnCall(1, true)
		fakeRows.NextReturnsOnCall(2, false)
		event1JSON := `{
			"event_guid": "raw-json-guid-1"
		}`
		event2JSON := `{
			"event_guid": "raw-json-guid-2"
		}`
		fakeRows.EventJSONReturnsOnCall(0, []byte(event1JSON), nil)
		fakeRows.EventJSONReturnsOnCall(1, []byte(event2JSON), nil)
		fakeStore.GetBillableEventRowsReturns(fakeRows, nil)

		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetBillableEventRowsCallCount()).To(Equal(1))
		filter := fakeStore.GetBillableEventRowsArgsForCall(0)
		Expect(filter.RangeStart).To(Equal("2001-01-01"))
		Expect(filter.RangeStop).To(Equal("2001-01-02"))
		Expect(filter.OrgGUIDs).To(Equal([]string{orgGUID1}))

		Expect(fakeRows.NextCallCount()).To(Equal(3))
		Expect(fakeRows.EventJSONCallCount()).To(Equal(2))
		Expect(fakeRows.CloseCallCount()).To(Equal(1))

		Expect(res.Body).To(MatchJSON("[" + event1JSON + "," + event2JSON + "]"))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should fetch BillableEvents from the store when manager", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
		fakeAuthorizer.OrganisationsReturns(true, nil)
		fakeRows := &fakes.FakeBillableEventRows{}
		fakeRows.CloseReturns(nil)
		fakeRows.NextReturnsOnCall(0, true)
		fakeRows.NextReturnsOnCall(1, true)
		fakeRows.NextReturnsOnCall(2, false)
		event1JSON := `{
			"event_guid": "raw-json-guid-1"
		}`
		event2JSON := `{
			"event_guid": "raw-json-guid-2"
		}`
		fakeRows.EventJSONReturnsOnCall(0, []byte(event1JSON), nil)
		fakeRows.EventJSONReturnsOnCall(1, []byte(event2JSON), nil)
		fakeStore.GetBillableEventRowsReturns(fakeRows, nil)

		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetBillableEventRowsCallCount()).To(Equal(1))
		filter := fakeStore.GetBillableEventRowsArgsForCall(0)
		Expect(filter.RangeStart).To(Equal("2001-01-01"))
		Expect(filter.RangeStop).To(Equal("2001-01-02"))
		Expect(filter.OrgGUIDs).To(Equal([]string{orgGUID1}))

		Expect(fakeRows.NextCallCount()).To(Equal(3))
		Expect(fakeRows.EventJSONCallCount()).To(Equal(2))
		Expect(fakeRows.CloseCallCount()).To(Equal(1))

		Expect(res.Body).To(MatchJSON("[" + event1JSON + "," + event2JSON + "]"))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should return error if GetBillableEventRows returns error", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(true, nil)
		queryErr := errors.New("query-error")
		fakeStore.GetBillableEventRowsReturns(nil, queryErr)

		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetBillableEventRowsCallCount()).To(Equal(1))

		Expect(res.Body).To(MatchJSON(`{
			"error": "internal server error"
		}`))
		Expect(res.Code).To(Equal(500))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
