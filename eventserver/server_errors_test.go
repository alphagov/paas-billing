package eventserver_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"

	"github.com/labstack/echo"

	. "github.com/alphagov/paas-billing/eventserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {

	var (
		e *echo.Echo
	)

	BeforeEach(func() {
		e = New(Config{
			Logger: lager.NewLogger("test"),
		})
	})

	AfterEach(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		e.Shutdown(ctx)
	})

	It("should catch panics and turn them into 'internal server error' json errors", func() {
		e.GET("/panic", func(c echo.Context) error {
			panic("bang")
			return c.JSON(http.StatusOK, nil)
		})
		req := httptest.NewRequest(echo.GET, "/panic", nil)
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		Expect(res.Code).To(Equal(500))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		Expect(res.Body.String()).To(Equal(`{"error":"internal server error"}`))
	})

	It("should return 'internal server error' for unknown errors", func() {
		e.GET("/error", func(c echo.Context) error {
			return fmt.Errorf("this-error-leaks-sensitive-info")
		})
		req := httptest.NewRequest(echo.GET, "/error", nil)
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		Expect(res.Code).To(Equal(500))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		Expect(res.Body.String()).To(Equal(`{"error":"internal server error"}`))
	})

	It("should pass-through errors of type echo.HTTPError (with message type string)", func() {
		e.GET("/expected-error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusTeapot, "friendly-string-message")
		})
		req := httptest.NewRequest(echo.GET, "/expected-error", nil)
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		Expect(res.Code).To(Equal(http.StatusTeapot))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		Expect(res.Body.String()).To(Equal(`{"error":"friendly-string-message"}`))
	})

	It("should pass-through errors of type echo.HTTPError (with message type error)", func() {
		e.GET("/expected-error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusTeapot, errors.New("friendly-error-message"))
		})
		req := httptest.NewRequest(echo.GET, "/expected-error", nil)
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)
		Expect(res.Code).To(Equal(http.StatusTeapot))
		Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		Expect(res.Body.String()).To(Equal(`{"error":"friendly-error-message"}`))
	})

})
