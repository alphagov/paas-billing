package cloudfoundry

import (
	"net/http"
	"os"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// CreateConfigFromEnv creates a go-cfclient config from envrironment variables
func CreateConfigFromEnv() *cfclient.Config {
	return &cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Username:          os.Getenv("CF_USERNAME"),
		Password:          os.Getenv("CF_PASSWORD"),
		ClientID:          os.Getenv("CF_CLIENT_ID"),
		ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
		Token:             os.Getenv("CF_TOKEN"),
		UserAgent:         os.Getenv("CF_USER_AGENT"),
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
