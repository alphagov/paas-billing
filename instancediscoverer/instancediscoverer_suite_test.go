package instancediscoverer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestInstanceDiscoverer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CFAppDiscoverer")
}
