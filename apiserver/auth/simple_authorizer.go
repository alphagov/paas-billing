package auth

import (
	"fmt"
	"strings"

	"github.com/labstack/echo"
)

var FakeBearerToken = "Bearer FAKE_TOKEN"

type SimpleAuthorizer struct {
	admin              bool
	authorizedOrgGUIDs []string
}

func (sa *SimpleAuthorizer) HasBillingAccess(orgs []string) (bool, error) {
	if ok, missmatch := SliceMatches(orgs, sa.authorizedOrgGUIDs); !ok {
		return false, fmt.Errorf("authorizer: no access to organisation: %s", missmatch)
	}

	return true, nil
}

func (sa *SimpleAuthorizer) Admin() (bool, error) {
	return sa.admin, nil
}

type SimpleAuthenticator struct {
	admin              bool
	authorizedOrgGUIDs []string
	authorizationError error
}

func (sa *SimpleAuthenticator) Authorize(c echo.Context) error {
	return sa.authorizationError
}

func (sa *SimpleAuthenticator) Exchange(c echo.Context) error {
	return sa.authorizationError
}

func (sa *SimpleAuthenticator) NewAuthorizer(token string) (Authorizer, error) {
	exp := strings.TrimPrefix(FakeBearerToken, "Bearer ")
	if token != exp {
		return nil, fmt.Errorf("SimpleAuthenticator failed: expected '%s' got '%s'", exp, token)
	}
	return &SimpleAuthorizer{
		authorizedOrgGUIDs: sa.authorizedOrgGUIDs,
		admin:              sa.admin,
	}, nil
}

var AuthenticatedNonAdmin = &SimpleAuthenticator{
	admin: false,
	authorizedOrgGUIDs: []string{
		"org_guid",
		"org_guid1",
		"org_guid2",
		"00000001-0001-0000-0000-000000000000",
		"00000001-0002-0000-0000-000000000000",
		"00000001-0003-0000-0000-000000000000",
		"00000002-0001-0000-0000-000000000000",
		"00000002-0002-0000-0000-000000000000",
		"00000003-0005-0000-0000-000000000000",
		"o1s1",
		"o2s1",
	},
}

var AuthenticatedAdmin = &SimpleAuthenticator{
	admin:              true,
	authorizedOrgGUIDs: []string{},
}

var NonAuthenticated = &SimpleAuthenticator{
	admin:              false,
	authorizedOrgGUIDs: []string{},
	authorizationError: nil,
}
