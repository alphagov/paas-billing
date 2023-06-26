package apiserver_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"

	"github.com/alphagov/paas-billing/apiserver/auth/authfakes"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"

	"code.cloudfoundry.org/lager"

	"github.com/labstack/echo/v4"

	. "github.com/alphagov/paas-billing/apiserver"
	"github.com/alphagov/paas-billing/eventio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BillableEventsHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *authfakes.FakeAuthenticator
		fakeAuthorizer    *authfakes.FakeAuthorizer
		fakeStore         *eventiofakes.FakeEventStore
		token             = "ACCESS_GRANTED_TOKEN"
		orgGUID1          = "f5f32499-db32-4ab7-a314-20cbe3e49080"
	)

	BeforeEach(func(specContext SpecContext) {
		fakeStore = &eventiofakes.FakeEventStore{}
		fakeAuthenticator = &authfakes.FakeAuthenticator{}
		fakeAuthorizer = &authfakes.FakeAuthorizer{}
		cfg = Config{
			Authenticator: fakeAuthenticator,
			Logger:        lager.NewLogger("test"),
			Store:         fakeStore,
			EnablePanic:   true,
		}
		ctx, cancel = context.WithCancel(specContext)
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
		fakeAuthorizer.HasBillingAccessReturns(false, nil)
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
			"error": "you need to be billing_manager or an administrator to retrieve the billing data"
		}`))
		Expect(res.Code).To(Equal(401))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should fetch BillableEvents from the store when admin", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(true, nil)
		fakeAuthorizer.HasBillingAccessReturns(false, nil)
		fakeRows := &eventiofakes.FakeBillableEventRows{}
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
		_, filter := fakeStore.GetBillableEventRowsArgsForCall(0)
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
		fakeAuthorizer.HasBillingAccessReturns(true, nil)
		fakeRows := &eventiofakes.FakeBillableEventRows{}
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
		Expect(fakeStore.GetConsolidatedBillableEventRowsCallCount()).To(Equal(0))
		_, filter := fakeStore.GetBillableEventRowsArgsForCall(0)
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

	It("should fetch ConsolidatedBillableEvents for whole months which have been consolidated and billable events otherwise", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
		fakeAuthorizer.HasBillingAccessReturns(true, nil)
		fakeRows := &eventiofakes.FakeBillableEventRows{}
		fakeRows.CloseReturns(nil)
		fakeRows.NextReturns(false)
		fakeStore.IsRangeConsolidatedReturnsOnCall(0, false, nil)
		fakeStore.IsRangeConsolidatedReturnsOnCall(1, true, nil)
		fakeStore.IsRangeConsolidatedReturnsOnCall(2, true, nil)
		fakeStore.IsRangeConsolidatedReturnsOnCall(3, false, nil)
		fakeStore.GetBillableEventRowsReturns(fakeRows, nil)
		fakeStore.GetConsolidatedBillableEventRowsReturns(fakeRows, nil)

		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-15")
		q.Set("range_stop", "2001-04-15")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetBillableEventRowsCallCount()).To(Equal(2))
		Expect(fakeStore.GetConsolidatedBillableEventRowsCallCount()).To(Equal(2))
		_, filter1 := fakeStore.GetBillableEventRowsArgsForCall(0)
		_, filter2 := fakeStore.GetConsolidatedBillableEventRowsArgsForCall(0)
		_, filter3 := fakeStore.GetConsolidatedBillableEventRowsArgsForCall(1)
		_, filter4 := fakeStore.GetBillableEventRowsArgsForCall(1)
		receivedFilters := []eventio.EventFilter{
			filter1,
			filter2,
			filter3,
			filter4,
		}
		Expect(receivedFilters).To(Equal([]eventio.EventFilter{
			{RangeStart: "2001-01-15", RangeStop: "2001-02-01", OrgGUIDs: []string{orgGUID1}},
			{RangeStart: "2001-02-01", RangeStop: "2001-03-01", OrgGUIDs: []string{orgGUID1}},
			{RangeStart: "2001-03-01", RangeStop: "2001-04-01", OrgGUIDs: []string{orgGUID1}},
			{RangeStart: "2001-04-01", RangeStop: "2001-04-15", OrgGUIDs: []string{orgGUID1}},
		}))

		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})
	It("should fetch ConsolidatedBillableEvents when the filter range has been consolidated", func() {
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
		fakeAuthorizer.HasBillingAccessReturns(true, nil)
		fakeRows := &eventiofakes.FakeBillableEventRows{}
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
		fakeStore.IsRangeConsolidatedReturns(true, nil)
		fakeStore.GetConsolidatedBillableEventRowsReturns(fakeRows, nil)

		u := url.URL{}
		u.Path = "/billable_events"
		q := u.Query()
		q.Set("org_guid", orgGUID1)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-02-01")
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetBillableEventRowsCallCount()).To(Equal(0))
		Expect(fakeStore.GetConsolidatedBillableEventRowsCallCount()).To(Equal(1))
		_, filter := fakeStore.GetConsolidatedBillableEventRowsArgsForCall(0)
		Expect(filter.RangeStart).To(Equal("2001-01-01"))
		Expect(filter.RangeStop).To(Equal("2001-02-01"))
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
