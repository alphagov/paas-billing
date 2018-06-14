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

var _ = Describe("ServicePlans", func() {

	var (
		tempdb      *testenv.TempDB
		fakeClient  *fakes.FakeCFDataClient
		store       *cfstore.Store
		testService = cfstore.Service{
			Guid:              uuid.NewV4().String(),
			Label:             "test-service",
			Description:       "test-service-desc",
			CreatedAt:         "2001-01-01T00:00:00+00:00",
			UpdatedAt:         "2002-02-02T00:00:00+00:00",
			ServiceBrokerGuid: uuid.NewV4().String(),
		}
	)

	BeforeEach(func() {
		var err error
		tempdb, err = testenv.Open(testenv.BasicConfig)
		Expect(err).ToNot(HaveOccurred())

		fakeClient = &fakes.FakeCFDataClient{}
		fakeClient.ListServicesReturnsOnCall(0, []cfstore.Service{testService}, nil)
		fakeClient.ListServicePlansReturnsOnCall(0, []cfstore.ServicePlan{}, nil)

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

	It("should be safe to call Init() multiple times", func() {
		fakeClient.ListServicesReturns([]cfstore.Service{testService}, nil)
		fakeClient.ListServicePlansReturns([]cfstore.ServicePlan{}, nil)
		Expect(store.Init()).To(Succeed())
		Expect(store.Init()).To(Succeed())
	})

	DescribeTable("should fail to write record with invalid data",
		func(expectedErr string, servicePlan cfstore.ServicePlan) {
			fakeClient.ListServicePlansReturnsOnCall(1, []cfstore.ServicePlan{
				servicePlan,
			}, nil)

			err := store.CollectServicePlans()
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
		},
		Entry("bad CreatedAt", `invalid input syntax for type timestamp with time zone: "bad-created-at"`, cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "my-service-plan",
			CreatedAt:   "bad-created-at",
			UpdatedAt:   "2002-02-02T02:02:02+00:00",
			ServiceGuid: testService.Guid,
		}),
		Entry("bad UpdatedAt", `invalid input syntax for type timestamp with time zone: "bad-updated-at"`, cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "my-service-plan",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			UpdatedAt:   "bad-updated-at",
			ServiceGuid: testService.Guid,
		}),
		Entry("bad Name", `violates check constraint "service_plans_name_check"`, cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			UpdatedAt:   "2002-02-02T02:02:02+00:00",
			ServiceGuid: testService.Guid,
		}),
		Entry("bad UniqueId", `invalid input syntax for uuid: "bad-unique-id"`, cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    "bad-unique-id",
			Name:        "my-service-plan",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			UpdatedAt:   "2002-02-02T02:02:02+00:00",
			ServiceGuid: testService.Guid,
		}),
	)

	It("should collect service plans from client", func() {
		servicePlan1 := cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "my-service-plan",
			UpdatedAt:   "2002-02-02T02:02:02+00:00",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			Extra:       "Blah blah extra stuff ",
			ServiceGuid: testService.Guid,
		}

		fakeClient.ListServicePlansReturnsOnCall(1, []cfstore.ServicePlan{
			servicePlan1,
		}, nil)

		Expect(store.CollectServicePlans()).To(Succeed())

		Expect(
			tempdb.Query(`select * from service_plans`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":               servicePlan1.Guid,
				"unique_id":          servicePlan1.UniqueId,
				"name":               "my-service-plan",
				"updated_at":         "2002-02-02T02:02:02+00:00",
				"created_at":         "2001-01-01T01:01:01+00:00",
				"extra":              "Blah blah extra stuff ",
				"free":               false,
				"valid_from":         "2001-01-01T01:01:01+00:00",
				"description":        "",
				"active":             false,
				"public":             false,
				"service_guid":       testService.Guid,
				"service_valid_from": testService.CreatedAt,
			},
		}))
	})

	It("should create a new version of service_plan when it has changed", func() {
		servicePlanVersion1 := cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "my-service-plan",
			UpdatedAt:   "2001-01-01T01:01:01+00:00",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			Extra:       "Blah blah extra stuff ",
			ServiceGuid: testService.Guid,
		}
		fakeClient.ListServicePlansReturnsOnCall(1, []cfstore.ServicePlan{
			servicePlanVersion1,
		}, nil)
		Expect(store.CollectServicePlans()).To(Succeed())

		servicePlanVersion2 := servicePlanVersion1
		servicePlanVersion2.Name = "my-service-plan-renamed"
		servicePlanVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		fakeClient.ListServicePlansReturnsOnCall(2, []cfstore.ServicePlan{
			servicePlanVersion2,
		}, nil)
		Expect(store.CollectServicePlans()).To(Succeed())

		Expect(
			tempdb.Query(`select * from service_plans`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":               servicePlanVersion1.Guid,
				"unique_id":          servicePlanVersion1.UniqueId,
				"name":               "my-service-plan",
				"updated_at":         "2001-01-01T01:01:01+00:00",
				"created_at":         "2001-01-01T01:01:01+00:00",
				"extra":              "Blah blah extra stuff ",
				"free":               false,
				"valid_from":         "2001-01-01T01:01:01+00:00",
				"description":        "",
				"active":             false,
				"public":             false,
				"service_guid":       testService.Guid,
				"service_valid_from": testService.CreatedAt,
			},
			{
				"guid":               servicePlanVersion2.Guid,
				"unique_id":          servicePlanVersion2.UniqueId,
				"name":               "my-service-plan-renamed",
				"updated_at":         "2002-02-02T02:02:02+00:00",
				"created_at":         "2001-01-01T01:01:01+00:00",
				"extra":              "Blah blah extra stuff ",
				"free":               false,
				"valid_from":         "2002-02-02T02:02:02+00:00",
				"description":        "",
				"active":             false,
				"public":             false,
				"service_guid":       testService.Guid,
				"service_valid_from": testService.CreatedAt,
			},
		}))
	})

	It("should only record versions of service_plans that have changed", func() {
		servicePlanVersion1 := cfstore.ServicePlan{
			Guid:        uuid.NewV4().String(),
			UniqueId:    uuid.NewV4().String(),
			Name:        "my-service-plan",
			UpdatedAt:   "2001-01-01T01:01:01+00:00",
			CreatedAt:   "2001-01-01T01:01:01+00:00",
			Extra:       "Blah blah extra stuff ",
			ServiceGuid: testService.Guid,
		}
		fakeClient.ListServicePlansReturnsOnCall(1, []cfstore.ServicePlan{
			servicePlanVersion1,
		}, nil)
		Expect(store.CollectServicePlans()).To(Succeed())

		fakeClient.ListServicePlansReturnsOnCall(2, []cfstore.ServicePlan{
			servicePlanVersion1,
		}, nil)
		Expect(store.CollectServicePlans()).To(Succeed())

		servicePlanVersion2 := servicePlanVersion1
		servicePlanVersion2.Name = "my-service-plan-renamed"
		servicePlanVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		fakeClient.ListServicePlansReturnsOnCall(3, []cfstore.ServicePlan{
			servicePlanVersion2,
		}, nil)
		Expect(store.CollectServicePlans()).To(Succeed())

		Expect(
			tempdb.Query(`select * from service_plans`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":               servicePlanVersion1.Guid,
				"unique_id":          servicePlanVersion1.UniqueId,
				"name":               "my-service-plan",
				"updated_at":         "2001-01-01T01:01:01+00:00",
				"created_at":         "2001-01-01T01:01:01+00:00",
				"extra":              "Blah blah extra stuff ",
				"free":               false,
				"valid_from":         "2001-01-01T01:01:01+00:00",
				"description":        "",
				"active":             false,
				"public":             false,
				"service_guid":       testService.Guid,
				"service_valid_from": testService.CreatedAt,
			},
			{
				"guid":               servicePlanVersion2.Guid,
				"unique_id":          servicePlanVersion2.UniqueId,
				"name":               "my-service-plan-renamed",
				"updated_at":         "2002-02-02T02:02:02+00:00",
				"created_at":         "2001-01-01T01:01:01+00:00",
				"extra":              "Blah blah extra stuff ",
				"free":               false,
				"valid_from":         "2002-02-02T02:02:02+00:00",
				"description":        "",
				"active":             false,
				"public":             false,
				"service_guid":       testService.Guid,
				"service_valid_from": testService.CreatedAt,
			},
		}))
	})

})
