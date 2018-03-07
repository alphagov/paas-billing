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

// Backup structure
type Backup struct {
	ID           string `json:"id"`
	Deploymentid string `json:"deployment_id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	DownloadLink string `json:"download_link"`
}

// backupsResponse is used to represent and remove the JSON+HAL Embedded wrapper
type backupsResponse struct {
	Embedded struct {
		Backups []Backup `json:"backups"`
	} `json:"_embedded"`
}

//GetBackupsForDeploymentJSON returns backup details for deployment
func (c *Client) GetBackupsForDeploymentJSON(deploymentid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid + "/backups")
}

//GetBackupsForDeployment returns backup details for deployment
func (c *Client) GetBackupsForDeployment(deploymentid string) (*[]Backup, []error) {
	body, errs := c.GetBackupsForDeploymentJSON(deploymentid)

	if errs != nil {
		return nil, errs
	}

	backupsResponse := backupsResponse{}
	json.Unmarshal([]byte(body), &backupsResponse)
	Backups := backupsResponse.Embedded.Backups

	return &Backups, nil
}

//StartBackupForDeploymentJSON starts backup and returns JSON response
func (c *Client) StartBackupForDeploymentJSON(deploymentid string) (string, []error) {
	response, body, errs := c.newRequest("POST", apibase+"deployments/"+deploymentid+"/backups").
		End()

	if response.StatusCode != 202 { // Expect Accepted on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

//StartBackupForDeployment starts backup and returns recipe
func (c *Client) StartBackupForDeployment(deploymentid string) (*Recipe, []error) {
	body, errs := c.StartBackupForDeploymentJSON(deploymentid)
	if errs != nil {
		return nil, errs
	}

	recipe := Recipe{}
	json.Unmarshal([]byte(body), &recipe)

	return &recipe, nil
}

//GetBackupDetailsForDeploymentJSON returns the details and download link for a backup
func (c *Client) GetBackupDetailsForDeploymentJSON(deploymentid string, backupid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid + "/backups/" + backupid)
}

//GetBackupDetailsForDeployment returns backup details for deployment
func (c *Client) GetBackupDetailsForDeployment(deploymentid string, backupid string) (*Backup, []error) {
	body, errs := c.GetBackupDetailsForDeploymentJSON(deploymentid, backupid)

	if errs != nil {
		return nil, errs
	}

	backup := Backup{}
	json.Unmarshal([]byte(body), &backup)

	return &backup, nil
}

//RestoreBackupParams Parameters to be completed before creating a deployment
type RestoreBackupParams struct {
	DeploymentID string
	BackupID     string
	Name         string
	ClusterID    string
	Datacenter   string
	Version      string
	SSL          bool
}

type restoreBackupParams struct {
	DeploymentID string                        `json:"-"`
	BackupID     string                        `json:"-"`
	Deployment   restoreBackupDeploymentParams `json:"deployment"`
}

type restoreBackupDeploymentParams struct {
	Name       string `json:"name"`
	ClusterID  string `json:"cluster_id,omitempty"`
	Datacenter string `json:"datacenter,omitempty"`
	Version    string `json:"version,omitempty"`
	SSL        bool   `json:"ssl,omitempty"`
}

//RestoreBackupJSON performs the call
func (c *Client) RestoreBackupJSON(params RestoreBackupParams) (string, []error) {
	backupparams := restoreBackupParams{
		DeploymentID: params.DeploymentID,
		BackupID:     params.BackupID,
		Deployment: restoreBackupDeploymentParams{Name: params.Name,
			ClusterID:  params.ClusterID,
			Datacenter: params.Datacenter,
			Version:    params.Version,
			SSL:        params.SSL,
		},
	}

	response, body, errs := c.newRequest("POST", apibase+"deployments/"+params.DeploymentID+"/backups/"+params.BackupID+"/restore").
		Send(backupparams).
		End()

	if response.StatusCode != 202 { // Expect Accepted on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

//RestoreBackup creates a deployment
func (c *Client) RestoreBackup(params RestoreBackupParams) (*Deployment, []error) {

	// This is a POST not a GET, so it builds its own request

	body, errs := c.RestoreBackupJSON(params)

	if errs != nil {
		return nil, errs
	}

	deployed := Deployment{}
	json.Unmarshal([]byte(body), &deployed)

	return &deployed, nil
}
