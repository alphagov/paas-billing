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
	"fmt"
	"time"
)

// Deployment structure
type Deployment struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Type                string            `json:"type"`
	CreatedAt           time.Time         `json:"created_at"`
	ProvisionRecipeID   string            `json:"provision_recipe_id,omitempty"`
	CACertificateBase64 string            `json:"ca_certificate_base64,omitempty"`
	Connection          ConnectionStrings `json:"connection_strings,omitempty"`
	Notes               string            `json:"notes,omitempty"`
	CustomerBillingCode string            `json:"customer_billing_code,omitempty"`
	Version             string            `json:"version,omitempty"`
	ClusterID           string            `json:"cluster_id,omitempty"`
	Links               Links             `json:"_links"`
}

// Links structure, part of the Deployment struct
type Links struct {
	ComposeWebUILink Link `json:"compose_web_ui"`
	ScalingsLink     Link `json:"scalings"`
	BackupsLink      Link `json:"backups"`
	AlertsLink       Link `json:"alerts"`
	PortalUsersLink  Link `json:"portal_users"`
	ClusterLink      Link `json:"cluster"`
}

// ConnectionStrings structure, part of the Deployment struct
type ConnectionStrings struct {
	Health   []string            `json:"health,omitempty"`
	SSH      []string            `json:"ssh,omitempty"`
	Admin    []string            `json:"admin,omitempty"`
	SSHAdmin []string            `json:"ssh_admin,omitempty"`
	CLI      []string            `json:"cli,omitempty"`
	Direct   []string            `json:"direct,omitempty"`
	Maps     []map[string]string `json:"maps,omitempty"`
	Misc     interface{}         `json:"misc,omitempty"`
}

// deploymentsResource is used to represent and remove the JSON+HAL Embedded wrapper
type deploymentsResponse struct {
	Embedded struct {
		Deployments []Deployment `json:"deployments"`
	} `json:"_embedded"`
}

//CreateDeploymentParams Parameters to be completed before creating a deployment
type CreateDeploymentParams struct {
	Deployment DeploymentParams `json:"deployment"`
}

type patchDeployment struct {
	DeploymentID string                `json:"-"`
	Deployment   patchDeploymentParams `json:"deployment"`
}

type patchDeploymentParams struct {
	Notes               string `json:"notes,omitempty"`
	CustomerBillingCode string `json:"customer_billing_code,omitempty"`
}

// PatchDeploymentParams is used to pass parameters to PatchDeployment
type PatchDeploymentParams struct {
	DeploymentID        string `json:"omit"`
	Notes               string `json:"notes,omitempty"`
	CustomerBillingCode string `json:"customer_billing_code,omitempty"`
}

// DeploymentParams core parameters for a new deployment
type DeploymentParams struct {
	Name                string   `json:"name"`
	AccountID           string   `json:"account_id"`
	ClusterID           string   `json:"cluster_id,omitempty"`
	Datacenter          string   `json:"datacenter,omitempty"`
	ProvisioningTags    []string `json:"provisioning_tags,omitempty"`
	DatabaseType        string   `json:"type"`
	Version             string   `json:"version,omitempty"`
	Units               int      `json:"units,omitempty"`
	SSL                 bool     `json:"ssl,omitempty"`
	CacheMode           bool     `json:"cache_mode,omitempty"`
	WiredTiger          bool     `json:"wired_tiger,omitempty"`
	Notes               string   `json:"notes,omitempty"`
	CustomerBillingCode string   `json:"customer_billing_code,omitempty"`
}

//VersionTransition a struct wrapper for version transition information
type VersionTransition struct {
	Application string `json:"application"`
	Method      string `json:"method"`
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

type versionsResponse struct {
	Embedded struct {
		VersionTransitions []VersionTransition `json:"transitions"`
	} `json:"_embedded"`
}

type patchDeploymentVersionParams struct {
	Deployment deploymentVersion `json:"deployment"`
}

type deploymentVersion struct {
	Version string `json:"version"`
}

//CreateDeploymentJSON performs the call
func (c *Client) CreateDeploymentJSON(params DeploymentParams) (string, []error) {
	deploymentparams := CreateDeploymentParams{Deployment: params}

	response, body, errs := c.newRequest("POST", apibase+"deployments").
		Send(deploymentparams).
		End()

	if response.StatusCode != 202 { // Expect Accepted on success - assume error on anything else
		myerrors := Errors{}
		err := json.Unmarshal([]byte(body), &myerrors)
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to parse error - status code %d - body %s", response.StatusCode, response.Body))
		} else {
			errs = append(errs, fmt.Errorf("%v", myerrors.Error))
		}
	}

	return body, errs
}

