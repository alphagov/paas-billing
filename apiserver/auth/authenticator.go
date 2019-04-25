package auth

import "github.com/labstack/echo"

type Authenticator interface {
	Exchange(c echo.Context) error
	Authorize(c echo.Context) error
	NewAuthorizer(string) (Authorizer, error)
}
