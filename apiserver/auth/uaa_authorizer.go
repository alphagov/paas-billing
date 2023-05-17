package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
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
		return nil, errors.New("no auth token: unauthorized")
	}
	return &ClientAuthorizer{
		endpoint: uaa.Config.Endpoint.TokenURL,
		token:    token,
	}, nil
}

type UAAClaims struct {
	UserID    string   `json:"user_id"`
	Scope     []string `json:"scope"`
	Email     string   `json:"email"`
	UserName  string   `json:"user_name"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

func (claims *UAAClaims) Valid() error {
	return nil
}

type ClientAuthorizer struct {
	claims   *UAAClaims
	endpoint string
	token    string
	scopes   []string
}

func (a ClientAuthorizer) client() (*cfclient.Client, error) {
	return cfclient.NewClient(&cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Token:             a.token,
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
	})
}

func (a *ClientAuthorizer) HasBillingAccess(requestedOrgs []string) (bool, error) {
	err := a.composeClaims()
	if err != nil {
		return false, err
	}
	cf, err := a.client()
	if err != nil {
		return false, err
	}
	billingManagerOrganisations, err := cf.ListUserBillingManagedOrgs(a.claims.UserID)
	if err != nil {
		return false, err
	}
	managerOrganisations, err := cf.ListUserManagedOrgs(a.claims.UserID)
	if err != nil {
		return false, err
	}
	orgGUIDs := []string{}
	for _, org := range billingManagerOrganisations {
		orgGUIDs = append(orgGUIDs, org.Guid)
	}
	for _, org := range managerOrganisations {
		orgGUIDs = append(orgGUIDs, org.Guid)
	}

	if ok, mismatch := SliceMatches(requestedOrgs, orgGUIDs); !ok {
		return false, fmt.Errorf("authorizer: no access to organisation: %s", mismatch)
	}

	return true, nil
}

func (a *ClientAuthorizer) Admin() (bool, error) {
	if ok, err := a.hasScope("cloud_controller.admin_read_only"); ok {
		return true, nil
	} else if err != nil {
		return false, err
	}
	if ok, err := a.hasScope("cloud_controller.global_auditor"); ok {
		return true, nil
	} else if err != nil {
		return false, err
	}
	if ok, err := a.hasScope("cloud_controller.admin"); ok {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
}

func (a *ClientAuthorizer) hasScope(scope string) (bool, error) {
	if a.scopes == nil {
		var err error
		a.scopes, err = a.getVerifiedScopes()
		if err != nil {
			return false, err
		}
	}
	for _, authorizedScope := range a.scopes {
		if scope == authorizedScope {
			return true, nil
		}
	}
	return false, nil
}

func (a *ClientAuthorizer) composeClaims() error {
	// See if we've already performed all this actions.
	if a.claims != nil {
		return nil
	}

	tokenEndpoint, err := url.Parse(a.endpoint)
	if err != nil {
		return err
	}
	tokenEndpoint.Path = "/token_keys"
	v := url.Values{}
	v.Set("token", a.token)
	req, err := http.NewRequest("GET", tokenEndpoint.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(os.Getenv("CF_CLIENT_ID"), os.Getenv("CF_CLIENT_SECRET"))
	resp, err := newHTTPClient().Do(req) // FIXME: token_keys could be cached, that's kinda the point
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("got status code %d while fetching token_keys", resp.StatusCode)
	}
	var verified struct {
		Keys []struct {
			Public string `json:"value"`
			Alg    string `json:"alg"`
			Kid    string `json:"kid"`
		} `json:"keys"`
	}
	err = json.NewDecoder(resp.Body).Decode(&verified)
	if err != nil {
		return err
	}
	token, err := jwt.ParseWithClaims(a.token, &UAAClaims{}, func(token *jwt.Token) (interface{}, error) {
		for _, key := range verified.Keys {
			if key.Alg == token.Header["alg"] {
				if key.Alg == "RS256" || key.Alg == "RS512" {
					if key.Kid == token.Header["kid"] {
						return jwt.ParseRSAPublicKeyFromPEM([]byte(key.Public))
					}
				}
			}
		}
		return nil, errors.New("unable to verify token")
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("token invalid")
	}
	claims, ok := token.Claims.(*UAAClaims)
	if !ok {
		return fmt.Errorf("token claims type")
	}

	a.claims = claims

	return nil
}

func (a *ClientAuthorizer) getVerifiedScopes() ([]string, error) {
	err := a.composeClaims()
	if err != nil {
		return nil, err
	}

	return a.claims.Scope, nil
}
