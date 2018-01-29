package collector_test

import (
	"os"

	. "github.com/alphagov/paas-billing/collector"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	BeforeEach(func() {
		os.Setenv("COLLECTOR_DEFAULT_SCHEDULE", "1s")
		os.Setenv("COLLECTOR_MIN_WAIT_TIME", "2s")
		os.Setenv("COLLECTOR_FETCH_LIMIT", "3")
		os.Setenv("COLLECTOR_RECORD_MIN_AGE", "4s")
	})

	AfterEach(func() {
		os.Unsetenv("COLLECTOR_DEFAULT_SCHEDULE")
		os.Unsetenv("COLLECTOR_MIN_WAIT_TIME")
		os.Unsetenv("COLLECTOR_FETCH_LIMIT")
		os.Unsetenv("COLLECTOR_RECORD_MIN_AGE")
	})

	Context("When created from environment variables", func() {
		It("should read the environment variables correctly", func() {
			config := CreateConfigFromEnv()
			Expect(config.DefaultSchedule).To(Equal("1s"))
			Expect(config.MinWaitTime).To(Equal("2s"))
			Expect(config.FetchLimit).To(Equal("3"))
			Expect(config.RecordMinAge).To(Equal("4s"))
		})
	})

})
