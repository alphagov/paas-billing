package auth

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"

	"github.com/labstack/echo"
)

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true"},
		},
	}
}

func CreateConfigFromEnv() (*oauth2.Config, error) {
	apiEndpoint := os.Getenv("CF_API_ADDRESS")
	if apiEndpoint == "" {
		return nil, fmt.Errorf("CF_API_ADDRESS environment variable required")
	}
	httpClient := newHTTPClient()
	resp, err := httpClient.Get(apiEndpoint + "/v2/info")
	if err != nil {
		return nil, fmt.Errorf("failed to request /v2/info", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("got status %d from /v2/info", resp.StatusCode)
	}
	var endpoints struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("failed to decode /v2/info json", err)
	}
	return &oauth2.Config{
		ClientID:     os.Getenv("CF_CLIENT_ID"),
		ClientSecret: os.Getenv("CF_CLIENT_SECRET"),
		Scopes:       []string{"openid", "cloud_controller.admin_read_only", "cloud_controller.read", "cloud_controller.global_auditor", "cloud_controller.admin"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthorizationEndpoint + "/oauth/authorize",
			TokenURL: endpoints.TokenEndpoint + "/oauth/token",
		},
		RedirectURL: os.Getenv("CF_CLIENT_REDIRECT_URL"),
	}, nil
}

func getTokenFromRequest(c echo.Context) (string, error) {
	if t := c.Request().Header.Get(echo.HeaderAuthorization); t != "" {
		parts := strings.Split(t, " ")
		if len(parts) != 2 {
			return "", errors.New("invalid Authorization header")
		}
		if strings.ToLower(parts[0]) != "bearer" {
			return "", errors.New("unsupported Authorization header type")
		}
		if parts[1] == "" {
			return "", errors.New("missing Authorization Bearer token data")
		}
		return parts[1], nil
	} else if cookie, err := c.Cookie(CookieAuthorization); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	return "", errors.New("no access_token in request")
}
