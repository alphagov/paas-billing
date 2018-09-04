package main

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Unsetenv("APP_ROOT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("COLLECTOR_SCHEDULE")
		os.Unsetenv("COLLECTOR_MIN_WAIT_TIME")
		os.Unsetenv("CF_FETCH_LIMIT")
		os.Unsetenv("CF_RECORD_MIN_AGE")
		os.Unsetenv("CF_API_ADDRESS")
		os.Unsetenv("CF_USERNAME")
		os.Unsetenv("CF_PASSWORD")
		os.Unsetenv("CF_CLIENT_ID")
		os.Unsetenv("CF_CLIENT_SECRET")
		os.Unsetenv("CF_SKIP_SSL_VALIDATION")
		os.Unsetenv("CF_TOKEN")
		os.Unsetenv("CF_USER_AGENT")
		os.Unsetenv("PROCESSOR_SCHEDULE")
		os.Unsetenv("PORT")
	})

	It("should set sensible defaults for the config when no environment variables set", func() {
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Logger).ToNot(BeNil())
		Expect(cfg.DatabaseURL).To(Equal("postgres://postgres:@localhost:5432/"))
		Expect(cfg.AppRootDir).To(Equal(getwd()))
		Expect(cfg.Collector.Schedule).To(Equal(15 * time.Minute))
		Expect(cfg.Collector.MinWaitTime).To(Equal(3 * time.Second))
		Expect(cfg.CFFetcher.RecordMinAge).To(Equal(10 * time.Minute))
		Expect(cfg.CFFetcher.FetchLimit).To(Equal(50))
		Expect(cfg.Processor.Schedule).To(Equal(30 * time.Minute))
		Expect(cfg.ServerPort).To(Equal(8881))
	})

	DescribeTable("should return error when failing to parse durations",
		func(variableName string) {
			os.Setenv(variableName, "bad-duration")
			_, err := NewConfigFromEnv()
			Expect(err).To(MatchError("time: invalid duration bad-duration"))
		},
		Entry("bad schedule", "COLLECTOR_SCHEDULE"),
		Entry("bad min wait time", "COLLECTOR_MIN_WAIT_TIME"),
		Entry("bad record min age", "CF_RECORD_MIN_AGE"),
		Entry("bad processor schedule", "PROCESSOR_SCHEDULE"),
	)

	DescribeTable("should return error when failing to parse integers",
		func(variableName string) {
			os.Setenv(variableName, "NaN")
			_, err := NewConfigFromEnv()
			Expect(err).To(MatchError(ContainSubstring("invalid syntax")))
		},
		Entry("bad cf fetch limit", "CF_FETCH_LIMIT"),
	)

	It("should set DatabaseURL from DATABASE_URL", func() {
		os.Setenv("DATABASE_URL", "postgres://test.database.local")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.DatabaseURL).To(Equal("postgres://test.database.local"))
	})

	It("should set Collector.Schedule from COLLECTOR_SCHEDULE", func() {
		os.Setenv("COLLECTOR_SCHEDULE", "50m")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Collector.Schedule).To(Equal(50 * time.Minute))
	})

	It("should set Collector.MinWaitTime from COLLECTOR_MIN_WAIT_TIME", func() {
		os.Setenv("COLLECTOR_MIN_WAIT_TIME", "6m")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Collector.MinWaitTime).To(Equal(6 * time.Minute))
	})

	It("should set CFFetcher.RecordMinAge from CF_RECORD_MIN_AGE", func() {
		os.Setenv("CF_RECORD_MIN_AGE", "4s")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.RecordMinAge).To(Equal(time.Duration(4 * time.Second)))
	})

	It("should set CFFetcher.FetchLimit from CF_FETCH_LIMIT", func() {
		os.Setenv("CF_FETCH_LIMIT", "30")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.FetchLimit).To(Equal(30))
	})

	It("should set CFFetcher.ClientConfig.ApiAddress from CF_API_ADDRESS", func() {
		os.Setenv("CF_API_ADDRESS", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.ApiAddress).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.Username from CF_USERNAME", func() {
		os.Setenv("CF_USERNAME", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.Username).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.Password from CF_PASSWORD", func() {
		os.Setenv("CF_PASSWORD", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.Password).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.ClientID from CF_CLIENT_ID", func() {
		os.Setenv("CF_CLIENT_ID", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.ClientID).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.ClientSecret from CF_CLIENT_SECRET", func() {
		os.Setenv("CF_CLIENT_SECRET", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.ClientSecret).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.SkipSslValidation from CF_SKIP_SSL_VALIDATION", func() {
		os.Setenv("CF_SKIP_SSL_VALIDATION", "true")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.SkipSslValidation).To(BeTrue())
	})

	It("should set CFFetcher.ClientConfig.Token from CF_TOKEN", func() {
		os.Setenv("CF_TOKEN", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.Token).To(Equal("set-in-test"))
	})

	It("should set CFFetcher.ClientConfig.UserAgent from CF_USER_AGENT", func() {
		os.Setenv("CF_USER_AGENT", "set-in-test")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.CFFetcher.ClientConfig.UserAgent).To(Equal("set-in-test"))
	})

	It("should set Processor.Schedule from PROCESSOR_SCHEDULE", func() {
		os.Setenv("PROCESSOR_SCHEDULE", "12h")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Processor.Schedule).To(Equal(12 * time.Hour))
	})

})
