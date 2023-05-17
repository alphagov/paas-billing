package apiserver_test

import (
	"context"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/apiserver/auth/authfakes"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PricingPlansHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *authfakes.FakeAuthenticator
		fakeAuthorizer    *authfakes.FakeAuthorizer
		fakeStore         *eventiofakes.FakeEventStore
		token             = "ACCESS_GRANTED_TOKEN"
	)

	BeforeEach(func() {
		fakeStore = &eventiofakes.FakeEventStore{}
		fakeAuthenticator = &authfakes.FakeAuthenticator{}
		fakeAuthorizer = &authfakes.FakeAuthorizer{}
		cfg = Config{
			Authenticator: fakeAuthenticator,
			Logger:        lager.NewLogger("test"),
			Store:         fakeStore,
			EnablePanic:   true,
		}
		ctx, cancel = context.WithCancel(context.Background())
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
	})

	AfterEach(func() {
		defer cancel()
	})

	It("should request the plans from the store and return json", func() {
		fakeStore.GetPricingPlansReturns([]eventio.PricingPlan{
			{
				PlanGUID:      eventstore.ComputePlanGUID,
				ValidFrom:     "2001-01-01",
				Name:          "PLAN1",
				MemoryInMB:    164,
				StorageInMB:   165,
				NumberOfNodes: 1,
				Components: []eventio.PricingPlanComponent{
					{
						Name:         "PLAN1COMPONENT1",
						Formula:      "1111 * 1",
						CurrencyCode: "GBP",
						VATCode:      "Standard",
					},
				},
			},
			{
				PlanGUID:      eventstore.ComputePlanGUID,
				ValidFrom:     "2002-01-01",
				Name:          "PLAN2",
				MemoryInMB:    264,
				StorageInMB:   265,
				NumberOfNodes: 2,
				Components: []eventio.PricingPlanComponent{
					{
						Name:         "PLAN2COMPONENT1",
						Formula:      "2222 * 1",
						CurrencyCode: "GBP",
						VATCode:      "Standard",
					},
					{
						Name:         "PLAN2COMPONENT2",
						Formula:      "2222 * 2",
						CurrencyCode: "GBP",
						VATCode:      "Standard",
					},
				},
			},
		}, nil)
		rangeStart := "2001-01-01"
		rangeStop := "2002-02-02"

		u := url.URL{}
		u.Path = "/pricing_plans"
		q := u.Query()
		q.Set("range_start", rangeStart)
		q.Set("range_stop", rangeStop)
		u.RawQuery = q.Encode()
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		req.Header.Set("Authorization", "bearer "+token)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetPricingPlansCallCount()).To(Equal(1))

		filter := fakeStore.GetPricingPlansArgsForCall(0)
		Expect(filter.RangeStart).To(Equal(rangeStart))
		Expect(filter.RangeStop).To(Equal(rangeStop))

		Expect(res.Body).To(MatchJSON(`[
			{
				"name": "PLAN1",
				"plan_guid": "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"valid_from": "2001-01-01",
				"components": [
					{
						"name": "PLAN1COMPONENT1",
						"formula": "1111 * 1",
						"vat_code": "Standard",
						"currency_code": "GBP"
					}
				],
				"memory_in_mb": 164,
				"storage_in_mb": 165,
				"number_of_nodes": 1
			},
			{
				"name": "PLAN2",
				"plan_guid": "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"valid_from": "2002-01-01",
				"components": [
					{
						"name": "PLAN2COMPONENT1",
						"formula": "2222 * 1",
						"vat_code": "Standard",
						"currency_code": "GBP"
					},
					{
						"name": "PLAN2COMPONENT2",
						"formula": "2222 * 2",
						"vat_code": "Standard",
						"currency_code": "GBP"
					}
				],
				"memory_in_mb": 264,
				"storage_in_mb": 265,
				"number_of_nodes": 2
			}
		]`))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
