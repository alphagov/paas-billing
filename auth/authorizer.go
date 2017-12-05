package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
)

type Authorizer interface {
	Spaces() ([]string, error)
	Admin() bool
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
	endpoint string
	token    string
	scopes   []string
}

func (a ClientAuthorizer) client() (cloudfoundry.Client, error) {
	return cloudfoundry.NewClient(&cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Token:             a.token,
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
	})
}

func (a *ClientAuthorizer) Spaces() ([]string, error) {
	cf, err := a.client()
	if err != nil {
		return nil, err
	}
	spaces, err := cf.GetSpaces()
	if err != nil {
		return nil, err
	}
	spaceGUIDs := []string{}
	for guid, _ := range spaces {
		spaceGUIDs = append(spaceGUIDs, guid)
	}
	return spaceGUIDs, nil
}

func (a *ClientAuthorizer) Admin() bool {
	return a.hasScope("cloud_controller.admin_read_only") || a.hasScope("cloud_controller.global_auditor") || a.hasScope("cloud_controller.admin")
}

func (a *ClientAuthorizer) hasScope(scope string) bool {
	if a.scopes == nil {
		var err error
		a.scopes, err = a.getVerifiedScopes()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
	}
	for _, authorizedScope := range a.scopes {
		if scope == authorizedScope {
			return true
		}
	}
	return false
}

func (a *ClientAuthorizer) getVerifiedScopes() ([]string, error) {
	tokenEndpoint, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, err
	}
	tokenEndpoint.Path = "/token_keys"
	v := url.Values{}
	v.Set("token", a.token)
	req, err := http.NewRequest("GET", tokenEndpoint.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(os.Getenv("CF_CLIENT_ID"), os.Getenv("CF_CLIENT_SECRET"))
	resp, err := newHTTPClient().Do(req) // FIXME: token_keys could be cached, that's kinda the point
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("got status code %d while fetching token_keys", resp.StatusCode)
	}
	var verified struct {
		Keys []struct {
			Public string `json:"value"`
			Alg    string `json:"alg"`
		} `json:"keys"`
	}
	err = json.NewDecoder(resp.Body).Decode(&verified)
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(a.token, &UAAClaims{}, func(token *jwt.Token) (interface{}, error) {
		for _, key := range verified.Keys {
			if key.Alg == token.Header["alg"] {
				if key.Alg == "RS256" || key.Alg == "RS512" {
					return jwt.ParseRSAPublicKeyFromPEM([]byte(key.Public))
				}
			}
		}
		return nil, fmt.Errorf("no key found for", token.Header)
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token invalid")
	}
	claims, ok := token.Claims.(*UAAClaims)
	if !ok {
		return nil, fmt.Errorf("token claims type")
	}
	return claims.Scope, nil
}
