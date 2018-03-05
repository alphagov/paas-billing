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

type auditEventsResponse struct {
	Embedded struct {
		AuditEvents []AuditEvent `json:"audit_events"`
	} `json:"_embedded"`
}

// AuditEvent is the returned information for a single Compose audit event
type AuditEvent struct {
	Links        Links             `json:"_links"`
	AccountID    string            `json:"account_id"`
	ClusterID    string            `json:"cluster_id,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	Data         map[string]string `json:"data"`
	DeploymentID string            `json:"deployment_id,omitempty"`
	Email        string            `json:"email,omitempty"`
	Event        string            `json:"event"`
	ID           string            `json:"id"`
	IP           string            `json:"ip"`
	UserAgent    string            `json:"user_agent"`
	UserID       string            `json:"user_id"`
}

// AuditEventsParams is the structure of entirely optional options for
// filtering or paging through audit events
type AuditEventsParams struct {
	OlderThan *time.Time `json:"older_than,omitempty"`
	NewerThan *time.Time `json:"newer_than,omitempty"`
	Cursor    string     `json:"cursor,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// GetAuditEventsJSON performs the call
func (c *Client) GetAuditEventsJSON(params AuditEventsParams) (string, []error) {
	response, body, errs := c.newRequest("GET", apibase+"audit_events").
		Query(params).
		End()

	if response.StatusCode != 200 { // Expect Accepted on success - assume error on anything else
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

// GetAuditEvents returns all audit_events the API Key has access to.
func (c *Client) GetAuditEvents(params AuditEventsParams) (*[]AuditEvent, []error) {
	body, errs := c.GetAuditEventsJSON(params)
	if len(errs) != 0 {
		return nil, errs
	}

	response := auditEventsResponse{}
	json.Unmarshal([]byte(body), &response)
	return &response.Embedded.AuditEvents, nil
}

// GetAuditEventJSON performs the call
func (c *Client) GetAuditEventJSON(id string) (string, []error) {
	response, body, errs := c.newRequest("GET", apibase+"/audit_events/"+id).
		End()

	if response.StatusCode != 200 { // Expect Accepted on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

// GetAuditEvent returns the specified audit_event
func (c *Client) GetAuditEvent(id string) (*AuditEvent, []error) {
	body, errs := c.GetAuditEventJSON(id)
	if len(errs) != 0 {
		return nil, errs
	}

	auditEvent := AuditEvent{}
	json.Unmarshal([]byte(body), &auditEvent)
	return &auditEvent, nil
}
