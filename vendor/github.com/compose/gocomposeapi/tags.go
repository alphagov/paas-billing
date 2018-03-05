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

type clusterTags struct {
	ClusterTags clusterTagList `json:"cluster"`
}

type clusterTagList struct {
	Tags []string `json:"tags"`
}

// AddTagsToClusterJSON returns raw result of call to add tags cluster endpoint
func (c *Client) AddTagsToClusterJSON(clusterID string, tags []string) (string, []error) {
	return c.updateClusterTagsJSON(clusterID, "POST", tags)
}

// AddTagsToCluster returns cluster data of cluster to which tahs have been added
func (c *Client) AddTagsToCluster(clusterID string, tags []string) (*Cluster, []error) {
	return c.updateClusterTags(clusterID, "POST", tags)
}

// DeleteTagsFromClusterJSON returns raw result of call to delete tags cluster endpoint
func (c *Client) DeleteTagsFromClusterJSON(clusterID string, tags []string) (string, []error) {
	return c.updateClusterTagsJSON(clusterID, "DELETE", tags)
}

// DeleteTagsFromCluster returns cluster data after deleting tags from cluster
func (c *Client) DeleteTagsFromCluster(clusterID string, tags []string) (*Cluster, []error) {
	return c.updateClusterTags(clusterID, "DELETE", tags)
}

// ReplaceTagsOnClusterJSON returns raw result of call to replace tags cluster endpoint
func (c *Client) ReplaceTagsOnClusterJSON(clusterID string, tags []string) (string, []error) {
	return c.updateClusterTagsJSON(clusterID, "PATCH", tags)
}

// ReplaceTagsOnCluster returns cluster data after replacing tags on cluster
func (c *Client) ReplaceTagsOnCluster(clusterID string, tags []string) (*Cluster, []error) {
	return c.updateClusterTags(clusterID, "PATCH", tags)
}

func (c *Client) updateClusterTagsJSON(clusterID, method string, tags []string) (string, []error) {

	response, body, errs := c.newRequest(method, tagsEndpoint(clusterID)).
		Send(clusterTags{ClusterTags: clusterTagList{Tags: tags}}).
		End()

	if response.StatusCode != 200 { // Expect OK on success - assume error on anything else
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

func (c *Client) updateClusterTags(clusterID, method string, tags []string) (*Cluster, []error) {
	body, errs := c.updateClusterTagsJSON(clusterID, method, tags)
	if errs != nil {
		return nil, errs
	}

	response := Cluster{}
	json.Unmarshal([]byte(body), &response)
	return &response, nil
}

func tagsEndpoint(clusterID string) string {
	return fmt.Sprintf("%sclusters/%s/tags", apibase, clusterID)
}
