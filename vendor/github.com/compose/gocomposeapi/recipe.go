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
	"time"
)

// Recipe structure
type Recipe struct {
	ID           string    `json:"id"`
	Template     string    `json:"template"`
	Status       string    `json:"status"`
	StatusDetail string    `json:"status_detail"`
	AccountID    string    `json:"account_id"`
	DeploymentID string    `json:"deployment_id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Embedded     struct {
		Recipes []Recipe `json:"recipes"`
	} `json:"_embedded"`
}

type recipeResponse struct {
	Embedded struct {
		Recipes []Recipe `json:"recipes"`
	} `json:"_embedded"`
}

//GetRecipeJSON Gets raw JSON for recipeid
func (c *Client) GetRecipeJSON(recipeid string) (string, []error) {
	return c.getJSON("recipes/" + recipeid)
}

//GetRecipe gets status of Recipe
func (c *Client) GetRecipe(recipeid string) (*Recipe, []error) {
	body, errs := c.GetRecipeJSON(recipeid)

	if errs != nil {
		return nil, errs
	}

	recipe := Recipe{}
	json.Unmarshal([]byte(body), &recipe)

	return &recipe, nil
}

//GetRecipesForDeploymentJSON returns raw JSON for getRecipesforDeployment
func (c *Client) GetRecipesForDeploymentJSON(deploymentid string) (string, []error) {
	return c.getJSON("deployments/" + deploymentid + "/recipes")
}

//GetRecipesForDeployment gets deployment recipe life
func (c *Client) GetRecipesForDeployment(deploymentid string) (*[]Recipe, []error) {
	body, errs := c.GetRecipesForDeploymentJSON(deploymentid)

	if errs != nil {
		return nil, errs
	}

	recipeResponse := Recipe{}
	json.Unmarshal([]byte(body), &recipeResponse)
	recipes := recipeResponse.Embedded.Recipes

	return &recipes, nil
}
