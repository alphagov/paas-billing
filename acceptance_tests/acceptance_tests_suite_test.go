package acceptance_tests_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var TestConfig AcceptanceTestConfig

type AcceptanceTestConfig struct {
	BillingAPIURL string
	BearerToken   string
}

func TestAcceptanceTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AcceptanceTests Suite")
}

var (
	billingAPIURL = os.Getenv("BILLING_API_URL")
	cfBearerToken = os.Getenv("CF_BEARER_TOKEN")
)

var _ = BeforeSuite(func() {
	Expect(billingAPIURL).ToNot(Equal(""), "Billing API was empty")
	Expect(cfBearerToken).ToNot(Equal(""), "Bearer token was empty")

	TestConfig = AcceptanceTestConfig{
		BillingAPIURL: billingAPIURL,
		BearerToken:   cfBearerToken,
	}
})
