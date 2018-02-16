package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo"
)

const (
	CookieAuthorization = "authorization"
)

type Authenticator interface {
	Exchange(c echo.Context) error
	Authorize(c echo.Context) error
	NewAuthorizer(string) (Authorizer, error)
}

func UAATokenAuthentication(authority Authenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, err := getTokenFromRequest(c)
			if err != nil {
				return unauthorized(c, err)
			}
			authorizer, err := authority.NewAuthorizer(token)
			if err != nil {
				return unauthorized(c, err)
			}
			c.Set("authorizer", authorizer)
			return next(c)
		}
	}
}

func unauthorized(c echo.Context, err error) error {
	acceptHeader := c.Request().Header.Get(echo.HeaderAccept)
	accepts := strings.Split(acceptHeader, ",")
	for _, accept := range accepts {
		if accept == echo.MIMETextHTML || accept == echo.MIMETextHTMLCharsetUTF8 {
			return c.Redirect(http.StatusFound, "/oauth/authorize")
		}
	}
	return echo.NewHTTPError(http.StatusUnauthorized, err)
}

func AdminOnly(fn echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authorized, ok := c.Get("authorizer").(Authorizer)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "authentication middleware not configured")
		}
		if !authorized.Admin() {
			return echo.NewHTTPError(http.StatusUnauthorized, "access denied")
		}
		return fn(c)
	}
}
