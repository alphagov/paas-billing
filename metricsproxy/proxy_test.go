package metricsproxy_test

import (
	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-billing/metricsproxy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"net/url"
)

var _ = Describe("metricsProxy", func() {
	var (
		proxy MetricsProxy
	)

	BeforeEach(func() {
		proxy = New(Config{
			Logger: lager.NewLogger("proxy"),
		})
	})
	It("should correctly proxy the request to the remote server", func() {
		testCfHeader := map[string]string{"test-header": "test-value"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/metrics" {
				w.WriteHeader(http.StatusConflict)
				_, err := w.Write([]byte("It didn't work!"))
				Expect(err).ToNot(HaveOccurred())
				return
			}
			if r.URL.Path == "/metrics" {
				if r.Header.Get("test-header") != "test-value" {
					w.WriteHeader(400)
					return
				}
				w.Header().Add("some-header", "some-value")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("It works"))
				Expect(err).ToNot(HaveOccurred())
			}
		}))
		defer server.Close()

		req := &http.Request{Header: http.Header{}}
		res := httptest.NewRecorder()

		appURL, err := url.Parse(server.URL)
		Expect(err).ToNot(HaveOccurred())
		appURL.Path = "/metrics"

		proxy.ForwardRequestToURL(res, req, appURL, testCfHeader)

		Expect(res.Code).To(Equal(http.StatusOK))

		// check we get the body back from the remote server
		Expect(res.Body.Bytes()).To(ContainSubstring("It works"))

		// check that arbitrary headers are returned
		Expect(res.Header().Get("some-header")).To(Equal("some-value"))

		req = &http.Request{Header: http.Header{}}
		res = httptest.NewRecorder()

		appURL.Path = "/potato"
		proxy.ForwardRequestToURL(res, req, appURL, testCfHeader)

		Expect(res.Code).To(Equal(http.StatusConflict))

		Expect(len(res.Body.Bytes())).To(BeZero()) // test that we strip the body from non-200 responses

	})

})
