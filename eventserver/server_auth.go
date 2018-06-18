package eventserver

import (
	"errors"

	"github.com/alphagov/paas-billing/eventserver/auth"
	"github.com/labstack/echo"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrBillingAccess      = errors.New("you need to be billing_manager or an administrator to retreive the billing data")
)

// isAdmin checks if there is a token in the request with an operator scope
// (cloud_controler.admin / cloud_controler.read_only_admin / global_auditor)
// isBillingManager checks if the user has either role assigned within the org
// (billing_manager / org_manager)
// Either of the above should satisfy the authorizer.
func authorize(c echo.Context, uaa auth.Authenticator, orgs []string) (bool, error) {
	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		return false, err
	}
	authorizer, err := uaa.NewAuthorizer(token)
	if err != nil {
		return false, err
	}
	isAdmin, errA := authorizer.Admin()
	hasBillingAccess, errM := authorizer.HasBillingAccess(orgs)
	if errA != nil && errM != nil {
		return false, ErrInvalidCredentials
	}
	if !isAdmin && !hasBillingAccess {
		return false, ErrBillingAccess
	}
	return true, nil
}
