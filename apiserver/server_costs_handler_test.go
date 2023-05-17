package apiserver_test

import (
	"context"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/apiserver/auth/authfakes"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"
	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TotalCostHandler", func() {
	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *authfakes.FakeAuthenticator
		fakeAuthorizer    *authfakes.FakeAuthorizer
		fakeStore         *eventiofakes.FakeEventStore
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

	It("should return the total cost by plan_guids as json", func() {
		fakeStore.GetTotalCostReturns([]eventio.TotalCost{
			{
				PlanGUID: "b1341aba-63f9-4747-9abd-d48313483044",
				Cost:     45.23,
			},
			{
				PlanGUID: "f1019263-081c-4776-bd9e-b056e4a32e31",
				Cost:     543,
			},
			{
				PlanGUID: "f19ac069-ed93-47c9-98ff-c23945b56cb9",
				Cost:     6.23,
			},
		}, nil)

		u := url.URL{}
		u.Path = "/totals"
		req := httptest.NewRequest(echo.GET, u.String(), nil)
		res := httptest.NewRecorder()

		e := New(cfg)
		e.ServeHTTP(res, req)
		defer e.Shutdown(ctx)

		Expect(fakeStore.GetTotalCostCallCount()).To(Equal(1))

		Expect(res.Body).To(MatchJSON(`[
            {
                "plan_guid": "b1341aba-63f9-4747-9abd-d48313483044",
                "cost": 45.23
			},
			{
                "plan_guid": "f1019263-081c-4776-bd9e-b056e4a32e31",
                "cost": 543
			},
			{
                "plan_guid": "f19ac069-ed93-47c9-98ff-c23945b56cb9",
                "cost": 6.23
            }
        ]`))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
