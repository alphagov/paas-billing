package cloudfoundry_test

import (
	"os"

	. "github.com/alphagov/paas-usage-events-collector/cloudfoundry"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	BeforeEach(func() {
		os.Setenv("CF_API_ADDRESS", "api_address_1")
		os.Setenv("CF_USERNAME", "username_1")
		os.Setenv("CF_PASSWORD", "password_1")
		os.Setenv("CF_CLIENT_ID", "client_id_1")
		os.Setenv("CF_CLIENT_SECRET", "client_secret_1")
		os.Setenv("CF_SKIP_SSL_VALIDATION", "true")
		os.Setenv("CF_TOKEN", "token_1")
		os.Setenv("CF_USER_AGENT", "user_agent_1")
	})

	AfterEach(func() {
		os.Unsetenv("CF_API_ADDRESS")
		os.Unsetenv("CF_USERNAME")
		os.Unsetenv("CF_PASSWORD")
		os.Unsetenv("CF_CLIENT_ID")
		os.Unsetenv("CF_CLIENT_SECRET")
		os.Unsetenv("CF_SKIP_SSL_VALIDATION")
		os.Unsetenv("CF_TOKEN")
		os.Unsetenv("CF_USER_AGENT")
	})

	Context("When created from environment variables", func() {
		It("should read the values correctly", func() {
			config := CreateConfigFromEnv()
			Expect(config.ApiAddress).To(Equal("api_address_1"))
			Expect(config.Username).To(Equal("username_1"))
			Expect(config.Password).To(Equal("password_1"))
			Expect(config.ClientID).To(Equal("client_id_1"))
			Expect(config.ClientSecret).To(Equal("client_secret_1"))
			Expect(config.SkipSslValidation).To(BeTrue())
			Expect(config.Token).To(Equal("token_1"))
			Expect(config.UserAgent).To(Equal("user_agent_1"))
		})
	})

	Context("When CF_SKIP_SSL_VALIDATION is not set to true", func() {
		It("should set SkipSslValidation to false", func() {
			os.Setenv("CF_SKIP_SSL_VALIDATION", "foo")
			config := CreateConfigFromEnv()
			Expect(config.SkipSslValidation).To(BeFalse())
		})
	})

})
