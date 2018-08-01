package cfstore

import (
	"encoding/json"
	"io/ioutil"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
)

type ServicePlan struct {
	Name                string      `json:"name"`
	Guid                string      `json:"guid"`
	CreatedAt           string      `json:"created_at"`
	UpdatedAt           string      `json:"updated_at"`
	Free                bool        `json:"free"`
	Description         string      `json:"description"`
	ServiceGuid         string      `json:"service_guid"`
	Extra               interface{} `json:"extra"`
	UniqueId            string      `json:"unique_id"`
	Public              bool        `json:"public"`
	Active              bool        `json:"active"`
	Bindable            bool        `json:"bindable"`
	ServiceUrl          string      `json:"service_url"`
	ServiceInstancesUrl string      `json:"service_instances_url"`
}

type CFDataClient interface {
	ListServicePlans() ([]ServicePlan, error)
	ListServices() ([]cfclient.Service, error)
	ListOrgs() ([]cfclient.Org, error)
	ListSpaces() ([]cfclient.Space, error)
}

var _ CFDataClient = &Client{}

type Client struct {
	Client *cfclient.Client
}

func (c *Client) ListServicePlans() ([]ServicePlan, error) {
	var servicePlans []ServicePlan
	requestUrl := "/v2/service_plans"
	for {
		var servicePlansResp cfclient.ServicePlansResponse
		r := c.Client.NewRequest("GET", requestUrl)
		resp, err := c.Client.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error requesting service plans")
		}
		resBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading service plans request:")
		}
		err = json.Unmarshal(resBody, &servicePlansResp)
		if err != nil {
			return nil, errors.Wrap(err, "Error unmarshaling service plans")
		}
		for _, servicePlanResource := range servicePlansResp.Resources {
			servicePlans = append(servicePlans, ServicePlan{
				Guid:        servicePlanResource.Meta.Guid,
				CreatedAt:   servicePlanResource.Meta.CreatedAt,
				UpdatedAt:   servicePlanResource.Meta.UpdatedAt,
				Free:        servicePlanResource.Entity.Free,
				Name:        servicePlanResource.Entity.Name,
				Description: servicePlanResource.Entity.Description,
				ServiceGuid: servicePlanResource.Entity.ServiceGuid,
				Extra:       servicePlanResource.Entity.Extra,
				UniqueId:    servicePlanResource.Entity.UniqueId,
				Public:      servicePlanResource.Entity.Public,
				Active:      servicePlanResource.Entity.Active,
				Bindable:    servicePlanResource.Entity.Bindable,
			})
		}
		requestUrl = servicePlansResp.NextUrl
		if requestUrl == "" {
			break
		}
	}
	return servicePlans, nil
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
