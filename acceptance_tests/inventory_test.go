package acceptance_tests_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/url"
	"time"
	"github.com/alphagov/paas-billing/eventio"
	"io"
  "encoding/json"
)

var _ = Describe("Acceptance", func() {
	It("Inventory can get billable events", func(){
		billingAPIURL, err := url.Parse(TestConfig.BillingAPIURL)
		Expect(err).ToNot(HaveOccurred())

		billingAPIURL.Path = "/billable_events"

		q := billingAPIURL.Query()
		q.Set("range_start", time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
		q.Set("range_stop", time.Now().Format("2006-01-02"))
		billingAPIURL.RawQuery = q.Encode()
		billingAPIURL.ForceQuery = true

		req, err := http.NewRequest("GET", billingAPIURL.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		headers := req.Header
		headers.Set("Authorization", fmt.Sprintf("Bearer %s", TestConfig.BearerToken))
		req.Header = headers

		client := &http.Client{}
		resp, err := client.Do(req)

		Expect(err).ToNot(HaveOccurred())

		Expect(resp.StatusCode).To(Equal(200))

		body, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		var billable_events []eventio.BillableEvent

		err = json.Unmarshal(body, &billable_events)
		Expect(err).ToNot(HaveOccurred())
		Expect(billable_events).ToNot(BeNil())



	})
})
