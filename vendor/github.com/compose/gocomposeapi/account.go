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

// Account structure
type Account struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// AccountParams is the list of parameters that can be updated on an existing
// account
type AccountParams struct {
	StripeCustomerID string `json:"stripe_customer_id"`
}

type updateAccount struct {
	Account AccountParams `json:"account"`
}

type accountResponse struct {
	Embedded struct {
		Accounts []Account `json:"accounts"`
	} `json:"_embedded"`
}

type accountUsersResponse struct {
	Embedded struct {
		Users []User `json:"users"`
	} `json:"_embedded"`
}

//GetAccountJSON gets JSON string from endpoint
func (c *Client) GetAccountJSON() (string, []error) { return c.getJSON("accounts") }

//GetAccount Gets first Account struct from account endpoint
func (c *Client) GetAccount() (*Account, []error) {
	body, errs := c.GetAccountJSON()

	if errs != nil {
		return nil, errs
	}

	accountsResponse := accountResponse{}
	json.Unmarshal([]byte(body), &accountsResponse)
	firstAccount := accountsResponse.Embedded.Accounts[0]

	return &firstAccount, nil
}

//GetAccountUsersJSON gets the JSON string from the users endpoint for this account
func (c *Client) GetAccountUsersJSON() (string, []error) {
	account, errs := c.GetAccount()
	if errs != nil {
		return "", errs
	}
	return c.getJSON("accounts/" + account.ID + "/users")
}

//GetAccountUsers gets the user array for the current account
func (c *Client) GetAccountUsers() ([]User, []error) {
	body, errs := c.GetAccountUsersJSON()

	if errs != nil {
		return nil, errs
	}

	accountUsersResponse := accountUsersResponse{}
	json.Unmarshal([]byte(body), &accountUsersResponse)

	return accountUsersResponse.Embedded.Users, nil
}

// CreateAccountUserJSON performs the call
func (c *Client) CreateAccountUserJSON(accountID string, params UserParams) (string, []error) {
	response, body, errs := c.newRequest("POST",
		apibase+"accounts/"+accountID+"/users").
		Send(params).
		End()

	if response.StatusCode != 201 { // Expect Created on success
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

// CreateAccountUser adds a new user to an account and returns a User object on success
func (c *Client) CreateAccountUser(accountID string, params UserParams) (*User, []error) {
	body, errs := c.CreateAccountUserJSON(accountID, params)
	if len(errs) != 0 {
		return nil, errs
	}

	user := User{}
	json.Unmarshal([]byte(body), &user)
	return &user, nil
}

// DeleteAccountUserJSON performs the call
func (c *Client) DeleteAccountUserJSON(accountID, userID string) (string, []error) {
	response, body, errs := c.newRequest("DELETE",
		apibase+"accounts/"+accountID+"/users/"+userID).
		End()

	if response.StatusCode != 200 { // Expect OK on success
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

// DeleteAccountUser removes a user from the provided account
func (c *Client) DeleteAccountUser(accountID, userID string) (*User, []error) {
	body, errs := c.DeleteAccountUserJSON(accountID, userID)
	if len(errs) != 0 {
		return nil, errs
	}

	user := User{}
	json.Unmarshal([]byte(body), &user)
	return &user, nil
}
