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

var _ = Describe("Orgs", func() {

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
		fakeClient.ListOrgsReturnsOnCall(0, []cfstore.Orgs{}, nil)

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

	DescribeTable("should fail to write Orgs record with invalid data",
		func(expectedErr string, orgs cfstore.Orgs) {
			fakeClient.ListOrgsReturnsOnCall(1, []cfstore.Orgs{
				orgs,
			}, nil)

			err := store.CollectOrgs()
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
		},
		Entry("bad CreatedAt", `invalid input syntax for type timestamp with time zone: "bad-created-at"`, cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-Org",
			CreatedAt:                   "bad-created-at",
			UpdatedAt:                   "2002-02-02T02:02:02+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}),
		Entry("bad UpdatedAt", `invalid input syntax for type timestamp with time zone: "bad-updated-at"`, cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-Org",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			UpdatedAt:                   "bad-updated-at",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}),
		Entry("bad Label", `violates check constraint "services_label_check"`, cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			UpdatedAt:                   "2002-02-02T02:02:02+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}),
	)

	It("should collect orgs from client", func() {
		org1 := cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-org",
			UpdatedAt:                   "2002-02-02T02:02:02+00:00",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}

		fakeClient.ListOrgsReturnsOnCall(1, []cfstore.Orgs{
			org1,
		}, nil)

		Expect(store.CollectOrgs()).To(Succeed())

		Expect(
			tempdb.Query(`select * from orgs`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                           org1.Guid,
				"name":                           "my-org",
				"updated_at":                     "2002-02-02T02:02:02+00:00",
				"created_at":                     "2001-01-01T01:01:01+00:00",
				"valid_from":                     "2001-01-01T01:01:01+00:00",
				"quota_definition_guid":          org1.QuotaDefinitionGuid,
				"default_isolation_segment_guid": org1.DefaultIsolationSegmentGuid,
			},
		}))
	})

	It("should create a new version of service when it has changed", func() {
		OrgVersion1 := cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-Org",
			UpdatedAt:                   "2001-01-01T01:01:01+00:00",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}
		fakeClient.ListOrgsReturnsOnCall(1, []cfstore.Orgs{
			OrgVersion1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())

		OrgVersion2 := OrgVersion1
		OrgVersion2.Name = "my-Org-renamed"
		OrgVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		OrgVersion2.QuotaDefinitionGuid = uuid.NewV4().String()
		fakeClient.ListOrgsReturnsOnCall(2, []cfstore.Orgs{
			OrgVersion2,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())

		Expect(
			tempdb.Query(`select * from orgs`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                           OrgVersion1.Guid,
				"name":                           "my-Org",
				"updated_at":                     "2001-01-01T01:01:01+00:00",
				"created_at":                     "2001-01-01T01:01:01+00:00",
				"valid_from":                     "2001-01-01T01:01:01+00:00",
				"quota_definition_guid":          OrgVersion1.QuotaDefinitionGuid,
				"default_isolation_segment_guid": OrgVersion1.DefaultIsolationSegmentGuid,
			},
			{
				"guid":                           OrgVersion2.Guid,
				"name":                           "my-Org-renamed",
				"updated_at":                     "2002-02-02T02:02:02+00:00",
				"created_at":                     "2001-01-01T01:01:01+00:00",
				"valid_from":                     "2002-02-02T02:02:02+00:00",
				"quota_definition_guid":          OrgVersion2.QuotaDefinitionGuid,
				"default_isolation_segment_guid": OrgVersion2.DefaultIsolationSegmentGuid,
			},
		}))
	})

	It("should only record versions of Orgs that have changed", func() {
		OrgVersion1 := cfstore.Orgs{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-Org",
			UpdatedAt:                   "2001-01-01T01:01:01+00:00",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}
		fakeClient.ListOrgsReturnsOnCall(1, []cfstore.Orgs{
			OrgVersion1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())

		fakeClient.ListOrgsReturnsOnCall(2, []cfstore.Orgs{
			OrgVersion1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())

		OrgVersion2 := OrgVersion1
		OrgVersion2.Name = "my-Org-renamed"
		OrgVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		OrgVersion2.QuotaDefinitionGuid = uuid.NewV4().String()
		fakeClient.ListOrgsReturnsOnCall(3, []cfstore.Orgs{
			OrgVersion2,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())

		Expect(
			tempdb.Query(`select * from orgs`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                           OrgVersion1.Guid,
				"name":                           "my-Org",
				"updated_at":                     "2001-01-01T01:01:01+00:00",
				"created_at":                     "2001-01-01T01:01:01+00:00",
				"valid_from":                     "2001-01-01T01:01:01+00:00",
				"quota_definition_guid":          OrgVersion1.QuotaDefinitionGuid,
				"default_isolation_segment_guid": OrgVersion1.DefaultIsolationSegmentGuid,
			},
			{
				"guid":                           OrgVersion2.Guid,
				"Name":                           "my-Org-renamed",
				"description":                    "my-org_url",
				"updated_at":                     "2002-02-02T02:02:02+00:00",
				"created_at":                     "2001-01-01T01:01:01+00:00",
				"valid_from":                     "2002-02-02T02:02:02+00:00",
				"quota_definition_guid":          OrgVersion2.QuotaDefinitionGuid,
				"default_isolation_segment_guid": OrgVersion2.DefaultIsolationSegmentGuid,
			},
		}))
	})

})
