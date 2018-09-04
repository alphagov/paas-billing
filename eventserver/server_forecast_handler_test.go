package eventserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/eventserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ForecastEventsHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *fakes.FakeAuthenticator
		fakeAuthorizer    *fakes.FakeAuthorizer
		fakeStore         *fakes.FakeEventStore
	)

	BeforeEach(func() {
		fakeStore = &fakes.FakeEventStore{}
		fakeAuthenticator = &fakes.FakeAuthenticator{}
		fakeAuthorizer = &fakes.FakeAuthorizer{}
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
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

	It("should forecast BillableEvents based on given UsageEvents", func() {
		var err error
		inputEvent1JSON := `{
			"event_guid": "00000000-0000-0000-0000-000000000001",
			"resource_guid": "00000000-0000-0000-0001-000000000001",
			"resource_name": "fake-app-1",
			"resource_type": "app",
			"org_guid": "` + eventstore.DummyOrgGUID + `",
			"org_name": "` + eventstore.DummyOrgName + `",
			"space_guid": "` + eventstore.DummySpaceGUID + `",
			"space_name": "` + eventstore.DummySpaceName + `",
			"event_start": "2001-01-01T00:00",
			"event_stop": "2001-01-01T01:00",
			"plan_unique_id": "` + eventstore.DummyPlanUniqueID + `",
			"plan_name": "instance",
			"service_name": "app",
			"service_guid": "` + eventstore.ComputeServiceGUID + `",
			"number_of_nodes": 2,
			"memory_in_mb": 64,
			"storage_in_mb": 1024
		}`
		inputEvent2JSON := `{
			"event_guid": "00000000-0000-0000-0000-000000000002",
			"resource_guid": "00000000-0000-0000-0002-000000000002",
			"resource_name": "fake-app-2",
			"resource_type": "app",
			"org_guid": "` + eventstore.DummyOrgGUID + `",
			"org_name": "` + eventstore.DummyOrgName + `",
			"space_guid": "` + eventstore.DummySpaceGUID + `",
			"space_name": "` + eventstore.DummySpaceName + `",
			"event_start": "2001-01-01T00:00",
			"event_stop": "2001-01-01T05:00",
			"plan_unique_id": "` + eventstore.DummyPlanUniqueID + `",
			"plan_name": "instance",
			"service_name": "app",
			"service_guid": "` + eventstore.ComputeServiceGUID + `",
			"number_of_nodes": 1,
			"memory_in_mb": 64,
			"storage_in_mb": 1024
		}`

		billingEvent1JSON := `{
			"event_guid": "raw-json-guid-1"
		}`
		billingEvent2JSON := `{
			"event_guid": "raw-json-guid-2"
		}`

		inputEventsJSON := fmt.Sprintf("[%s]", strings.Join([]string{
			inputEvent1JSON,
			inputEvent2JSON,
		}, ","))

		inputEvents := []eventio.UsageEvent{}
		err = json.Unmarshal([]byte(inputEventsJSON), &inputEvents)
		Expect(err).ToNot(HaveOccurred())

		fakeRows := &fakes.FakeBillableEventRows{}
		fakeRows.CloseReturns(nil)
		fakeRows.NextReturnsOnCall(0, true)
		fakeRows.NextReturnsOnCall(1, true)
		fakeRows.NextReturnsOnCall(2, false)
		fakeRows.EventJSONReturnsOnCall(0, []byte(billingEvent1JSON), nil)
		fakeRows.EventJSONReturnsOnCall(1, []byte(billingEvent2JSON), nil)
		fakeStore.ForecastBillableEventRowsReturnsOnCall(0, fakeRows, nil)

		u := url.URL{}
		u.Path = "/forecast_events"
		q := u.Query()
		q.Set("org_guid", eventstore.DummyOrgGUID)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-02-01")
		q.Set("events", inputEventsJSON)
		u.RawQuery = q.Encode()

		req := httptest.NewRequest(echo.GET, u.String(), nil)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.ForecastBillableEventRowsCallCount()).To(Equal(1))
		requestedInputEvents, requestedFilter := fakeStore.ForecastBillableEventRowsArgsForCall(0)

		Expect(requestedInputEvents).To(Equal(inputEvents))
		Expect(requestedFilter.RangeStart).To(Equal("2001-01-01"))
		Expect(requestedFilter.RangeStop).To(Equal("2001-02-01"))
		Expect(requestedFilter.OrgGUIDs).To(Equal([]string{eventstore.DummyOrgGUID}))

		Expect(fakeRows.NextCallCount()).To(Equal(3))
		Expect(fakeRows.EventJSONCallCount()).To(Equal(2))
		Expect(fakeRows.CloseCallCount()).To(Equal(1))

		outputEventsJSON := fmt.Sprintf("[%s]", strings.Join([]string{
			billingEvent1JSON,
			billingEvent2JSON,
		}, ","))
		Expect(res.Body).To(MatchJSON(outputEventsJSON))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should suport pass the Plan Unique ID using the legacy `plan_guid` and return BillableEvents based on given UsageEvents", func() {
		var err error
		inputEvent1JSON := `{
			"event_guid": "00000000-0000-0000-0000-000000000001",
			"resource_guid": "00000000-0000-0000-0001-000000000001",
			"resource_name": "fake-app-1",
			"resource_type": "app",
			"org_guid": "` + eventstore.DummyOrgGUID + `",
			"org_name": "` + eventstore.DummyOrgName + `",
			"space_guid": "` + eventstore.DummySpaceGUID + `",
			"space_name": "` + eventstore.DummySpaceName + `",
			"event_start": "2001-01-01T00:00",
			"event_stop": "2001-01-01T01:00",
			"plan_guid": "` + eventstore.DummyPlanUniqueID + `",
			"plan_name": "instance",
			"service_name": "app",
			"service_guid": "` + eventstore.ComputeServiceGUID + `",
			"number_of_nodes": 2,
			"memory_in_mb": 64,
			"storage_in_mb": 1024
		}`
		inputEvent2JSON := `{
			"event_guid": "00000000-0000-0000-0000-000000000002",
			"resource_guid": "00000000-0000-0000-0002-000000000002",
			"resource_name": "fake-app-2",
			"resource_type": "app",
			"org_guid": "` + eventstore.DummyOrgGUID + `",
			"org_name": "` + eventstore.DummyOrgName + `",
			"space_guid": "` + eventstore.DummySpaceGUID + `",
			"space_name": "` + eventstore.DummySpaceName + `",
			"event_start": "2001-01-01T00:00",
			"event_stop": "2001-01-01T05:00",
			"plan_guid": "` + eventstore.DummyPlanUniqueID + `",
			"plan_name": "instance",
			"service_name": "app",
			"service_guid": "` + eventstore.ComputeServiceGUID + `",
			"number_of_nodes": 1,
			"memory_in_mb": 64,
			"storage_in_mb": 1024
		}`

		billingEvent1JSON := `{
			"event_guid": "raw-json-guid-1"
		}`
		billingEvent2JSON := `{
			"event_guid": "raw-json-guid-2"
		}`

		inputEventsJSON := fmt.Sprintf("[%s]", strings.Join([]string{
			inputEvent1JSON,
			inputEvent2JSON,
		}, ","))

		inputEvents := []eventio.UsageEvent{}
		err = json.Unmarshal([]byte(inputEventsJSON), &inputEvents)
		Expect(err).ToNot(HaveOccurred())
		for i := range inputEvents {
			inputEvents[i].PlanUniqueID = inputEvents[i].LegacyPlanGUID
		}

		fakeRows := &fakes.FakeBillableEventRows{}
		fakeRows.CloseReturns(nil)
		fakeRows.NextReturnsOnCall(0, true)
		fakeRows.NextReturnsOnCall(1, true)
		fakeRows.NextReturnsOnCall(2, false)
		fakeRows.EventJSONReturnsOnCall(0, []byte(billingEvent1JSON), nil)
		fakeRows.EventJSONReturnsOnCall(1, []byte(billingEvent2JSON), nil)
		fakeStore.ForecastBillableEventRowsReturnsOnCall(0, fakeRows, nil)

		u := url.URL{}
		u.Path = "/forecast_events"
		q := u.Query()
		q.Set("org_guid", eventstore.DummyOrgGUID)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-02-01")
		q.Set("events", inputEventsJSON)
		u.RawQuery = q.Encode()

		req := httptest.NewRequest(echo.GET, u.String(), nil)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.ForecastBillableEventRowsCallCount()).To(Equal(1))
		requestedInputEvents, requestedFilter := fakeStore.ForecastBillableEventRowsArgsForCall(0)

		Expect(requestedInputEvents).To(Equal(inputEvents))
		Expect(requestedFilter.RangeStart).To(Equal("2001-01-01"))
		Expect(requestedFilter.RangeStop).To(Equal("2001-02-01"))
		Expect(requestedFilter.OrgGUIDs).To(Equal([]string{eventstore.DummyOrgGUID}))

		Expect(fakeRows.NextCallCount()).To(Equal(3))
		Expect(fakeRows.EventJSONCallCount()).To(Equal(2))
		Expect(fakeRows.CloseCallCount()).To(Equal(1))

		outputEventsJSON := fmt.Sprintf("[%s]", strings.Join([]string{
			billingEvent1JSON,
			billingEvent2JSON,
		}, ","))
		Expect(res.Body).To(MatchJSON(outputEventsJSON))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should throw an error if an unauthorized OrgGUID is requested", func() {
		unauthorizedGUID := "cc0deaaf-bc3c-4c07-82c1-63b9f6dee4b3"
		u := url.URL{}
		u.Path = "/forecast_events"
		q := u.Query()
		q.Set("org_guid", unauthorizedGUID)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-02-01")
		q.Set("events", `[]`)
		u.RawQuery = q.Encode()

		req := httptest.NewRequest(echo.GET, u.String(), nil)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.ForecastBillableEventRowsCallCount()).To(Equal(0))
		Expect(res.Body).To(MatchJSON(fmt.Sprintf(`{
			"error": "you are not authorized to forecast events for org '%s'"
		}`, unauthorizedGUID)))
		Expect(res.Code).To(Equal(403))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
