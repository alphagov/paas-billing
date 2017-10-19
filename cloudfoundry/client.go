package cloudfoundry

import (
	"net/http"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// Client is a general Cloud Foundry API client interface
type Client interface {
	Get(path string) (*http.Response, error)
}

// client wraps the go-cfclient as the NewRequest and DoRequest methods use unexported structs
// therefore we can't mock the client for tests
type client struct {
	cfClient *cfclient.Client
}

// NewClient creates a new Cloud Foundry API client
func NewClient(config *cfclient.Config) (Client, error) {
	cfClient, err := cfclient.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &client{cfClient: cfClient}, nil
}

// Get does a GET requests against the API and returns the the result body
func (c client) Get(path string) (*http.Response, error) {
	req := c.cfClient.NewRequest("GET", path)
	return c.cfClient.DoRequest(req)
}
