// Copyright 2016 Compose, an IBM Company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package composeapi

import (
	"encoding/json"
)

//Datacenter structure
type Datacenter struct {
	Region   string `json:"region"`
	Provider string `json:"provider"`
	Slug     string `json:"slug"`
}

type datacentersResponse struct {
	Embedded struct {
		Datacenters []Datacenter `json:"datacenters"`
	} `json:"_embedded"`
}

//GetDatacentersJSON gets datacenters available as a string
func (c *Client) GetDatacentersJSON() (string, []error) {
	return c.getJSON("datacenters")
}

//GetDatacenters gets datacenters available as a Go struct
func (c *Client) GetDatacenters() (*[]Datacenter, []error) {
	body, errs := c.GetDatacentersJSON()

	if errs != nil {
		return nil, errs
	}

	datacenterResponse := datacentersResponse{}
	json.Unmarshal([]byte(body), &datacenterResponse)
	datacenters := datacenterResponse.Embedded.Datacenters

	return &datacenters, nil
}
