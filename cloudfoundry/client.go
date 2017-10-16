package cloudfoundry

import (
	"net/http"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// Client is a general Cloud Foundry API client interface
type Client interface {
	Get(path string) (*http.Response, error)
}

// ClientWrapper wraps the go-cfclient client with the Client interface
type ClientWrapper struct {
	cf *cfclient.Client
}

// NewClientWrapper creates a new Cloud Foundry API client wrapper
func NewClientWrapper(config *cfclient.Config) (*ClientWrapper, error) {
	client, err := cfclient.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ClientWrapper{cf: client}, nil
}

// Get does a GET requests against the API and returns the the result body
func (c ClientWrapper) Get(path string) (*http.Response, error) {
	req := c.cf.NewRequest("GET", path)
	return c.cf.DoRequest(req)
}
