package main

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Unsetenv("APP_ROOT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("COLLECTOR_SCHEDULE")
		os.Unsetenv("COLLECTOR_MIN_WAIT_TIME")
		os.Unsetenv("DB_CONN_MAX_IDLE_TIME")
		os.Unsetenv("DB_CONN_MAX_LIFETIME")
		os.Unsetenv("DB_MAX_IDLE_CONNS")
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
		os.Unsetenv("LISTEN_HOST")
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
		Expect(cfg.Processor.Schedule).To(Equal(720 * time.Minute))
		Expect(cfg.ServerPort).To(Equal(8881))
		Expect(cfg.DBConnMaxIdleTime).To(Equal(10*time.Minute))
		Expect(cfg.DBConnMaxLifetime).To(Equal(1*time.Hour))
		Expect(cfg.DBMaxIdleConns).To(Equal(1))
		Expect(cfg.ServerHost).To(Equal(""))
		Expect(cfg.ListenAddr).To(Equal(fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)))
	})

	DescribeTable("should return error when failing to parse durations",
		func(variableName string) {
			os.Setenv(variableName, "bad-duration")
			_, err := NewConfigFromEnv()
			Expect(err).To(MatchError("time: invalid duration \"bad-duration\""))
		},
		Entry("bad schedule", "COLLECTOR_SCHEDULE"),
		Entry("bad min wait time", "COLLECTOR_MIN_WAIT_TIME"),
		Entry("bad record min age", "CF_RECORD_MIN_AGE"),
		Entry("bad processor schedule", "PROCESSOR_SCHEDULE"),
		Entry("bad db conn max idle time", "DB_CONN_MAX_IDLE_TIME"),
		Entry("bad db conn max lifetime", "DB_CONN_MAX_LIFETIME"),
	)

	DescribeTable("should return error when failing to parse integers",
		func(variableName string) {
			os.Setenv(variableName, "NaN")
			_, err := NewConfigFromEnv()
			Expect(err).To(MatchError(ContainSubstring("invalid syntax")))
		},
		Entry("bad cf fetch limit", "CF_FETCH_LIMIT"),
		Entry("bad max idle conns", "DB_MAX_IDLE_CONNS"),
		Entry("bad ServerPort", "PORT"),
	)

	It("should set DatabaseURL from DATABASE_URL", func() {
		os.Setenv("DATABASE_URL", "postgres://test.database.local")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.DatabaseURL).To(Equal("postgres://test.database.local"))
	})

	It("should set DBConnMaxIdleTime from DB_CONN_MAX_IDLE_TIME", func() {
		os.Setenv("DB_CONN_MAX_IDLE_TIME", "50m")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.DBConnMaxIdleTime).To(Equal(50 * time.Minute))
	})

	It("should set DBConnMaxLifetime from DB_CONN_MAX_LIFETIME", func() {
		os.Setenv("DB_CONN_MAX_LIFETIME", "50m")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.DBConnMaxLifetime).To(Equal(50 * time.Minute))
	})

	It("should set DBMaxIdleConns from DB_MAX_IDLE_CONNS", func() {
		os.Setenv("DB_MAX_IDLE_CONNS", "5")
		cfg, err := NewConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.DBMaxIdleConns).To(Equal(5))
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

	Describe("cfg.AppRootDir should be set correctly", func() {
		Context("if $PWD is set and is not empty", func() {
			It("should be set to the value of $PWD", func() {
				os.Setenv("PWD", "/tmp")
				cfg, err := NewConfigFromEnv()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.AppRootDir).To(Equal("/tmp"))
			})
		})
		Context("if $PWD is set and is an empty string", func() {
			It("should be set to the value of os.Getwd()", func() {
				os.Setenv("PWD", "")
				cfg, err := NewConfigFromEnv()
				Expect(err).ToNot(HaveOccurred())
				osWd, _ := os.Getwd()
				Expect(cfg.AppRootDir).To(Equal(osWd))
			})
		})
		Context("if $PWD is unset", func() {
			It("should be set to the value of os.Getwd()", func() {
				os.Unsetenv("PWD")
				cfg, err := NewConfigFromEnv()
				Expect(err).ToNot(HaveOccurred())
				osWd, _ := os.Getwd()
				Expect(cfg.AppRootDir).To(Equal(osWd))
			})
		})
	})

})