//CreateDeployment creates a deployment
func (c *Client) CreateDeployment(params DeploymentParams) (*Deployment, []error) {

	// This is a POST not a GET, so it builds its own request

	body, errs := c.CreateDeploymentJSON(params)

	if errs != nil {
		return nil, errs
	}

	deployed := Deployment{}
	json.Unmarshal([]byte(body), &deployed)

	return &deployed, nil
}

//GetDeploymentsJSON returns raw deployment
func (c *Client) GetDeploymentsJSON() (string, []error) { return c.getJSON("deployments") }

//GetDeployments returns deployment structure
func (c *Client) GetDeployments() (*[]Deployment, []error) {
	body, errs := c.GetDeploymentsJSON()

	if errs != nil {
		return nil, errs
	}

	deploymentResponse := deploymentsResponse{}
	json.Unmarshal([]byte(body), &deploymentResponse)
	deployments := deploymentResponse.Embedded.Deployments

	return &deployments, nil
}

//GetDeploymentJSON returns raw deployment
func (c *Client) GetDeploymentJSON(deploymentid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid)
}

//GetDeployment returns deployment structure
func (c *Client) GetDeployment(deploymentid string) (*Deployment, []error) {
	body, errs := c.GetDeploymentJSON(deploymentid)

	if errs != nil {
		return nil, errs
	}

	deployment := Deployment{}
	json.Unmarshal([]byte(body), &deployment)

	return &deployment, nil
}

//GetDeploymentByName returns a deployment of a given name
func (c *Client) GetDeploymentByName(deploymentName string) (*Deployment, []error) {
	deployments, errs := c.GetDeployments()
	if errs != nil {
		return nil, errs
	}

	for _, deployment := range *deployments {
		if deployment.Name == deploymentName {
			return c.GetDeployment(deployment.ID)
		}
	}

	return nil, []error{fmt.Errorf("deployment not found: %s", deploymentName)}
}

//DeprovisionDeploymentJSON performs the call
func (c *Client) DeprovisionDeploymentJSON(deploymentID string) (string, []error) {

	response, body, errs := c.newRequest("DELETE", apibase+"deployments/"+deploymentID).
		End()

	if response.StatusCode != 202 { // Expect Accepted on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

//DeprovisionDeployment deletes a deployment
func (c *Client) DeprovisionDeployment(deploymentID string) (*Recipe, []error) {

	// This is a POST not a GET, so it builds its own request

	body, errs := c.DeprovisionDeploymentJSON(deploymentID)

	if errs != nil {
		return nil, errs
	}

	deprovrecipe := Recipe{}
	json.Unmarshal([]byte(body), &deprovrecipe)

	return &deprovrecipe, nil
}

//PatchDeploymentJSON performs the call
func (c *Client) PatchDeploymentJSON(params PatchDeploymentParams) (string, []error) {

	patchParams := patchDeployment{DeploymentID: params.DeploymentID,
		Deployment: patchDeploymentParams{
			CustomerBillingCode: params.CustomerBillingCode,
			Notes:               params.Notes,
		}}

	response, body, errs := c.newRequest("PATCH", apibase+"deployments/"+patchParams.DeploymentID).
		Send(patchParams).
		End()

	if response.StatusCode != 200 { // Expect Accepted on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

//PatchDeployment patches a deployment
func (c *Client) PatchDeployment(params PatchDeploymentParams) (*Deployment, []error) {

	// This is a POST not a GET, so it builds its own request

	body, errs := c.PatchDeploymentJSON(params)

	if errs != nil {
		return nil, errs
	}

	deployed := Deployment{}
	json.Unmarshal([]byte(body), &deployed)

	return &deployed, nil
}
