package server_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/server"
	"github.com/alphagov/paas-billing/server/auth"
	"github.com/labstack/echo"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type SimpleAuthenticator struct{}

func (sa *SimpleAuthenticator) Authorize(c echo.Context) error {
	return fmt.Errorf("unauthorized-in-test")
}

func (sa *SimpleAuthenticator) Exchange(c echo.Context) error {
	return fmt.Errorf("unauthorized-in-test")
}

func (sa *SimpleAuthenticator) NewAuthorizer(token string) (auth.Authorizer, error) {
	return nil, fmt.Errorf("not-authorizor-in-test")
}

var _ = Describe("Server", func() {

	var cfg = server.Config{
		// Authority: &SimpleAuthenticator{},
		Logger: lager.NewLogger("test"),
	}

	It("should catch panics and turn them into 'internal server error' json errors", func() {
		e := server.New(cfg)
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
		e := server.New(cfg)
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
		e := server.New(cfg)
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
		e := server.New(cfg)
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
