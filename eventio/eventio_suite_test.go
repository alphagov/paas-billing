package eventio_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEventio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Eventio Suite")
}
