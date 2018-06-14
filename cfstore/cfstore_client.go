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

type Service struct {
	Guid              string   `json:"guid"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	Label             string   `json:"label"`
	Description       string   `json:"description"`
	Active            bool     `json:"active"`
	Bindable          bool     `json:"bindable"`
	ServiceBrokerGuid string   `json:"service_broker_guid"`
	PlanUpdateable    bool     `json:"plan_updateable"`
	Tags              []string `json:"tags"`
}

type CFDataClient interface {
	ListServicePlans() ([]ServicePlan, error)
	ListServices() ([]Service, error)
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
			servicePlan := ServicePlan{}
			servicePlan.Guid = servicePlanResource.Meta.Guid
			servicePlan.CreatedAt = servicePlanResource.Meta.CreatedAt
			servicePlan.UpdatedAt = servicePlanResource.Meta.UpdatedAt
			servicePlan.Free = servicePlanResource.Entity.Free
			servicePlan.Name = servicePlanResource.Entity.Name
			servicePlan.Description = servicePlanResource.Entity.Description
			servicePlan.ServiceGuid = servicePlanResource.Entity.ServiceGuid
			servicePlan.Extra = servicePlanResource.Entity.Extra
			servicePlan.UniqueId = servicePlanResource.Entity.UniqueId
			servicePlan.Public = servicePlanResource.Entity.Public
			servicePlan.Active = servicePlanResource.Entity.Active
			servicePlan.Bindable = servicePlanResource.Entity.Bindable
			servicePlans = append(servicePlans, servicePlan)
		}
		requestUrl = servicePlansResp.NextUrl
		if requestUrl == "" {
			break
		}
	}
	return servicePlans, nil
}

func (c *Client) ListServices() ([]Service, error) {
	var services []Service
	requestUrl := "/v2/services"
	for {
		var servicesResp cfclient.ServicesResponse
		r := c.Client.NewRequest("GET", requestUrl)
		resp, err := c.Client.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error requesting services")
		}
		resBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading services request:")
		}
		err = json.Unmarshal(resBody, &servicesResp)
		if err != nil {
			return nil, errors.Wrap(err, "Error unmarshaling services")
		}
		for _, serviceResource := range servicesResp.Resources {
			service := Service{}
			service.Guid = serviceResource.Meta.Guid
			service.CreatedAt = serviceResource.Meta.CreatedAt
			service.UpdatedAt = serviceResource.Meta.UpdatedAt
			service.Label = serviceResource.Entity.Label
			service.Description = serviceResource.Entity.Description
			service.Active = serviceResource.Entity.Active
			service.Bindable = serviceResource.Entity.Bindable
			service.PlanUpdateable = serviceResource.Entity.PlanUpdateable
			service.ServiceBrokerGuid = serviceResource.Entity.ServiceBrokerGuid
			service.Tags = serviceResource.Entity.Tags
			services = append(services, service)
		}
		requestUrl = servicesResp.NextUrl
		if requestUrl == "" {
			break
		}
	}
	return services, nil
}
