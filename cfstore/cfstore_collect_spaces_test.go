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

var _ = Describe("Spaces", func() {

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
		fakeClient.ListSpacesReturnsOnCall(0, []cfstore.Spaces{}, nil)

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

	DescribeTable("should fail to write space record with invalid data",
		func(expectedErr string, spaces cfstore.Spaces) {
			fakeClient.ListSpacesReturnsOnCall(1, []cfstore.Spaces{
				spaces,
			}, nil)

			err := store.CollectSpaces()
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
		},
		Entry("bad CreatedAt", `invalid input syntax for type timestamp with time zone: "bad-created-at"`, cfstore.Spaces{
			Guid:             uuid.NewV4().String(),
			OrganizationGuid: uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			Name:             "my-space",
			OrgURL:           "my-org_url",
			CreatedAt:        "bad-created-at",
			UpdatedAt:        "2002-02-02T02:02:02+00:00",
		}),
		Entry("bad UpdatedAt", `invalid input syntax for type timestamp with time zone: "bad-updated-at"`, cfstore.Spaces{
			Guid:             uuid.NewV4().String(),
			OrganizationGuid: uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			Name:             "my-space",
			OrgURL:           "mmy-org_url",
			CreatedAt:        "2001-01-01T01:01:01+00:00",
			UpdatedAt:        "bad-updated-at",
		}),
		Entry("bad Name", `violates check constraint "spaces_space_name_check"`, cfstore.Spaces{
			Guid:             uuid.NewV4().String(),
			OrganizationGuid: uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			Name:             "",
			OrgURL:           "my-org_url",
			CreatedAt:        "2001-01-01T01:01:01+00:00",
			UpdatedAt:        "2002-02-02T02:02:02+00:00",
		}),
	)

	It("should collect spaces from client", func() {
		service1 := cfstore.Spaces{
			Guid:                 uuid.NewV4().String(),
			OrganizationGuid:     uuid.NewV4().String(),
			Name:                 "my-space",
			OrgURL:               "my-org_url",
			UpdatedAt:            "2002-02-02T02:02:02+00:00",
			CreatedAt:            "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
		}

		fakeClient.ListSpacesReturnsOnCall(1, []cfstore.Spaces{
			service1,
		}, nil)

		Expect(store.CollectSpaces()).To(Succeed())

		Expect(
			tempdb.Query(`select * from spaces`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                        service1.Guid,
				"organization_guid":         service1.OrganizationGuid,
				"space_name":                  "my-space",
				"org_url":            "my-org_url",
				"updated_at":                  "2002-02-02T02:02:02+00:00",
				"created_at":                  "2001-01-01T01:01:01+00:00",
				"valid_from":                  "2001-01-01T01:01:01+00:00",
				"quota_definition_guid": service1.QuotaDefinitionGuid,
				"isolation_segment_guid":      service1.IsolationSegmentGuid,
			},
		}))
	})

	It("should create a new version of space when it has changed", func() {
		spaceVersion1 := cfstore.Spaces{
			Guid:                 uuid.NewV4().String(),
			OrganizationGuid:     uuid.NewV4().String(),
			Name:                 "my-space",
			OrgURL:               "my-org_url",
			UpdatedAt:            "2001-01-01T01:01:01+00:00",
			CreatedAt:            "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
		}
		fakeClient.ListSpacesReturnsOnCall(1, []cfstore.Spaces{
			spaceVersion1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())

		spaceVersion2 := spaceVersion1
		spaceVersion2.Name = "my-space-renamed"
		spaceVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		spaceVersion2.QuotaDefinitionGuid = uuid.NewV4().String()
		fakeClient.ListSpacesReturnsOnCall(2, []cfstore.Spaces{
			spaceVersion2,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())

		Expect(
			tempdb.Query(`select * from spaces`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                        spaceVersion1.Guid,
				"organization_guid":           spaceVersion1.OrganizationGuid,
				"space_name":                  "my-space",
				"org_url":                     "my-org_url",
				"updated_at":                  "2001-01-01T01:01:01+00:00",
				"created_at":                  "2001-01-01T01:01:01+00:00",
				"valid_from":                  "2001-01-01T01:01:01+00:00",
				"quota_definition_guid": spaceVersion1.QuotaDefinitionGuid,
				"isolation_segment_guid":      spaceVersion1.IsolationSegmentGuid,
			},
			{
				"guid":                        spaceVersion2.Guid,
				"organization_guid":           spaceVersion2.OrganizationGuid,
				"space_name":                  "my-space-renamed",
				"org_url":                     "my-org_url",
				"updated_at":                  "2002-02-02T02:02:02+00:00",
				"created_at":                  "2001-01-01T01:01:01+00:00",
				"valid_from":                  "2002-02-02T02:02:02+00:00",
				"quota_definition_guid": spaceVersion2.QuotaDefinitionGuid,
				"isolation_segment_guid":      spaceVersion2.IsolationSegmentGuid,
			},
		}))
	})

	It("should only record versions of spaces that have changed", func() {
		spaceVersion1 := cfstore.Spaces{
			Guid:                 uuid.NewV4().String(),
			OrganizationGuid:     uuid.NewV4().String(),
			Name:                 "my-space",
			OrgURL:               "my-org_url",
			UpdatedAt:            "2001-01-01T01:01:01+00:00",
			CreatedAt:            "2001-01-01T01:01:01+00:00",
			QuotaDefinitionGuid:  uuid.NewV4().String(),
			IsolationSegmentGuid: uuid.NewV4().String(),
		}
		fakeClient.ListSpacesReturnsOnCall(1, []cfstore.Spaces{
			spaceVersion1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())

		fakeClient.ListSpacesReturnsOnCall(2, []cfstore.Spaces{
			spaceVersion1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())

		spaceVersion2 := spaceVersion1
		spaceVersion2.Name = "my-space-renamed"
		spaceVersion2.UpdatedAt = "2002-02-02T02:02:02+00:00"
		spaceVersion2.QuotaDefinitionGuid = uuid.NewV4().String()
		fakeClient.ListSpacesReturnsOnCall(3, []cfstore.Spaces{
			spaceVersion2,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())

		Expect(
			tempdb.Query(`select * from spaces`),
		).To(MatchJSON(testenv.Rows{
			{
				"guid":                        spaceVersion1.Guid,
				"organization_guid":           spaceVersion1.OrganizationGuid,
				"space_name":                  "my-space",
				"org_url":                     "my-org_url",
				"updated_at":                  "2001-01-01T01:01:01+00:00",
				"created_at":                  "2001-01-01T01:01:01+00:00",
				"valid_from":                  "2001-01-01T01:01:01+00:00",
				"quota_definition_guid": spaceVersion1.QuotaDefinitionGuid,
				"isolation_segment_guid":      spaceVersion1.IsolationSegmentGuid,
			},
			{
				"guid":                        spaceVersion2.Guid,
				"organization_guid":         spaceVersion2.OrganizationGuid,
				"space_name":                  "my-space-renamed",
				"org_url":                     "my-org_url",
				"updated_at":                  "2002-02-02T02:02:02+00:00",
				"created_at":                  "2001-01-01T01:01:01+00:00",
				"valid_from":                  "2002-02-02T02:02:02+00:00",
				"quota_definition_guid": spaceVersion2.QuotaDefinitionGuid,
				"isolation_segment_guid":      spaceVersion2.IsolationSegmentGuid,
			},
		}))
	})

})
