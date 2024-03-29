package apiserver_test

import (
	"context"
	"errors"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"

	"github.com/labstack/echo/v4"

	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventStoreStatusHandler", func() {

	var (
		ctx       context.Context
		cancel    context.CancelFunc
		cfg       Config
		fakeStore *eventiofakes.FakeEventStore
	)

	BeforeEach(func() {

		fakeStore = &eventiofakes.FakeEventStore{}
		cfg = Config{
			Logger: lager.NewLogger("test"),
			Store:  fakeStore,
		}
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		defer cancel()
	})

	It("should return a json ok true message", func() {
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()

		e := NewBaseServer(cfg)
		e.ServeHTTP(res, req)

		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"ok": true
		}`))
		Expect(res.Code).To(Equal(200))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})

	It("should return a json ok false message if store db failure", func() {
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()

		fakeStore.PingReturns(errors.New("Fake DB Error"))

		e := NewBaseServer(cfg)
		e.ServeHTTP(res, req)

		defer e.Shutdown(ctx)

		Expect(res.Body).To(MatchJSON(`{
			"ok": false
		}`))
		Expect(res.Code).To(Equal(500))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	})
})
