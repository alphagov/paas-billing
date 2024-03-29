package cfstore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/cloudfoundry-community/go-cfclient"
)

//counterfeiter:generate . CFDataClient
type CFDataClient interface {
	ListServicePlans() ([]cfclient.ServicePlan, error)
	ListServices() ([]cfclient.Service, error)
	ListOrgs() ([]V3Org, error)
	ListSpaces() ([]cfclient.Space, error)
}

var _ CFDataClient = &Client{}

type V3OrgMetadataAnnotations struct {
	Owner string `json:"owner"`
}

type V3OrgMetadata struct {
	Annotations V3OrgMetadataAnnotations `json:"annotations"`
}

type V3OrgRelationshipsQuotaData struct {
	Guid string `json:"guid"`
}

type V3OrgRelationshipsQuota struct {
	Data V3OrgRelationshipsQuotaData `json:"data"`
}

type V3OrgRelationships struct {
	Quota V3OrgRelationshipsQuota `json:"quota"`
}

type V3Org struct {
	Guid          string             `json:"guid"`
	Name          string             `json:"name"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
	Metadata      V3OrgMetadata      `json:"metadata"`
	Relationships V3OrgRelationships `json:"relationships"`
}

type Pagination struct {
	TotalResults int `json:"total_results"`
	TotalPages   int `json:"total_pages"`
	First        struct {
		Href string `json:"href"`
	} `json:"first"`
	Last struct {
		Href string `json:"href"`
	} `json:"last"`
	Next struct {
		Href string `json:"href"`
	} `json:"next"`
	Previous struct {
		Href string `json:"href"`
	} `json:"previous"`
}

type V3OrgsResponse struct {
	Pagination Pagination `json:"pagination"`
	Resources  []V3Org    `json:"resources"`
}

type Client struct {
	Client *cfclient.Client
}

func (c *Client) ListServicePlans() ([]cfclient.ServicePlan, error) {
	return c.Client.ListServicePlans()
}

func (c *Client) ListServices() ([]cfclient.Service, error) {
	return c.Client.ListServices()
}

func (c *Client) ListOrgs() ([]V3Org, error) {
	var orgs []V3Org
	requestQuery := "/v3/organizations?"
	for {
		req := c.Client.NewRequest("GET", requestQuery)
		res, err := c.Client.DoRequest(req)
		if err != nil {
			return nil, fmt.Errorf("unable to interact with v3 api gathering orgs: %v", err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read out body into bytes array: %v", err)
		}

		var orgsRes V3OrgsResponse
		if err = json.Unmarshal(body, &orgsRes); err != nil {
			return nil, fmt.Errorf("unable to unmarshal json response into the struct: %v", err)
		}

		for _, org := range orgsRes.Resources {
			orgs = append(orgs, org)
		}

		requestUrl := orgsRes.Pagination.Next.Href
		if requestUrl == "" {
			break
		}

		urlParsed, err := url.Parse(requestUrl)
		if err != nil {
			return nil, fmt.Errorf("unable to parse pagination URL: %v", err)
		}

		requestQuery = urlParsed.Path + "?" + urlParsed.RawQuery
	}

	return orgs, nil
}

func (c *Client) ListSpaces() ([]cfclient.Space, error) {
	return c.Client.ListSpaces()
}
