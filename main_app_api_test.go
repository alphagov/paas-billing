package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	"github.com/labstack/echo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = It("Should perform a smoke test against a real environment", func() {
	enabled := os.Getenv("ENABLE_SMOKE_TESTS")
	if enabled != "true" {
		Skip("smoke tests are disabled set ENABLE_SMOKE_TESTS=true to enable them")
	}

	var (
		err                 error
		session             *Session
		session_collector   *Session
		tempDB              *testenv.TempDB
		anOrgGUIDWithEvents string
		validAuthToken      = os.Getenv("TEST_AUTH_TOKEN")
		httpClient          = &http.Client{}
		port                = "8765"
	)

	By("Setting up a environment", func() {
		tempDB, err = testenv.New()
		Expect(err).ToNot(HaveOccurred())
		os.Setenv("DATABASE_URL", tempDB.TempConnectionString)
		os.Setenv("COLLECTOR_MIN_WAIT_TIME", "100ms")
		os.Setenv("PROCESSOR_SCHEDULE", "5s")
		os.Setenv("PORT", port)
	})

	defer By("Removing the temp database environment", func() {
		tempDB.Close()
	})

	By("Ensuring a TEST_AUTH_TOKEN environment variable is set", func() {
		Expect(validAuthToken).ToNot(BeEmpty(), "TEST_AUTH_TOKEN must be set to a valid cf oauth-token for this test")
	})

	By("Starting the app", func() {
		api := exec.Command(BinaryPath, "api")
		collector := exec.Command(BinaryPath, "collector")
		session, err = Start(api, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		session_collector, err = Start(collector, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session.Out, 10*time.Second).Should(Say("paas-billing.starting"))
	})

	defer By("Killing the app (if it hasn't already been shutdown)", func() {
		session.Kill()
		session_collector.Kill()
	})

	By("Waiting for the APIServer to report it has started", func() {
		Eventually(session.Out, 60*time.Second).Should(Say("paas-billing.api.started"))
	})

	By("Waiting for some raw events to be collected in the database", func() {
		Eventually(func() interface{} {
			return tempDB.Get(`select count(*) from app_usage_events`)
		}, 1*time.Minute).Should(BeNumerically(">", 0), "expected some app_usage_events to be collected")
	})

	By("Waiting for some events to be processed in the database", func() {
		Eventually(func() interface{} {
			return tempDB.Get(`select count(*) from events`)
		}, 5*time.Second).Should(BeNumerically(">", 0), "expected some events to processed")
		anOrgGUIDWithEvents = tempDB.Get(`select org_guid::text from events limit 1`).(string)
		Expect(anOrgGUIDWithEvents).ToNot(BeEmpty())
	})

	By("Expecting requests to /usage_events without a token to return status 401", func() {
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s/usage_events", port))
		Expect(err).ToNot(HaveOccurred())
		q := u.Query()
		q.Set("org_guid", anOrgGUIDWithEvents)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(echo.GET, u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := httpClient.Do(req)
		Expect(err).ToNot(HaveOccurred())

		Expect(res.StatusCode).To(Equal(401))
	})

	By("Expecting requests to /usage_events with a token to return some events", func() {
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s/usage_events", port))
		Expect(err).ToNot(HaveOccurred())
		q := u.Query()
		q.Set("org_guid", anOrgGUIDWithEvents)
		q.Set("range_start", "1970-01-01")
		q.Set("range_stop", "2030-01-01")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(echo.GET, u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("Authorization", validAuthToken)
		res, err := httpClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		data, err := ioutil.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(200))

		var usageEvents []eventio.UsageEvent
		err = json.Unmarshal(data, &usageEvents)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(usageEvents)).To(BeNumerically(">", 1), "expected at least 1 event to get processed by now")
		for _, ev := range usageEvents {
			Expect(ev.EventGUID).ToNot(BeEmpty())
		}
	})

	By("Expecting requests to /billable_events without a token to return status 401", func() {
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s/billable_events", port))
		Expect(err).ToNot(HaveOccurred())
		q := u.Query()
		q.Set("org_guid", anOrgGUIDWithEvents)
		q.Set("range_start", "2001-01-01")
		q.Set("range_stop", "2001-01-02")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(echo.GET, u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := httpClient.Do(req)
		Expect(err).ToNot(HaveOccurred())

		Expect(res.StatusCode).To(Equal(401))
	})

	By("Expecting requests to /billable_events with a token to return some events", func() {
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s/billable_events", port))
		Expect(err).ToNot(HaveOccurred())
		q := u.Query()
		q.Set("org_guid", anOrgGUIDWithEvents)
		q.Set("range_start", "1970-01-01")
		q.Set("range_stop", "2030-01-01")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(echo.GET, u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("Authorization", validAuthToken)
		res, err := httpClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		data, err := ioutil.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(200))

		var billableEvents []eventio.BillableEvent
		err = json.Unmarshal(data, &billableEvents)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(billableEvents)).To(BeNumerically(">", 0), "expected at least one billing event to be processed by now")
		for _, ev := range billableEvents {
			Expect(ev.EventGUID).ToNot(BeEmpty())
			Expect(ev.EventStart).ToNot(BeEmpty())
			Expect(ev.EventStop).ToNot(BeEmpty())
			Expect(ev.ResourceGUID).ToNot(BeEmpty())
			Expect(ev.ResourceType).ToNot(BeEmpty())
			Expect(ev.ResourceName).ToNot(BeEmpty())
			Expect(ev.PlanGUID).ToNot(BeEmpty())
			Expect(ev.Price.IncVAT).ToNot(BeEmpty())
			Expect(ev.Price.ExVAT).ToNot(BeEmpty())
			Expect(len(ev.Price.Details)).To(BeNumerically(">", 0))
		}
	})

	By("Expecting the /forecast_events calculator return events", func() {
		inputEventsJSON := `[{
			"event_guid": "00000000-0000-0000-0000-000000000001",
			"resource_guid": "00000000-0000-0000-0001-000000000001",
			"resource_name": "fake-app-1",
			"resource_type": "app",
			"org_guid": "` + eventstore.DummyOrgGUID + `",
			"space_guid": "` + eventstore.DummySpaceGUID + `",
			"event_start": "2001-01-01T00:00",
			"event_stop": "2001-02-01T00:00",
			"plan_guid": "` + eventstore.ComputePlanGUID + `",
			"number_of_nodes": 1,
			"memory_in_mb": 1024,
			"storage_in_mb": 1024
		}]`
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s/forecast_events", port))
		Expect(err).ToNot(HaveOccurred())
		q := u.Query()
		q.Set("org_guid", eventstore.DummyOrgGUID)
		q.Set("range_start", "1970-01-01")
		q.Set("range_stop", "2030-01-01")
		q.Set("events", inputEventsJSON)
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(echo.GET, u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := httpClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		data, err := ioutil.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(200))

		var billableEvents []eventio.BillableEvent
		err = json.Unmarshal(data, &billableEvents)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(billableEvents)).To(Equal(1))
		for _, ev := range billableEvents {
			Expect(ev.EventGUID).ToNot(BeEmpty())
			Expect(ev.EventStart).To(Equal("2001-01-01T00:00:00+00:00"))
			Expect(ev.EventStop).To(Equal("2001-02-01T00:00:00+00:00"))
			Expect(ev.Price.ExVAT).To(Equal("0.01"))
		}
	})

	By("Sending SIGTERM to the process", func() {
		session.Terminate()
		Eventually(session.Out, 10*time.Second).Should(Say("paas-billing.stopping"))
	})

	By("Waiting until the process exits cleanly", func() {
		Eventually(session, 60*time.Second).Should(Exit(0))
	})
})
