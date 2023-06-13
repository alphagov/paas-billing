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
	BillingAPIURLFromEnv            = os.Getenv("BILLING_API_URL")
	CFAdminBearerTokenFromEnv       = os.Getenv("CF_ADMIN_BEARER_TOKEN")
	CFNonAdminBearerTokenFromEnv    = os.Getenv("CF_NONADMIN_BEARER_TOKEN")
	CFNonAdminBillingManagerOrgGUID = os.Getenv("CF_NONADMIN_BILLING_MANAGER_ORG_GUID")
	MetricsProxyURLFromEnv          = os.Getenv("METRICSPROXY_API_URL")
)
