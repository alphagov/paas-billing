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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VATRatesHandler", func() {
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
		fakeStore.GetVATRatesReturns([]eventio.VATRate{
			{
				Code:      "Standard",
				ValidFrom: "2001-01-01",
				ValidTo:   "9999-12-31",
				Rate:      0.2,
			},
			{
				Code:      "Reduced",
				ValidFrom: "2001-07-01",
				ValidTo:   "9999-12-31",
				Rate:      0.05,
			},
			{
				Code:      "Zero",
				ValidFrom: "2002-01-01",
				ValidTo:   "9999-12-31",
				Rate:      0.0,
			},
		}, nil)
		rangeStart := "2001-01-01"
		rangeStop := "2002-02-02"

		u := url.URL{}
		u.Path = "/vat_rates"
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

		Expect(fakeStore.GetVATRatesCallCount()).To(Equal(1))

		filter := fakeStore.GetVATRatesArgsForCall(0)
		Expect(filter.RangeStart).To(Equal(rangeStart))
		Expect(filter.RangeStop).To(Equal(rangeStop))

		Expect(res.Body).To(MatchJSON(`[
            {
                "code": "Standard",
                "valid_from": "2001-01-01",
                "valid_to": "9999-12-31",
                "rate": 0.2
            },
            {
                "code": "Reduced",
                "valid_from": "2001-07-01",
                "valid_to": "9999-12-31",
                "rate": 0.05
            },
            {
                "code": "Zero",
                "valid_from": "2002-01-01",
                "valid_to": "9999-12-31",
                "rate": 0.0
            }
        ]`))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

})
