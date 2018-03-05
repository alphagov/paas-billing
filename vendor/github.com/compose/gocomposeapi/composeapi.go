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
//

// Package composeapi provides an idiomatic Go wrapper around the Compose
// API for database platform for deployment, management and monitoring.
package composeapi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"
)

const (
	apibase = "https://api.compose.io/2016-07/"
)

// Client is a structure that holds session information for the API
type Client struct {
	// The number of times to retry a failing request if the status code is
	// retryable (e.g. for HTTP 429 or 500)
	Retries int
	// The interval to wait between retries. gorequest does not yet support
	// exponential back-off on retries
	RetryInterval time.Duration
	// RetryStatusCodes is the list of status codes to retry for
	RetryStatusCodes []int

	apiToken      string
	logger        *log.Logger
	enableLogging bool
}

// NewClient returns a Client for further interaction with the API
func NewClient(apiToken string) (*Client, error) {
	return &Client{
		apiToken:      apiToken,
		logger:        log.New(ioutil.Discard, "", 0),
		Retries:       5,
		RetryInterval: 3 * time.Second,
		RetryStatusCodes: []int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}, nil
}

// SetLogger can enable or disable http logging to and from the Compose
// API endpoint using the provided io.Writer for the provided client.
func (c *Client) SetLogger(enableLogging bool, logger io.Writer) *Client {
	c.logger = log.New(logger, "[composeapi]", log.LstdFlags)
	c.enableLogging = enableLogging
	return c
}

func (c *Client) newRequest(method, targetURL string) *gorequest.SuperAgent {
	return gorequest.New().
		CustomMethod(method, targetURL).
		Set("Authorization", "Bearer "+c.apiToken).
		Set("Content-type", "application/json; charset=utf-8").
		SetLogger(c.logger).
		SetDebug(c.enableLogging).
		SetCurlCommand(c.enableLogging).
		Retry(c.Retries, c.RetryInterval, c.RetryStatusCodes...)
}

// Link structure for JSON+HAL links
type Link struct {
	HREF      string `json:"href"`
	Templated bool   `json:"templated"`
}

//Errors struct for parsing error returns
type Errors struct {
	Error map[string][]string `json:"errors,omitempty"`
}

//SimpleError struct for parsing simple error returns
type SimpleError struct {
	Error string `json:"errors"`
}

func printJSON(jsontext string) {
	var tempholder map[string]interface{}

	if err := json.Unmarshal([]byte(jsontext), &tempholder); err != nil {
		log.Fatal(err)
	}
	indentedjson, err := json.MarshalIndent(tempholder, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(indentedjson))
}

//SetAPIToken overrides the API token
func (c *Client) SetAPIToken(newtoken string) {
	c.apiToken = newtoken
}

//GetJSON Gets JSON string of content at an endpoint
func (c *Client) getJSON(endpoint string) (string, []error) {
	response, body, errs := c.newRequest("GET", apibase+endpoint).End()

	if response.StatusCode != 200 {
		errs = ProcessErrors(response.StatusCode, body)
	}

	return body, errs
}

//ProcessErrors tries to turn errors into an Errors struct
func ProcessErrors(statuscode int, body string) []error {
	errs := []error{}
	myerrors := Errors{}
	err := json.Unmarshal([]byte(body), &myerrors)
	// Did parsing like this break anything
	if err != nil {
		mysimpleerror := SimpleError{}
		err := json.Unmarshal([]byte(body), &mysimpleerror)
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to parse error - status code %d - body %s", statuscode, body))
		} else {
			errs = append(errs, fmt.Errorf("%s", mysimpleerror.Error))
		}
	} else {
		// Todo: iterate through and add eachg error.
		errs = append(errs, fmt.Errorf("%v", myerrors.Error))
	}

	return errs
}
