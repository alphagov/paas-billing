package cfstore_test

import (
	"github.com/alphagov/paas-billing/cfstore"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/alphagov/paas-billing/testenv"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"

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
		fakeClient.ListOrgsReturnsOnCall(0, []cfclient.Org{}, nil)

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

	It("should collect org data", func() {
		org1 := cfclient.Org{
			Guid:                        uuid.NewV4().String(),
			Name:                        "my-org",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			UpdatedAt:                   "2002-02-02T02:02:02+00:00",
			QuotaDefinitionGuid:         uuid.NewV4().String(),
			DefaultIsolationSegmentGuid: uuid.NewV4().String(),
		}

		By("storing the data using the created_at date for the valid_from field initially")
		fakeClient.ListOrgsReturnsOnCall(1, []cfclient.Org{
			org1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())
		expectedFirstRow := testenv.Row{
			"guid":                           org1.Guid,
			"name":                           org1.Name,
			"valid_from":                     org1.CreatedAt,
			"updated_at":                     org1.UpdatedAt,
			"created_at":                     org1.CreatedAt,
			"quota_definition_guid":          org1.QuotaDefinitionGuid,
			"default_isolation_segment_guid": org1.DefaultIsolationSegmentGuid,
		}
		expectedResult1 := testenv.Rows{expectedFirstRow}
		Expect(tempdb.Query(`select * from orgs`)).To(MatchJSON(expectedResult1))

		By("storing the data using the updated_at date for the valid_from field for all subsequent operations")
		expectedSecondRow := testenv.Row{
			"guid":                           org1.Guid,
			"name":                           org1.Name,
			"valid_from":                     org1.UpdatedAt, // THIS IS THE DIFFERENCE FROM 1stROW ^^
			"updated_at":                     org1.UpdatedAt,
			"created_at":                     org1.CreatedAt,
			"quota_definition_guid":          org1.QuotaDefinitionGuid,
			"default_isolation_segment_guid": org1.DefaultIsolationSegmentGuid,
		}
		expectedResult2 := testenv.Rows{expectedFirstRow, expectedSecondRow}

		fakeClient.ListOrgsReturnsOnCall(2, []cfclient.Org{
			org1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())
		Expect(tempdb.Query(`select * from orgs`)).To(MatchJSON(expectedResult2))

		By("not changing any data when the stored valid_from date matches the updated_at field during all subsequent operations")
		fakeClient.ListOrgsReturnsOnCall(3, []cfclient.Org{
			org1,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())
		Expect(tempdb.Query(`select * from orgs`)).To(MatchJSON(expectedResult2))

		By("storing updates to the org")
		org2 := cfclient.Org{
			Guid:                        org1.Guid,
			Name:                        "my-org",
			CreatedAt:                   "2001-01-01T01:01:01+00:00",
			UpdatedAt:                   "2003-03-03T03:03:03+00:00",
			QuotaDefinitionGuid:         org1.QuotaDefinitionGuid,
			DefaultIsolationSegmentGuid: org1.DefaultIsolationSegmentGuid,
		}
		fakeClient.ListOrgsReturnsOnCall(4, []cfclient.Org{
			org2,
		}, nil)
		Expect(store.CollectOrgs()).To(Succeed())
		expectedThirdRow := testenv.Row{
			"guid":                           org2.Guid,
			"name":                           org2.Name,
			"valid_from":                     org2.UpdatedAt,
			"updated_at":                     org2.UpdatedAt,
			"created_at":                     org2.CreatedAt,
			"quota_definition_guid":          org2.QuotaDefinitionGuid,
			"default_isolation_segment_guid": org2.DefaultIsolationSegmentGuid,
		}
		expectedResult3 := testenv.Rows{
			expectedFirstRow,
			expectedSecondRow,
			expectedThirdRow,
		}
		Expect(tempdb.Query(`select * from orgs`)).To(MatchJSON(expectedResult3))
	})

})
