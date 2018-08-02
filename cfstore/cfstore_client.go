package cfstore

import (
	"github.com/cloudfoundry-community/go-cfclient"
)

type CFDataClient interface {
	ListServicePlans() ([]cfclient.ServicePlan, error)
	ListServices() ([]cfclient.Service, error)
	ListOrgs() ([]cfclient.Org, error)
	ListSpaces() ([]cfclient.Space, error)
}

var _ CFDataClient = &Client{}

type Client struct {
	Client *cfclient.Client
}

func (c *Client) ListServicePlans() ([]cfclient.ServicePlan, error) {
	return c.Client.ListServicePlans()
}

func (c *Client) ListServices() ([]cfclient.Service, error) {
	return c.Client.ListServices()
}

func (c *Client) ListOrgs() ([]cfclient.Org, error) {
	return c.Client.ListOrgs()
}

func (c *Client) ListSpaces() ([]cfclient.Space, error) {
	return c.Client.ListSpaces()
}
