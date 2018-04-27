package cffetcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
)

// UsageEventsClient is a general Cloud Foundry API client interface
type UsageEventsClient interface {
	Get(path string) (*http.Response, error)
}

// client wraps the go-cfclient as the NewRequest and DoRequest methods use unexported structs
// therefore we can't mock the client for tests
type client struct {
	cfClient *cfclient.Client
}

// Get does a GET requests against the API and returns the the result body
func (c *client) Get(path string) (*http.Response, error) {
	req := c.cfClient.NewRequest("GET", path)
	return c.cfClient.DoRequest(req)
}

// MetaData contains the record metadata like id and creation date
type MetaData struct {
	GUID      string    `json:"guid"`
	CreatedAt time.Time `json:"created_at"`
}

// UsageEventList contains usage event records
type UsageEventList struct {
	Resources []UsageEvent `json:"resources"`
}

// UsageEvent represent a usage event record from the API
type UsageEvent struct {
	MetaData  MetaData
	EntityRaw json.RawMessage `json:"entity"`
}

// GUIDNil represents an empty GUID
const GUIDNil = "GUID_NIL"

const appType = "app"
const serviceType = "service"

// UsageEventsAPI is a common interface for the app and service usage event APIs
type UsageEventsAPI interface {
	Get(afterGUID string, count int, minAge time.Duration) (*UsageEventList, error)
	Type() string
}

// UsageEventsAPI is a CloudFoundry API client for getting usage events
type usageEventsAPI struct {
	eventType string
	client    UsageEventsClient
	logger    lager.Logger
}

// NewAppUsageEventsAPI returns with a new app usage events API client
func NewAppUsageEventsAPI(client UsageEventsClient, logger lager.Logger) UsageEventsAPI {
	return &usageEventsAPI{
		client:    client,
		eventType: appType,
		logger:    logger,
	}
}

// NewServiceUsageEventsAPI returns with a new service usage events API client
func NewServiceUsageEventsAPI(client UsageEventsClient, logger lager.Logger) UsageEventsAPI {
	return &usageEventsAPI{
		client:    client,
		eventType: serviceType,
		logger:    logger,
	}
}

func (u *usageEventsAPI) doRequest(path string, target interface{}) error {
	u.logger.Debug("fetching", lager.Data{
		"path": path,
	})

	resp, err := u.client.Get(path)
	if err != nil {
		return errors.Wrapf(err, "error fetching %s", path)
	}

	defer resp.Body.Close()
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "error reading %s body", path)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s request failed: %d %s", path, resp.StatusCode, resBody)
	}

	err = json.Unmarshal(resBody, target)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling %s", path)
	}

	return nil
}

// Get returns with the usage events or an error on failure
func (u *usageEventsAPI) Get(afterGUID string, count int, minAge time.Duration) (*UsageEventList, error) {
	if afterGUID == "" {
		panic("afterGUID parameter should not be empty")
	}

	url := fmt.Sprintf("/v2/%s_usage_events?results-per-page=%d", u.eventType, count)
	if afterGUID != GUIDNil {
		url = url + fmt.Sprintf("&after_guid=%s", afterGUID)
	}

	res := &UsageEventList{}
	if err := u.doRequest(url, res); err != nil {
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

// Type returns with the client type
func (u *usageEventsAPI) Type() string {
	return u.eventType
}
