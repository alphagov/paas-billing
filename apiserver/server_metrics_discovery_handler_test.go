package apiserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alphagov/paas-billing/apiserver/auth/authfakes"
	"github.com/alphagov/paas-billing/eventio/eventiofakes"
	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/alphagov/paas-billing/instancediscoverer/instancediscovererfakes"
	"github.com/alphagov/paas-billing/metricsproxy"
	"github.com/cloudfoundry-community/go-cfclient"
	"net/http"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-billing/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsDiscoveryHandler", func() {

	var (
		ctx               context.Context
		cancel            context.CancelFunc
		cfg               Config
		err               error
		fakeAuthenticator *authfakes.FakeAuthenticator
		fakeStore         *eventiofakes.FakeEventStore
		fakeClient        *instancediscovererfakes.FakeDiscovererClient
		appDiscoveryScope instancediscoverer.AppDiscoveryScope
		fakeOrg           cfclient.Org
		fakeSpace         cfclient.Space
		discoverer        instancediscoverer.CFAppDiscoverer
		proxy             metricsproxy.MetricsProxy
	)

	BeforeEach(func() {
		appDiscoveryScope = instancediscoverer.AppDiscoveryScope{
			SpaceName:        "test-space",
			SpaceID:          "test-space-id",
			OrganizationName: "test-org",
			OrganizationID:   "test-org-id",
			AppNames:         []string{"app-1", "app-2"},
		}
		fakeClient = &instancediscovererfakes.FakeDiscovererClient{}
		fakeOrg = cfclient.Org{
			Name: appDiscoveryScope.OrganizationName,
			Guid: appDiscoveryScope.OrganizationID,
		}
		fakeSpace = cfclient.Space{
			Name: appDiscoveryScope.SpaceName,
			Guid: appDiscoveryScope.SpaceID,
		}
		fakeClient.GetOrgByGuidReturns(fakeOrg, nil)
		fakeClient.GetSpaceByGuidReturns(fakeSpace, nil)
		discoverer, err = instancediscoverer.New(instancediscoverer.Config{
			Client:         fakeClient,
			Logger:         lager.NewLogger("test-discoverer"),
			DiscoveryScope: appDiscoveryScope,
		})
		Expect(err).ToNot(HaveOccurred())

		proxy = metricsproxy.New(metricsproxy.Config{
			Logger: lager.NewLogger("test-proxy"),
		})

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
	Describe("Discovering an allowed app", func() {
		When("When the app is allowed", func() {
			Context("There is a single instance", func() {
				It("should return a valid MetricsTarget response", func() {
					appByNameResponse := cfclient.App{
						Name:      "app-1",
						Guid:      "app-1-guid",
						Instances: 1,
					}

					fakeClient.AppByNameReturns(appByNameResponse, nil)
					u := url.URL{}
					u.Path = fmt.Sprintf("/discovery/%s", appByNameResponse.Name)
					req := httptest.NewRequest(http.MethodGet, u.String(), nil)
					res := httptest.NewRecorder()

					e := NewProxyMetrics(cfg, discoverer, proxy)
					e.ServeHTTP(res, req)
					defer e.Shutdown(ctx)

					Expect(fakeClient.AppByNameCallCount()).To(Equal(1))

					Expect(res.Code).To(Equal(http.StatusOK))
					Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

					var unmarshalledResponse []MetricsTarget
					err = json.Unmarshal(res.Body.Bytes(), &unmarshalledResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(unmarshalledResponse).To(HaveLen(1))
					target := unmarshalledResponse[0]

					Expect(target.Targets).To(Equal([]string{"example.com"}))
					Expect(target.Labels).To(Equal(MetricsLabels{
						MetricsPath:     fmt.Sprintf("/proxymetrics/%s/0", appByNameResponse.Name),
						OrgName:         fakeOrg.Name,
						SpaceName:       fakeSpace.Name,
						ApplicationName: appByNameResponse.Name,
						ApplicationId:   appByNameResponse.Guid,
						InstanceNumber:  "0",
						InstanceID:      fmt.Sprintf("%s:0", appByNameResponse.Guid),
					}))
				})
			})
			Context("There are multiple instances", func() {
				It("should return multiple metricsTargets if there are multiple instances", func() {
					appByNameResponse := cfclient.App{
						Name:      "app-1",
						Guid:      "app-1-guid",
						Instances: 2,
					}

					fakeClient.AppByNameReturns(appByNameResponse, nil)
					u := url.URL{}
					u.Path = fmt.Sprintf("/discovery/%s", appByNameResponse.Name)
					req := httptest.NewRequest(http.MethodGet, u.String(), nil)
					res := httptest.NewRecorder()

					e := NewProxyMetrics(cfg, discoverer, proxy)
					e.ServeHTTP(res, req)
					defer e.Shutdown(ctx)

					Expect(fakeClient.AppByNameCallCount()).To(Equal(1))

					Expect(res.Code).To(Equal(200))
					Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

					var unmarshalledResponse []MetricsTarget
					err = json.Unmarshal(res.Body.Bytes(), &unmarshalledResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(unmarshalledResponse).To(HaveLen(2))
					Expect(unmarshalledResponse[0].Labels.InstanceNumber).To(Equal("0"))
					Expect(unmarshalledResponse[0].Labels.InstanceID).To(Equal(
						fmt.Sprintf("%s:0", appByNameResponse.Guid)))
					Expect(unmarshalledResponse[1].Labels.InstanceNumber).To(Equal("1"))
					Expect(unmarshalledResponse[1].Labels.InstanceID).To(Equal(
						fmt.Sprintf("%s:1", appByNameResponse.Guid)))
				})
			})
			Context("The app does not exist", func() {
				It("should return 404", func() {
					fakeClient.AppByNameReturns(cfclient.App{}, cfclient.CloudFoundryError{Code: 100004})
					u := url.URL{}
					u.Path = fmt.Sprintf("/discovery/%s", appDiscoveryScope.AppNames[0])
					req := httptest.NewRequest(http.MethodGet, u.String(), nil)
					res := httptest.NewRecorder()

					e := NewProxyMetrics(cfg, discoverer, proxy)
					e.ServeHTTP(res, req)
					defer e.Shutdown(ctx)

					Expect(fakeClient.AppByNameCallCount()).To(Equal(1))

					Expect(res.Code).To(Equal(404))
					Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
					Expect(res.Body).To(MatchJSON(`[]`))
				})
			})
			Context("An unknown error occurs", func() {
				It("should return 500", func() {
					fakeClient.AppByNameReturns(cfclient.App{}, fmt.Errorf("some error"))
					u := url.URL{}
					u.Path = fmt.Sprintf("/discovery/%s", appDiscoveryScope.AppNames[0])
					req := httptest.NewRequest(http.MethodGet, u.String(), nil)
					res := httptest.NewRecorder()

					e := NewProxyMetrics(cfg, discoverer, proxy)
					e.ServeHTTP(res, req)
					defer e.Shutdown(ctx)

					Expect(fakeClient.AppByNameCallCount()).To(Equal(1))

					Expect(res.Code).To(Equal(http.StatusInternalServerError))
					Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
					Expect(res.Body).To(MatchJSON(`[]`))
				})
			})
		})
		When("When the app is not allowed", func() {
			It("should return 404", func() {
				appByNameResponse := cfclient.App{
					Name:      "app-3",
					Guid:      "app-3-guid",
					Instances: 1,
				}

				fakeClient.AppByNameReturns(appByNameResponse, nil)
				u := url.URL{}
				u.Path = fmt.Sprintf("/discovery/%s", appByNameResponse.Name)
				req := httptest.NewRequest(http.MethodGet, u.String(), nil)
				res := httptest.NewRecorder()

				e := NewProxyMetrics(cfg, discoverer, proxy)
				e.ServeHTTP(res, req)
				defer e.Shutdown(ctx)

				Expect(fakeClient.AppByNameCallCount()).To(Equal(0))

				Expect(res.Code).To(Equal(404))
				Expect(res.Header().Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
				Expect(res.Body).To(MatchJSON(`[]`))
			})
		})
	})
})
