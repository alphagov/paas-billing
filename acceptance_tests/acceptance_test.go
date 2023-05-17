package acceptance_tests_test

import (
	"encoding/json"
	"fmt"
	"github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"net/http"
	"net/url"
	"time"
)

var (
	err error

	billingAPIURL   *url.URL
	metricsProxyURL *url.URL
)

var _ = Describe("Acceptance", func() {
	Context("Billing API", Label("smoke"), func() {
		BeforeEach(func() {
			Expect(BillingAPIURLFromEnv).ToNot(BeEmpty(), "Billing API was empty")
			billingAPIURL, err = url.Parse(BillingAPIURLFromEnv)
			Expect(err).ToNot(HaveOccurred())

			Expect(CFBearerTokenFromEnv).ToNot(BeEmpty(), "Bearer token was empty")
		})

		It("can get pricing plans from api", func() {

			billingAPIURL.Path = "/pricing_plans"

			q := billingAPIURL.Query()
			q.Set("range_start", time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
			q.Set("range_stop", time.Now().Format("2006-01-02"))
			billingAPIURL.RawQuery = q.Encode()
			billingAPIURL.ForceQuery = true

			resp, err := http.Get(billingAPIURL.String())
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(200))
		})

		It("can get billable events from api", func() {
			billingAPIURL.Path = "/billable_events"

			q := billingAPIURL.Query()
			q.Set("range_start", time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
			q.Set("range_stop", time.Now().Format("2006-01-02"))
			billingAPIURL.RawQuery = q.Encode()
			billingAPIURL.ForceQuery = true

			req, err := http.NewRequest("GET", billingAPIURL.String(), nil)
			Expect(err).ToNot(HaveOccurred())
			headers := req.Header
			headers.Set("Authorization", fmt.Sprintf("Bearer %s", CFBearerTokenFromEnv))
			req.Header = headers

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(200))
		})
	})

	Context("Metrics Proxy", func() {
		BeforeEach(func() {
			Expect(MetricsProxyURLFromEnv).ToNot(BeEmpty(), "Metrics proxy URL was empty")
			metricsProxyURL, err = url.Parse(MetricsProxyURLFromEnv)
			Expect(err).ToNot(HaveOccurred())
		})
		DescribeTable("metrics proxy can discover all billing apps and proxy to their metrics",
			func(billingAppName string) {
				metricsProxyURL.Path = fmt.Sprintf("/discovery/%s", billingAppName)

				resp, err := http.Get(metricsProxyURL.String())

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				var metricsTargets []apiserver.MetricsTarget
				err = json.Unmarshal(body, &metricsTargets)

				Expect(err).ToNot(HaveOccurred())

				Expect(metricsTargets).ToNot(BeEmpty())
				Expect(metricsTargets[0].Targets).ToNot(BeEmpty())

				for _, metricsTarget := range metricsTargets {
					proxyMetricsURL := &url.URL{
						Scheme: metricsProxyURL.Scheme, // Retrieve the scheme from the envar so it works locally
						Host:   metricsTarget.Targets[0],
						Path:   metricsTarget.Labels.MetricsPath,
					}
					resp, err := http.Get(proxyMetricsURL.String())
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					defer resp.Body.Close()

					body, err := io.ReadAll(resp.Body)
					Expect(err).ToNot(HaveOccurred())

					Expect(body).ToNot(BeEmpty())
					Expect(body).To(ContainSubstring("go_gc_duration_seconds"))
				}

			},
			Entry("billing API", "paas-billing-api"),
			Entry("billing collector", "paas-billing-collector"),
		)
	})
})
