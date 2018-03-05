// Copyright 2017 Compose, an IBM Company
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
)

// TeamRole the name of the role and list of teams with that role for a
// deployment
type TeamRole struct {
	Name  string `json:"name"`
	Teams []Team `json:"teams"`
}

type updateTeamRole struct {
	TeamRole TeamRoleParams `json:"team_role"`
}

// TeamRoleParams the core parameters to create or delete a team role
type TeamRoleParams struct {
	Name   string `json:"name"`
	TeamID string `json:"team_id"`
}

type teamRolesResponse struct {
	Embedded struct {
		TeamRoles []TeamRole `json:"team_roles"`
	} `json:"_embedded"`
}

// CreateTeamRoleJSON performs the raw call to add a new team role
func (c *Client) CreateTeamRoleJSON(deploymentID string, params TeamRoleParams) (string, []error) {
	response, body, errs := c.newRequest("POST", teamRolesEndpoint(deploymentID)).
		Send(updateTeamRole{TeamRole: params}).
		End()

	if response.StatusCode != 201 {
		myErrors := Errors{}
		err := json.Unmarshal([]byte(body), &myErrors)
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to parse error - status code %d - body %s",
				response.StatusCode, body))
		} else {
			errs = append(errs, fmt.Errorf("%v", myErrors.Error))
		}
	}
	return body, errs
}

// CreateTeamRole adds a team role to a deployment
func (c *Client) CreateTeamRole(deploymentID string, params TeamRoleParams) (*TeamRole, []error) {
	body, errs := c.CreateTeamRoleJSON(deploymentID, params)
	if errs != nil {
		return nil, errs
	}

	teamRole := TeamRole{}
	json.Unmarshal([]byte(body), &teamRole)

	return &teamRole, nil
}

// GetTeamRolesJSON returns raw team roles
func (c *Client) GetTeamRolesJSON(deploymentID string) (string, []error) {
	return c.getJSON("deployments/" + deploymentID + "/team_roles")
}

// GetTeamRoles returns a slice of team roles for the given deployment
func (c *Client) GetTeamRoles(deploymentID string) (*[]TeamRole, []error) {
	body, errs := c.GetTeamRolesJSON(deploymentID)
	if errs != nil {
		return nil, errs
	}

	rolesResponse := teamRolesResponse{}
	json.Unmarshal([]byte(body), &rolesResponse)
	teamRoles := rolesResponse.Embedded.TeamRoles

	return &teamRoles, nil
}

// DeleteTeamRoleJSON is the raw call to remove a team_role
func (c *Client) DeleteTeamRoleJSON(deploymentID string, params TeamRoleParams) []error {
	response, body, errs := c.newRequest("DELETE", teamRolesEndpoint(deploymentID)).
		Send(updateTeamRole{TeamRole: params}).
		End()

	if response.StatusCode != 204 { // No response body is returned on success
		errs = ProcessErrors(response.StatusCode, body)
	}

	return errs
}

// DeleteTeamRole deletes a team role
func (c *Client) DeleteTeamRole(deploymentID string, params TeamRoleParams) []error {
	return c.DeleteTeamRoleJSON(deploymentID, params)
}

func teamRolesEndpoint(deploymentID string) string {
	return fmt.Sprintf("%sdeployments/%s/team_roles", apibase, deploymentID)
}
