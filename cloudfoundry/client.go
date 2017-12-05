package cloudfoundry

import (
	"net/http"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// Client is a general Cloud Foundry API client interface
type Client interface {
	Get(path string) (*http.Response, error)
	GetApps() (map[string]cfclient.App, error)
	GetServiceInstances() (map[string]cfclient.ServiceInstance, error)
	GetOrgs() (map[string]cfclient.Org, error)
	GetSpaces() (map[string]cfclient.Space, error)
	GetServices() (map[string]cfclient.Service, error)
	GetServicePlans() (map[string]cfclient.ServicePlan, error)
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
func (c *client) Get(path string) (*http.Response, error) {
	req := c.cfClient.NewRequest("GET", path)
	return c.cfClient.DoRequest(req)
}

// GetApps fetches all the app instances
func (c *client) GetApps() (map[string]cfclient.App, error) {
	appMap := map[string]cfclient.App{}
	apps, err := c.cfClient.ListApps()
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		appMap[app.Guid] = app
	}
	return appMap, nil
}

// GetServiceInstances fetches all the service instances
func (c *client) GetServiceInstances() (map[string]cfclient.ServiceInstance, error) {
	serviceInstanceMap := map[string]cfclient.ServiceInstance{}
	serviceInstances, err := c.cfClient.ListServiceInstances()
	if err != nil {
		return nil, err
	}
	for _, si := range serviceInstances {
		serviceInstanceMap[si.Guid] = si
	}
	return serviceInstanceMap, nil
}

// GetOrgs fetches all the organisations
func (c *client) GetOrgs() (map[string]cfclient.Org, error) {
	orgMap := map[string]cfclient.Org{}
	orgs, err := c.cfClient.ListOrgs()
	if err != nil {
		return nil, err
	}
	for _, org := range orgs {
		orgMap[org.Guid] = org
	}
	return orgMap, nil
}

// GetSpaces fetches all the spaces
func (c *client) GetSpaces() (map[string]cfclient.Space, error) {
	spaceMap := map[string]cfclient.Space{}
	spaces, err := c.cfClient.ListSpaces()
	if err != nil {
		return nil, err
	}
	for _, space := range spaces {
		spaceMap[space.Guid] = space
	}
	return spaceMap, nil
}

// GetServicePlans fetches all the service plans
func (c *client) GetServicePlans() (map[string]cfclient.ServicePlan, error) {
	planMap := map[string]cfclient.ServicePlan{}
	plans, err := c.cfClient.ListServicePlans()
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		planMap[plan.Guid] = plan
	}
	return planMap, nil
}

// GetServices fetches all the services
func (c *client) GetServices() (map[string]cfclient.Service, error) {
	serviceMap := map[string]cfclient.Service{}
	services, err := c.cfClient.ListServices()
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		serviceMap[service.Guid] = service
	}
	return serviceMap, nil
}
