package cfstore_test

import (
	"github.com/alphagov/paas-billing/cfstore"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("Services", func() {

	var (
		tempdb     *testenv.TempDB
		fakeClient *fakes.FakeCFDataClient
		store      *cfstore.Store
	)

	BeforeEach(func() {
		var err error
		tempdb, err = testenv.Open(testenv.BasicConfig)
		Expect(err).ToNot(HaveOccurred())

		fakeClient = &fakes.FakeCFDataClient{}
		fakeClient.ListServicesReturnsOnCall(0, []cfstore.Service{}, nil)

		store, err = cfstore.New(cfstore.Config{
			Client: fakeClient,
			DB:     tempdb.Conn,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(store.Init()).To(Succeed())
	})

	AfterEach(func() {
		tempdb.Close()
	})

	DescribeTable("should fail to write service record with invalid data",
		func(expectedErr string, servicePlan cfstore.Service) {
			fakeClient.ListServicesReturnsOnCall(1, []cfstore.Service{
				servicePlan,
			}, nil)

			err := store.CollectServices()
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
		},
		Entry("bad CreatedAt", `invalid input syntax for type timestamp with time zone: "bad-created-at"`, cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "my-service",
			Description:       "my-service-description",
			CreatedAt:         "bad-created-at",
			UpdatedAt:         "2002-02-02T02:02:02+00:00",
		}),
		Entry("bad UpdatedAt", `invalid input syntax for type timestamp with time zone: "bad-updated-at"`, cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "my-service",
			Description:       "my-service-description",
			CreatedAt:         "2001-01-01T01:01:01+00:00",
			UpdatedAt:         "bad-updated-at",
		}),
		Entry("bad Label", `violates check constraint "services_label_check"`, cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "",
			Description:       "my-service-description",
			CreatedAt:         "2001-01-01T01:01:01+00:00",
			UpdatedAt:         "2002-02-02T02:02:02+00:00",
		}),
	)

	It("should collect services from client", func() {
		service1 := cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "my-service",
			Description:       "my-service-description",
			UpdatedAt:         "2002-02-02T02:02:02+00:00",
			CreatedAt:         "2001-01-01T01:01:01+00:00",
		}

		fakeClient.ListServicesReturnsOnCall(1, []cfstore.Service{
			service1,
		}, nil)

		Expect(store.CollectServices()).To(Succeed())

		Expect(
			tempdb.Query(`select * from services`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                service1.Guid,
				"service_broker_guid": service1.ServiceBrokerGuid,
				"label":               "my-service",
				"description":         "my-service-description",
				"updated_at":          "2002-02-02T02:02:02+00:00",
				"created_at":          "2001-01-01T01:01:01+00:00",
				"valid_from":          "2001-01-01T01:01:01+00:00",
				"active":              false,
				"bindable":            false,
			},
		}))
	})

	It("should create a new version of service when it has changed", func() {
		serviceVersion1 := cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "my-service",
			Description:       "my-service-description",
			UpdatedAt:         "2001-01-01T01:01:01+00:00",
			CreatedAt:         "2001-01-01T01:01:01+00:00",
		}
		fakeClient.ListServicesReturnsOnCall(1, []cfstore.Service{
			serviceVersion1,
		}, nil)
		Expect(store.CollectServices()).To(Succeed())

		serviceVersion2 := serviceVersion1
		serviceVersion2.Label = "my-service-renamed"
		serviceVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		fakeClient.ListServicesReturnsOnCall(2, []cfstore.Service{
			serviceVersion2,
		}, nil)
		Expect(store.CollectServices()).To(Succeed())

		Expect(
			tempdb.Query(`select * from services`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                serviceVersion1.Guid,
				"service_broker_guid": serviceVersion1.ServiceBrokerGuid,
				"label":               "my-service",
				"description":         "my-service-description",
				"updated_at":          "2001-01-01T01:01:01+00:00",
				"created_at":          "2001-01-01T01:01:01+00:00",
				"valid_from":          "2001-01-01T01:01:01+00:00",
				"active":              false,
				"bindable":            false,
			},
			{
				"guid":                serviceVersion2.Guid,
				"service_broker_guid": serviceVersion2.ServiceBrokerGuid,
				"label":               "my-service-renamed",
				"description":         "my-service-description",
				"updated_at":          "2002-02-02T02:02:02+00:00",
				"created_at":          "2001-01-01T01:01:01+00:00",
				"valid_from":          "2002-02-02T02:02:02+00:00",
				"active":              false,
				"bindable":            false,
			},
		}))
	})

	It("should only record versions of services that have changed", func() {
		serviceVersion1 := cfstore.Service{
			Guid:              uuid.NewV4().String(),
			ServiceBrokerGuid: uuid.NewV4().String(),
			Label:             "my-service",
			Description:       "my-service-description",
			UpdatedAt:         "2001-01-01T01:01:01+00:00",
			CreatedAt:         "2001-01-01T01:01:01+00:00",
		}
		fakeClient.ListServicesReturnsOnCall(1, []cfstore.Service{
			serviceVersion1,
		}, nil)
		Expect(store.CollectServices()).To(Succeed())

		fakeClient.ListServicesReturnsOnCall(2, []cfstore.Service{
			serviceVersion1,
		}, nil)
		Expect(store.CollectServices()).To(Succeed())

		serviceVersion2 := serviceVersion1
		serviceVersion2.Label = "my-service-renamed"
		serviceVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		fakeClient.ListServicesReturnsOnCall(3, []cfstore.Service{
			serviceVersion2,
		}, nil)
		Expect(store.CollectServices()).To(Succeed())

		Expect(
			tempdb.Query(`select * from services`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                serviceVersion1.Guid,
				"service_broker_guid": serviceVersion1.ServiceBrokerGuid,
				"label":               "my-service",
				"description":         "my-service-description",
				"updated_at":          "2001-01-01T01:01:01+00:00",
				"created_at":          "2001-01-01T01:01:01+00:00",
				"valid_from":          "2001-01-01T01:01:01+00:00",
				"active":              false,
				"bindable":            false,
			},
			{
				"guid":                serviceVersion2.Guid,
				"service_broker_guid": serviceVersion2.ServiceBrokerGuid,
				"label":               "my-service-renamed",
				"description":         "my-service-description",
				"updated_at":          "2002-02-02T02:02:02+00:00",
				"created_at":          "2001-01-01T01:01:01+00:00",
				"valid_from":          "2002-02-02T02:02:02+00:00",
				"active":              false,
				"bindable":            false,
			},
		}))
	})

})
