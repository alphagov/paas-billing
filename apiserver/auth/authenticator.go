package auth

import "github.com/labstack/echo/v4"

//counterfeiter:generate . Authenticator
type Authenticator interface {
	Exchange(c echo.Context) error
	Authorize(c echo.Context) error
	NewAuthorizer(string) (Authorizer, error)
}
