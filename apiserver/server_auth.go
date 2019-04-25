package apiserver

import (
	"errors"
	"fmt"

	"github.com/alphagov/paas-billing/apiserver/auth"
	"github.com/labstack/echo"
)

// isAdmin checks if there is a token in the request with an operator scope
// (cloud_controller.admin / cloud_controller.read_only_admin / global_auditor)
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

	isAdmin, err := authorizer.Admin()
	if err != nil {
		return false, fmt.Errorf("invalid credentials: %s", err)
	}
	if isAdmin {
		return true, nil
	}

	hasBillingAccess, err := authorizer.HasBillingAccess(orgs)
	if err != nil {
		return false, fmt.Errorf("invalid credentials: %s", err)
	}
	if hasBillingAccess {
		return true, nil
	}
	return false, errors.New("you need to be billing_manager or an administrator to retrieve the billing data")
}
