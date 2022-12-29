package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"context"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo/v2"
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
		tempDB              *testenv.TempDB
		anOrgGUIDWithEvents string
		validAuthToken      = os.Getenv("TEST_AUTH_TOKEN")
		//httpClient          = &http.Client{}
		port = "8765"
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
		cmd := exec.Command(BinaryPath, "collector")
		cmd.Env = os.Environ()
		session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session.Out, 10*time.Second).Should(Say("paas-billing.starting"))
	})

	defer By("Killing the app (if it hasn't already been shutdown)", func() {
		session.Kill()
	})

	By("Waiting for the EventStore to report it has been initialized", func() {
		Eventually(session.Out, 20*time.Second).Should(Say("paas-billing.store.initializing"))
		Eventually(session.Out, 60*time.Second).Should(Say("paas-billing.store.initialized"))
	})

	By("Waiting for the HistoricDataStore to report it has been initialized", func() {
		Eventually(session.Out, 60*time.Second).Should(Say("paas-billing.historic-data-store.initialized"))
	})

	By("Ensuring Service/ServicePlan data exists after HistoricDataStore.Init", func() {
		Expect(
			tempDB.Get(`select count(*) from service_plans`),
		).To(BeNumerically(">", 0), "expected some service_plans to be collected during init")
		Expect(
			tempDB.Get(`select count(*) from services`),
		).To(BeNumerically(">", 0), "expected some services to be collected during init")
	})

	By("Waiting for some raw events to be collected in the database", func() {
		Eventually(func() interface{} {
			return tempDB.Get(`select count(*) from app_usage_events`)
		}, 1*time.Minute).Should(BeNumerically(">", 0), "expected some app_usage_events to be collected")
	})

	By("Waiting for some events to be processed in the database", func() {
		Eventually(func() interface{} {
			return tempDB.Get(`select count(*) from events`)
		}, 5*time.Minute).Should(BeNumerically(">", 0), "expected some events to processed")
		anOrgGUIDWithEvents = tempDB.Get(`select org_guid::text from events limit 1`).(string)
		Expect(anOrgGUIDWithEvents).ToNot(BeEmpty())
	})

	By("Sending SIGTERM to the process", func() {
		session.Terminate()
		Eventually(session.Out, 10*time.Second).Should(Say("paas-billing.stopping"))
	})

	By("Waiting until the process exits cleanly", func() {
		Eventually(session, 60*time.Second).Should(Exit(0))
	})
})

var _ = Describe("runRefreshAndConsolidateLoop", func() {
	var (
		fakeStore *fakes.FakeEventStore
		logger    lager.Logger
	)

	BeforeEach(func() {
		fakeStore = &fakes.FakeEventStore{}
		logger = lager.NewLogger("test")
	})

	It("should call Refresh and Consolidate every 'Schedule'", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		wg := sync.WaitGroup{}
		defer wg.Wait()
		defer cancel()

		go func() {
			wg.Add(1)
			runRefreshAndConsolidateLoop(ctx, logger, 1*time.Nanosecond, fakeStore)
			wg.Done()
		}()

		Eventually(func() int {
			return fakeStore.RefreshCallCount()
		}).Should(BeNumerically(">=", 1))

		Eventually(func() int {
			return fakeStore.ConsolidateAllCallCount()
		}).Should(BeNumerically(">=", 1))
	})

	It("should call Refresh and Consolidate once initially before 'Schedule'", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		wg := sync.WaitGroup{}
		defer wg.Wait()
		defer cancel()

		go func() {
			wg.Add(1)
			runRefreshAndConsolidateLoop(ctx, logger, 5*time.Second, fakeStore)
			wg.Done()
		}()

		Eventually(func() int {
			return fakeStore.RefreshCallCount()
		}).Should(Equal(1))

		Eventually(func() int {
			return fakeStore.ConsolidateAllCallCount()
		}).Should(Equal(1))
	})

	It("should not call Consolidate if Refresh fails", func() {
		fakeStore.RefreshReturns(fmt.Errorf("some-error"))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		wg := sync.WaitGroup{}
		defer wg.Wait()
		defer cancel()

		go func() {
			wg.Add(1)
			runRefreshAndConsolidateLoop(ctx, logger, 1*time.Nanosecond, fakeStore)
			wg.Done()
		}()

		Eventually(func() int {
			return fakeStore.RefreshCallCount()
		}).Should(BeNumerically(">=", 2))

		Consistently(func() int {
			return fakeStore.ConsolidateAllCallCount()
		}).Should(BeNumerically("==", 0))
	})
})
