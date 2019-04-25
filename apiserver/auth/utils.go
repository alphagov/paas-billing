package auth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
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
		return nil, fmt.Errorf("failed to request /v2/info: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("got status %d from /v2/info", resp.StatusCode)
	}
	var endpoints struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("failed to decode /v2/info json: %s", err)
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

// SliceMatches will iterate through both slices to find any incompatibilities in
// order to determine if the requested access is indeed allowed.
func SliceMatches(requested, allowed []string) (bool, string) {
	for _, r := range requested {
		if !inSlice(allowed, r) {
			return false, r
		}
	}

	return true, ""
}

func inSlice(slice []string, entry string) bool {
	for _, s := range slice {
		if s == entry {
			return true
		}
	}

	return false
}
