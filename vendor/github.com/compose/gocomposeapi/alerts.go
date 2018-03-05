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

//Alert type which holds each Alert
type Alert struct {
	CapsuleID    string `json:"capsule_id"`
	DeploymentID string `json:"deployment_id"`
	Message      string `json:"message"`
	Status       string `json:"status"`
}

//Alerts type which contains all the Alerts and a summary
type Alerts struct {
	Summary  string `json:"summary"`
	Embedded struct {
		Alerts []Alert `json:"alerts"`
	} `json:"_embedded"`
}

//GetAlertsForDeploymentJSON returns raw JSON for getAlertsforDeployment
func (c *Client) GetAlertsForDeploymentJSON(deploymentid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid + "/alerts")
}

//GetAlertsForDeployment gets deployment recipe life
func (c *Client) GetAlertsForDeployment(deploymentid string) (*Alerts, []error) {
	body, errs := c.GetAlertsForDeploymentJSON(deploymentid)

	if errs != nil {
		return nil, errs
	}

	Alerts := Alerts{}
	json.Unmarshal([]byte(body), &Alerts)

	return &Alerts, nil
}
