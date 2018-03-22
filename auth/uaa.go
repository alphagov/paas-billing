package auth

import (
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"golang.org/x/oauth2"
)

type Claims struct {
	Val string `json:"val"`
	jwt.StandardClaims
}

type UAA struct {
	Config *oauth2.Config
}

func (uaa *UAA) Authorize(c echo.Context) error {
	return fmt.Errorf("oauth login flow not implemented: an access token with cloud_controller.read is required")
}

func (uaa *UAA) Exchange(c echo.Context) error {
	return fmt.Errorf("oauth login flow not implemented: an access token with cloud_controller.read is required")
}

func (uaa *UAA) NewAuthorizer(token string) (Authorizer, error) {
	if token == "" {
		return nil, errors.New("no auth token: unauthozed")
	}
	return &ClientAuthorizer{
		endpoint: uaa.Config.Endpoint.TokenURL,
		token:    token,
	}, nil
}
