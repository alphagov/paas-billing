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
			services = append(services, Service{
				Guid:              serviceResource.Meta.Guid,
				CreatedAt:         serviceResource.Meta.CreatedAt,
				UpdatedAt:         serviceResource.Meta.UpdatedAt,
				Label:             serviceResource.Entity.Label,
				Description:       serviceResource.Entity.Description,
				Active:            serviceResource.Entity.Active,
				Bindable:          serviceResource.Entity.Bindable,
				PlanUpdateable:    serviceResource.Entity.PlanUpdateable,
				ServiceBrokerGuid: serviceResource.Entity.ServiceBrokerGuid,
				Tags:              serviceResource.Entity.Tags,
			})
		}
		requestUrl = servicesResp.NextUrl
		if requestUrl == "" {
			break
		}
	}
	return services, nil
}
