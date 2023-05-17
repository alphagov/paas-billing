package apiserver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"testing"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIServer")
}

var _ = BeforeEach(func() {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
})
