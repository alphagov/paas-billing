package eventserver

import (
	"errors"

	"github.com/alphagov/paas-billing/eventserver/auth"
	"github.com/labstack/echo"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAdminOnly          = errors.New("billing data currently requires cloud_controller.admin or cloud_controller.global_auditor scope")
)

// isAdmin checks if there is a token in the request with an operator scope
// (cloud_controler.admin / global_auditor)
func authorize(c echo.Context, uaa auth.Authenticator) (bool, error) {
	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		return false, err
	}
	authorizer, err := uaa.NewAuthorizer(token)
	if err != nil {
		return false, err
	}
	isAdmin, err := authorizer.Admin()
	if err != nil {
		return false, ErrInvalidCredentials
	}
	if !isAdmin {
		return false, ErrAdminOnly
	}
	return true, nil
}
