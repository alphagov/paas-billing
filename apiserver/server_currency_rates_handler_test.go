package apiserver_test

import (
	"context"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CurrencyRatesHandler", func() {
	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *fakes.FakeAuthenticator
		fakeAuthorizer    *fakes.FakeAuthorizer
		fakeStore         *fakes.FakeEventStore
		token             = "ACCESS_GRANTED_TOKEN"
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
		fakeAuthenticator.NewAuthorizerReturns(fakeAuthorizer, nil)
		fakeAuthorizer.AdminReturns(false, nil)
	})

	AfterEach(func() {
		defer cancel()
	})

	It("should request the rates from the store and return json", func() {
		fakeStore.GetCurrencyRatesReturns([]eventio.CurrencyRate{
			{
				Code:      "GBP",
				ValidFrom: "2001-01-01",
				Rate:      1.0,
			},
			{
				Code:      "USD",
				ValidFrom: "2002-01-01",
				Rate:      0.8,
			},
		}, nil)
		rangeStart := "2001-01-01"
		rangeStop := "2002-02-02"

		u := url.URL{}
		u.Path = "/currency_rates"
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

		Expect(fakeStore.GetCurrencyRatesCallCount()).To(Equal(1))

		filter := fakeStore.GetCurrencyRatesArgsForCall(0)
		Expect(filter.RangeStart).To(Equal(rangeStart))
		Expect(filter.RangeStop).To(Equal(rangeStop))

		Expect(res.Body).To(MatchJSON(`[
            {
                "code": "GBP",
                "valid_from": "2001-01-01",
                "rate": 1.0
            },
            {
                "code": "USD",
                "valid_from": "2002-01-01",
                "rate": 0.8
            }
        ]`))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
