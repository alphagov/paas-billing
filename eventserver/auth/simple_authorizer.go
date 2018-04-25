package auth

import (
	"fmt"
	"strings"

	"github.com/labstack/echo"
)

var FakeBearerToken = "Bearer FAKE_TOKEN"

type SimpleAuthorizer struct {
	admin                bool
	authorizedSpaceGUIDs []string
}

func (sa *SimpleAuthorizer) Spaces() ([]string, error) {
	return sa.authorizedSpaceGUIDs, nil
}

func (sa *SimpleAuthorizer) Admin() (bool, error) {
	return sa.admin, nil
}

type SimpleAuthenticator struct {
	admin                bool
	authorizedSpaceGUIDs []string
	authorizationError   error
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
		authorizedSpaceGUIDs: sa.authorizedSpaceGUIDs,
		admin:                sa.admin,
	}, nil
}

var AuthenticatedNonAdmin = &SimpleAuthenticator{
	admin: false,
	authorizedSpaceGUIDs: []string{
		"space_guid",
		"space_guid1",
		"space_guid2",
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
	admin:                true,
	authorizedSpaceGUIDs: []string{},
}

var NonAuthenticated = &SimpleAuthenticator{
	admin:                false,
	authorizedSpaceGUIDs: []string{},
	authorizationError:   nil,
}
