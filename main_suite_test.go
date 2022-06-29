package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var BinaryPath string

var _ = BeforeSuite(func() {
	var err error
	BinaryPath, err = gexec.Build("github.com/alphagov/paas-billing")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.Kill()
	gexec.CleanupBuildArtifacts()
})

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main")
}
