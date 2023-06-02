package instancediscoverer_test

import (
	"code.cloudfoundry.org/lager"
	"errors"
	. "github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/alphagov/paas-billing/instancediscoverer/instancediscovererfakes"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CFAppDiscoverer", func() {
	var (
		err               error
		fakeClient        *instancediscovererfakes.FakeDiscovererClient
		appDiscoveryScope AppDiscoveryScope
		discoverer        CFAppDiscoverer
	)

	BeforeEach(func() {
		fakeClient = &instancediscovererfakes.FakeDiscovererClient{}

		appDiscoveryScope = AppDiscoveryScope{
			SpaceName:        "test-space",
			SpaceID:          "test-space-id",
			OrganizationName: "test-org",
			OrganizationID:   "test-org-id",
			AppNames:         []string{"app-1", "app-2"},
		}

		discoverer, err = New(Config{
			Client:         fakeClient,
			Logger:         lager.NewLogger("test"),
			DiscoveryScope: appDiscoveryScope,
		})
		Expect(err).ToNot(HaveOccurred())

	})
	Context("when retrieving a space app by name", func() {
		It("should correctly return an app if allowed", func() {
			appByNameResponse := cfclient.App{Name: "app-1"}
			fakeClient.AppByNameReturns(appByNameResponse, nil)
			app, err := discoverer.GetSpaceAppByName("app-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(app).To(Equal(appByNameResponse))
		})
		It("should error if not allowed", func() {
			_, err := discoverer.GetSpaceAppByName("app-3")
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeIdenticalTo(AccessDeniedError))
			Expect(fakeClient.AppByNameCallCount()).To(BeZero())

		})
	})
	Context("when retrieving app routes by name", func() {
		It("should succeed if the cloudfoundry call succeeds", func() {
			fakeClient.AppByNameReturns(cfclient.App{Name: "app-1"}, nil)
			appRoutesResponse := []cfclient.Route{cfclient.Route{Host: "some-host.example.com"}}
			fakeClient.GetAppRoutesReturns(appRoutesResponse, nil)
			routes, err := discoverer.GetAppRoutesByName("app-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(routes).To(Equal(appRoutesResponse))
		})

	})
	Context("when creating a new CFAppDiscoverer", func() {
		It("should correctly retrieve the Org and Space from the cloudfoundry api", func() {
			orgResponse := cfclient.Org{
				Name: appDiscoveryScope.OrganizationName,
				Guid: appDiscoveryScope.OrganizationID,
			}
			spaceResponse := cfclient.Space{
				Name: appDiscoveryScope.SpaceName,
				Guid: appDiscoveryScope.SpaceID,
			}
			newFakeClient := &instancediscovererfakes.FakeDiscovererClient{}
			newFakeClient.GetOrgByGuidReturns(orgResponse, nil)
			newFakeClient.GetSpaceByGuidReturns(spaceResponse, nil)

			testDiscoverer, err := New(Config{
				Client:         newFakeClient,
				Logger:         lager.NewLogger("testDiscoverer"),
				DiscoveryScope: appDiscoveryScope,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(newFakeClient.GetOrgByGuidCallCount()).To(Equal(1))
			Expect(newFakeClient.GetOrgByGuidArgsForCall(0)).To(Equal(appDiscoveryScope.OrganizationID))
			Expect(newFakeClient.GetSpaceByGuidCallCount()).To(Equal(1))
			Expect(newFakeClient.GetSpaceByGuidArgsForCall(0)).To(Equal(appDiscoveryScope.SpaceID))

			Expect(testDiscoverer.Org()).To(Equal(orgResponse))
			Expect(testDiscoverer.Space()).To(Equal(spaceResponse))
		})
		It("should throw an error if the client and clientconfig is nil", func() {
			testDiscoverer, err := New(Config{
				Logger: lager.NewLogger("testDiscoverer"),
			})
			Expect(err).To(HaveOccurred())
			Expect(testDiscoverer).To(BeNil())
		})
	})
	Context("GetAppRouteURLsByName", func() {
		It("should return a list of *url.URL - one for each app route", func() {
			app := cfclient.App{
				Name: "app-1",
				Guid: "some-guid",
			}
			routes := []cfclient.Route{
				cfclient.Route{Host: "app-1", DomainGuid: "domain-1"},
				cfclient.Route{Host: "app-1", DomainGuid: "domain-2"},
			}
			fakeClient.AppByNameReturns(app, nil)
			fakeClient.GetAppRoutesReturns(routes, nil)

			fakeClient.GetDomainByGuidReturnsOnCall(0, cfclient.Domain{Name: "example.com"}, nil)
			fakeClient.GetDomainByGuidReturnsOnCall(1, cfclient.Domain{Name: "example.net"}, nil)
			urls, err := discoverer.GetAppRouteURLsByName("app-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(urls).ToNot(BeNil())
			Expect(urls).To(HaveLen(2))

			Expect(urls[0].Host).To(Equal("app-1.example.com"))
			Expect(urls[0].Scheme).To(Equal("https"))
			Expect(urls[1].Host).To(Equal("app-1.example.net"))
			Expect(urls[1].Scheme).To(Equal("https"))

		})
		It("should skip domains which throw an error on retrieval", func() {
			app := cfclient.App{
				Name: "app-1",
				Guid: "some-guid",
			}
			routes := []cfclient.Route{
				cfclient.Route{Host: "app-1", DomainGuid: "domain-1"},
				cfclient.Route{Host: "app-1", DomainGuid: "domain-2"},
			}
			fakeClient.AppByNameReturns(app, nil)
			fakeClient.GetAppRoutesReturns(routes, nil)

			fakeClient.GetDomainByGuidReturnsOnCall(0, cfclient.Domain{Name: "example.com"}, errors.New(""))
			fakeClient.GetDomainByGuidReturnsOnCall(1, cfclient.Domain{Name: "example.net"}, nil)
			urls, err := discoverer.GetAppRouteURLsByName("app-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(urls).ToNot(BeNil())
			Expect(urls).To(HaveLen(1))

			Expect(urls[0].Host).To(Equal("app-1.example.net"))
		})
	})
})
