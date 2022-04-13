package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type listV3ServiceInstancesResponse struct {
	Pagination cfclient.Pagination          `json:"pagination,omitempty"`
	Resources  []cfclient.V3ServiceInstance `json:"resources,omitempty"`
}

type booleanSetOperationResults struct {
	OnlyInFirstSet  map[string]bool
	OnlyInSecondSet map[string]bool
	SetIntersection map[string]bool
}

func get_boolean_results(firstSet map[string]bool, secondSet map[string]bool) booleanSetOperationResults {
	var results booleanSetOperationResults
	results.OnlyInFirstSet = make(map[string]bool)
	results.OnlyInSecondSet = make(map[string]bool)
	results.SetIntersection = make(map[string]bool)

	for k, _ := range firstSet {
		if secondSet[k] {
			results.SetIntersection[k] = true
		} else {
			results.OnlyInFirstSet[k] = true
		}
	}
	if len(results.SetIntersection) != len(secondSet) {
		for k, _ := range secondSet {
			if !firstSet[k] {
				results.OnlyInSecondSet[k] = true
			}
		}
	}
	return results

}
func join_guids(guids map[string]bool) string {

	var keys []string

	for k, _ := range guids {
		if k != "" {
			keys = append(keys, k)
		}
	}
	return (strings.Join(keys, ", "))
}

func get_service_instances(url string, req *http.Request) listV3ServiceInstancesResponse {
	fmt.Println(url)
	var err error 
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
        // Do the request to the cf api
	cfHTTPClient := &http.Client{}
	resp, err := cfHTTPClient.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var serviceInstances listV3ServiceInstancesResponse

	err = json.Unmarshal([]byte(body), &serviceInstances)
	if err != nil {
		fmt.Println(err)
	}

        return serviceInstances

}

func main() {
	//Do the billing api request
	billingAPIURL, err := url.Parse(os.Getenv("BILLING_API_URL"))
	if err != nil {
		fmt.Println(err)
	}

	billingAPIURL.Path = "/billable_events"

	q := billingAPIURL.Query()
	q.Set("range_start", time.Now().AddDate(0, 0, -2).Format("2006-01-02"))
	q.Set("range_stop", time.Now().Format("2006-01-02"))
	billingAPIURL.RawQuery = q.Encode()
	billingAPIURL.ForceQuery = true

	req, err := http.NewRequest("GET", billingAPIURL.String(), nil)
	if err != nil {
		fmt.Println(err)
	}
	headers := req.Header
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("CF_BEARER_TOKEN")))
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var billableEvents []eventio.BillableEvent

	err = json.Unmarshal([]byte(body), &billableEvents)
	if err != nil {
		fmt.Println(err)
	}
	billing_service_instance_guids := map[string]bool{}
	cf_service_instance_guids := map[string]bool{}

	for _, event := range(billableEvents) {
		// Insert all services found in billing into a map that can be compared later
		if event.ResourceType == "service" {
			billing_service_instance_guids[event.ResourceGUID] = true
		}
	}


	cfAPIURL, err := url.Parse(os.Getenv("CF_API_URL"))
	if err != nil {
		fmt.Println(err)
	}

	cfAPIURL.Path = "/v3/service_instances"

	q = cfAPIURL.Query()
	//Set up the query for cf things that have been created for four hours to avoid smoke tests and similar that won't have been picked up by billing yet
	q.Set("per_page", "1000")
	q.Set("created_ats[lt]", time.Now().Add(time.Hour * -4).Format("2006-01-02T15:04:05Z"))
	cfAPIURL.RawQuery = q.Encode()
	cfAPIURL.ForceQuery = true

	
        serviceInstances := get_service_instances(cfAPIURL.String(), req)

	err = json.Unmarshal([]byte(body), &serviceInstances)
	for len(serviceInstances.Resources) != 0 {
		for _, resource := range serviceInstances.Resources {
			//We do not currently charge for user defined services, so filter for service_instances_with_plans
			_, found_service_plan := resource.Relationships["service_plan"]

			// Insert all services found in cf into a map that can be compared later
			if found_service_plan {

				cf_service_instance_guids[resource.Guid] = true

			}
		}

		serviceInstances = get_service_instances(serviceInstances.Pagination.Next.Href, req)
	}

	fmt.Println("The number of service instances in cf: ", len(cf_service_instance_guids))
	fmt.Println("The number of service instances in paas-billing: ", len(billing_service_instance_guids))

	//Find the intersection between the two sets of service guids and also the sets of guids in one but not the other
	boolean_results := get_boolean_results(cf_service_instance_guids, billing_service_instance_guids)

	if len(boolean_results.OnlyInFirstSet) > 0 {
		fmt.Println("We found the following service_instance with these guids in cloudfoundry and not in billing: ", join_guids(boolean_results.OnlyInFirstSet))
		fmt.Println("This shouldn't happen, check that the billing collector app in the billing space is running happily")
		os.Exit(1)
	}
	if float64(len(boolean_results.OnlyInFirstSet))/float64(len(boolean_results.OnlyInFirstSet)+len(boolean_results.SetIntersection)) > 0.05 {

		fmt.Println("We found the following service_instance with these guids in billing and not cloudfoundry: ", join_guids(boolean_results.OnlyInSecondSet))
		fmt.Println("This is over 5 percent of all services, which is the current threshold to worry about. A number of services are short running and so are in billing and not in cloudfoundry, so some are to be expected")
		os.Exit(1)

	}
	
	if len (boolean_results.OnlyInSecondSet) !=0  {
		fmt.Println("We found the following guids in billing and not cloudfoundry :", join_guids(boolean_results.OnlyInSecondSet))
		fmt.Println("Nothing to worry about at the moment, but worth keeping an eye on")
	} else {
		fmt.Println("Both apis return the same services, so it is all good.")
	}
	
	os.Exit(0)
}
