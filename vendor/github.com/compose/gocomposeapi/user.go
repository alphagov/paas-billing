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

// User structure
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserParams structure
type UserParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

//GetUserJSON returns user JSON string
func (c *Client) GetUserJSON() (string, []error) {
	return c.getJSON("user")
}

//GetUser Gets information about user
func (c *Client) GetUser() (*User, []error) {
	body, errs := c.GetUserJSON()

	if errs != nil {
		return nil, errs
	}

	user := User{}
	json.Unmarshal([]byte(body), &user)
	return &user, nil
}
