package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
)

// GUIDNil represents an empty GUID
const GUIDNil = "GUID_NIL"

// Client is a CloudFoundry client
type Client struct {
	cf     *cfclient.Client
	logger lager.Logger
}

// NewClient creates a new CloudFoundry client
func NewClient(cfClient *cfclient.Client, logger lager.Logger) *Client {
	return &Client{cf: cfClient, logger: logger}
}

// Get queries the CloudFoundry API, parses the response JSON and returns with the result object
func (c *Client) Get(path string, target interface{}) error {
	c.logger.Debug("fetching", lager.Data{
		"path": path,
	})
	req := c.cf.NewRequest("GET", path)
	resp, err := c.cf.DoRequest(req)
	if err != nil {
		return errors.Wrapf(err, "error fetching %s", path)
	}
	defer resp.Body.Close()
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "error reading %s body", path)
	}
	err = json.Unmarshal(resBody, target)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling %s", path)
	}
	return nil
}

// GetAppUsageEvents returns with the app usage events or an error on failure
func (c *Client) GetAppUsageEvents(afterGUID string, count int, minAge time.Duration) (*AppUsageEventList, error) {
	if afterGUID == "" {
		panic("afterGUID parameter should not be empty")
	}

	url := fmt.Sprintf("/v2/app_usage_events?results-per-page=%d", count)
	if afterGUID != GUIDNil {
		url = url + fmt.Sprintf("&after_guid=%s", afterGUID)
	}

	res := &AppUsageEventList{}
	if err := c.Get(url, res); err != nil {
		return nil, err
	}

	t := time.Now().Add(-minAge)
	for i, record := range res.Resources {
		if record.MetaData.CreatedAt.After(t) {
			res.Resources = res.Resources[0:i]
			break
		}
	}

	return res, nil
}

// GetServiceUsageEvents returns with the service usage events or an error on failure
func (c *Client) GetServiceUsageEvents(afterGUID string, count int, minAge time.Duration) (*ServiceUsageEventList, error) {
	if afterGUID == "" {
		panic("afterGUID parameter should not be empty")
	}

	url := fmt.Sprintf("/v2/service_usage_events?results-per-page=%d", count)
	if afterGUID != GUIDNil {
		url = url + fmt.Sprintf("&after_guid=%s", afterGUID)
	}

	res := &ServiceUsageEventList{}
	if err := c.Get(url, res); err != nil {
		return nil, err
	}

	t := time.Now().Add(-minAge)
	for i, record := range res.Resources {
		if record.MetaData.CreatedAt.After(t) {
			res.Resources = res.Resources[0:i]
			break
		}
	}

	return res, nil
}
