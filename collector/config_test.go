package collector_test

import (
	"os"
	"time"

	. "github.com/alphagov/paas-billing/collector"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Unsetenv("COLLECTOR_DEFAULT_SCHEDULE")
		os.Unsetenv("COLLECTOR_MIN_WAIT_TIME")
		os.Unsetenv("COLLECTOR_FETCH_LIMIT")
		os.Unsetenv("COLLECTOR_RECORD_MIN_AGE")
	})

	Context("When created from environment variables", func() {
		It("should read the environment variables correctly", func() {
			os.Setenv("COLLECTOR_DEFAULT_SCHEDULE", "1s")
			os.Setenv("COLLECTOR_MIN_WAIT_TIME", "2s")
			os.Setenv("COLLECTOR_FETCH_LIMIT", "3")
			os.Setenv("COLLECTOR_RECORD_MIN_AGE", "4s")

			config, err := CreateConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(config.DefaultSchedule).To(Equal(time.Duration(1 * time.Second)))
			Expect(config.MinWaitTime).To(Equal(time.Duration(2 * time.Second)))
			Expect(config.FetchLimit).To(Equal(3))
			Expect(config.RecordMinAge).To(Equal(time.Duration(4 * time.Second)))
		})

		It("should use default values if none of the env vars are set", func() {
			config, err := CreateConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(config.DefaultSchedule.Seconds()).To(BeNumerically(">", 0))
			Expect(config.MinWaitTime.Seconds()).To(BeNumerically(">", 0))
			Expect(config.FetchLimit).To(BeNumerically(">", 0))
			Expect(config.RecordMinAge.Seconds()).To(BeNumerically(">", 0))
		})

		It("should error if default schedule is invalid", func() {
			os.Setenv("COLLECTOR_DEFAULT_SCHEDULE", "invalid duration")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_DEFAULT_SCHEDULE is invalid"))
		})

		It("should error if the min wait time is invalid", func() {
			os.Setenv("COLLECTOR_MIN_WAIT_TIME", "invalid duration")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_MIN_WAIT_TIME is invalid"))
		})

		It("should error if the fetch limit is invalid", func() {
			os.Setenv("COLLECTOR_FETCH_LIMIT", "NaN")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_FETCH_LIMIT is invalid"))
		})

		It("should error if the fetch limit is negative", func() {
			os.Setenv("COLLECTOR_FETCH_LIMIT", "-1")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_FETCH_LIMIT must be between 1 and 100"))
		})

		It("should error if the fetch limit is greater than the max limit", func() {
			os.Setenv("COLLECTOR_FETCH_LIMIT", "101")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_FETCH_LIMIT must be between 1 and 100"))
		})

		It("should error if the record min age is invalid", func() {
			os.Setenv("COLLECTOR_RECORD_MIN_AGE", "invalid duration")
			_, err := CreateConfigFromEnv()
			Expect(err).To(MatchError("COLLECTOR_RECORD_MIN_AGE is invalid"))
		})
	})

})
