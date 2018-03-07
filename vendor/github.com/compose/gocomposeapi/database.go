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

//Version structure
type Version struct {
	Application string `json:"application"`
	Status      string `json:"status"`
	Preferred   bool   `json:"preferred"`
	Version     string `json:"version"`
}

//GetVersionsForDeploymentJSON returns raw JSON for getVersionsforDeployment
func (c *Client) GetVersionsForDeploymentJSON(deploymentid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid + "/versions")
}

//GetVersionsForDeployment gets deployment recipe life
func (c *Client) GetVersionsForDeployment(deploymentid string) (*[]VersionTransition, []error) {
	body, errs := c.GetVersionsForDeploymentJSON(deploymentid)

	if errs != nil {
		return nil, errs
	}

	versionsResponse := versionsResponse{}
	json.Unmarshal([]byte(body), &versionsResponse)
	versionTransitions := versionsResponse.Embedded.VersionTransitions

	return &versionTransitions, nil
}

//UpdateVersionJSON returns raw JSON as result of patching version
func (c *Client) UpdateVersionJSON(deploymentID string, version string) (string, []error) {
	patchParams := patchDeploymentVersionParams{
		Deployment: deploymentVersion{Version: version},
	}

	response, body, errs := c.newRequest("PATCH", apibase+"deployments/"+deploymentID+"/versions").
		Send(patchParams).
		End()

	if response.StatusCode != 200 { // Expect OK on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

// UpdateVersion returns Recipe for version update that is now taking progress
func (c *Client) UpdateVersion(deploymentID, version string) (*Recipe, []error) {
	body, errs := c.UpdateVersionJSON(deploymentID, version)
	if errs != nil {
		return nil, errs
	}

	recipe := Recipe{}
	json.Unmarshal([]byte(body), &recipe)

	return &recipe, nil
}

//Database structure
type Database struct {
	DatabaseType string `json:"type"`
	Status       string `json:"status"`
	Embedded     struct {
		Versions []Version `json:"versions"`
	} `json:"_embedded"`
}

type databasesResponse struct {
	Embedded struct {
		Databases []Database `json:"applications"`
	} `json:"_embedded"`
}

//GetDatabasesJSON gets databases available as a string
func (c *Client) GetDatabasesJSON() (string, []error) {
	return c.getJSON("databases")
}

//GetDatabases gets databases available as a Go struct
func (c *Client) GetDatabases() (*[]Database, []error) {
	body, errs := c.GetDatabasesJSON()

	if errs != nil {
		return nil, errs
	}

	databaseResponse := databasesResponse{}
	json.Unmarshal([]byte(body), &databaseResponse)
	databases := databaseResponse.Embedded.Databases

	return &databases, nil
}
