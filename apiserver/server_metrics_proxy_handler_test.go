package apiserver_test

import (
	"code.cloudfoundry.org/lager"
	"context"
	"errors"
	"fmt"
	"github.com/alphagov/paas-billing/apiserver/auth/authfakes"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"
	"github.com/alphagov/paas-billing/instancediscoverer/instancediscovererfakes"
	"github.com/alphagov/paas-billing/metricsproxy/metricsproxyfakes"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo/v2"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerMetricsProxyHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		fakeAuthenticator *authfakes.FakeAuthenticator
		fakeStore         *eventiofakes.FakeEventStore
		fakeDiscoverer    *instancediscovererfakes.FakeCFAppDiscoverer
		fakeProxy         *metricsproxyfakes.FakeMetricsProxy
	)

	BeforeEach(func() {
		fakeDiscoverer = &instancediscovererfakes.FakeCFAppDiscoverer{}

		fakeProxy = &metricsproxyfakes.FakeMetricsProxy{}

		fakeStore = &eventiofakes.FakeEventStore{}
		fakeAuthenticator = &authfakes.FakeAuthenticator{}
		cfg = Config{
			Authenticator: fakeAuthenticator,
			Logger:        lager.NewLogger("test"),
			Store:         fakeStore,
			EnablePanic:   true,
		}
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		defer cancel()
	})
	Context("when requesting /proxymetrics/:appName/:appInstance", func() {
		It("should correctly proxy to the downstream", func() {

			app := cfclient.App{
				Name:      "app-1",
				Guid:      "app-1-guid",
				Instances: 1,
			}

			appURL := &url.URL{Host: "example.com", Scheme: "https"}

			fakeDiscoverer.GetSpaceAppByNameReturns(app, nil)
			fakeDiscoverer.GetAppRouteURLsByNameReturns(
				[]*url.URL{
					appURL,
				}, nil)

			fakeProxy.ForwardRequestToURLCalls(func(writer http.ResponseWriter, request *http.Request, u *url.URL, m map[string]string) {
				writer.WriteHeader(http.StatusOK)
				writer.Write([]byte("hello it worked yay"))
			})

			u := url.URL{}
			u.Path = fmt.Sprintf("/proxymetrics/%s/0", app.Name)
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			res := httptest.NewRecorder()

			e := NewProxyMetrics(cfg, fakeDiscoverer, fakeProxy)
			e.ServeHTTP(res, req)
			defer e.Shutdown(ctx)

			Expect(fakeProxy.ForwardRequestToURLCallCount()).To(Equal(1))

			_, argReq, url, headers := fakeProxy.ForwardRequestToURLArgsForCall(0)

			Expect(argReq).To(BeIdenticalTo(req))

			Expect(url.Host).To(Equal(appURL.Host))
			Expect(url.Path).To(Equal("/metrics"))

			Expect(headers["X-Cf-App-Instance"]).To(Equal(fmt.Sprintf("%s:0", app.Guid)))

			Expect(res.Code).To(Equal(http.StatusOK))
			Expect(res.Body.Bytes()).To(ContainSubstring("hello it worked yay"))
		})
		It("should return 404 if the app can't be retrieved", func() {
			fakeDiscoverer.GetSpaceAppByNameReturns(cfclient.App{}, errors.New(""))

			u := url.URL{Path: "/proxymetrics/aaaaa/0"}
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			res := httptest.NewRecorder()

			e := NewProxyMetrics(cfg, fakeDiscoverer, fakeProxy)
			e.ServeHTTP(res, req)
			defer e.Shutdown(ctx)

			Expect(fakeProxy.ForwardRequestToURLCallCount()).To(Equal(0))
			Expect(res.Code).To(Equal(http.StatusNotFound))
			Expect(res.Body).To(ContainSubstring("app not found"))
		})
		It("should return 400 if a non-integer instance ID is passed", func() {
			fakeDiscoverer.GetSpaceAppByNameReturns(cfclient.App{}, nil)

			u := url.URL{Path: "/proxymetrics/aaaaa/aaaaaa"}
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			res := httptest.NewRecorder()

			e := NewProxyMetrics(cfg, fakeDiscoverer, fakeProxy)
			e.ServeHTTP(res, req)
			defer e.Shutdown(ctx)

			Expect(fakeProxy.ForwardRequestToURLCallCount()).To(Equal(0))
			Expect(res.Code).To(Equal(http.StatusBadRequest))
			Expect(res.Body).To(ContainSubstring("must be an integer"))
		})
		It("should return 500 if an error occurrs while retrieving app URLs", func() {
			fakeDiscoverer.GetSpaceAppByNameReturns(cfclient.App{}, nil)
			fakeDiscoverer.GetAppRouteURLsByNameReturns(nil, errors.New(""))

			u := url.URL{Path: "/proxymetrics/aaaaa/0"}
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			res := httptest.NewRecorder()

			e := NewProxyMetrics(cfg, fakeDiscoverer, fakeProxy)
			e.ServeHTTP(res, req)
			defer e.Shutdown(ctx)

			Expect(fakeProxy.ForwardRequestToURLCallCount()).To(Equal(0))
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
			Expect(res.Body).To(ContainSubstring("could not get urls for app"))
		})
		It("should return 404 if not app urls are returned", func() {
			fakeDiscoverer.GetSpaceAppByNameReturns(cfclient.App{}, nil)
			fakeDiscoverer.GetAppRouteURLsByNameReturns([]*url.URL{}, nil)

			u := url.URL{Path: "/proxymetrics/aaaaa/0"}
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			res := httptest.NewRecorder()

			e := NewProxyMetrics(cfg, fakeDiscoverer, fakeProxy)
			e.ServeHTTP(res, req)
			defer e.Shutdown(ctx)

			Expect(fakeProxy.ForwardRequestToURLCallCount()).To(Equal(0))
			Expect(res.Code).To(Equal(http.StatusNotFound))
			Expect(res.Body).To(ContainSubstring("no urls for app"))
		})
	})

})
