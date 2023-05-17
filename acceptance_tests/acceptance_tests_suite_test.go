package acceptance_tests_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAcceptanceTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AcceptanceTests Suite")
}

var (
	BillingAPIURLFromEnv   = os.Getenv("BILLING_API_URL")
	CFBearerTokenFromEnv   = os.Getenv("CF_BEARER_TOKEN")
	MetricsProxyURLFromEnv = os.Getenv("METRICSPROXY_API_URL")
)
