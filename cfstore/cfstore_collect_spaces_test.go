package cfstore_test

import (
	"github.com/alphagov/paas-billing/cfstore"
	"github.com/alphagov/paas-billing/fakes"
	"github.com/alphagov/paas-billing/testenv"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo/v2"

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
		fakeClient.ListSpacesReturnsOnCall(0, []cfclient.Space{}, nil)

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

	It("should collect space data", func() {
		space1 := cfclient.Space{
			Guid:      uuid.NewV4().String(),
			Name:      "my-space",
			CreatedAt: "2001-01-01T01:01:01+00:00",
			UpdatedAt: "2002-02-02T02:02:02+00:00",
		}

		By("storing the data using the created_at date for the valid_from field initially")
		fakeClient.ListSpacesReturnsOnCall(1, []cfclient.Space{
			space1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())
		expectedFirstRow := testenv.Row{
			"guid":       space1.Guid,
			"name":       space1.Name,
			"valid_from": space1.CreatedAt,
			"updated_at": space1.UpdatedAt,
			"created_at": space1.CreatedAt,
		}
		expectedResult1 := testenv.Rows{expectedFirstRow}
		Expect(tempdb.Query(`select * from spaces`)).To(MatchJSON(expectedResult1))

		By("storing the data using the updated_at date for the valid_from field for all subsequent operations")
		expectedSecondRow := testenv.Row{
			"guid":       space1.Guid,
			"name":       space1.Name,
			"valid_from": space1.UpdatedAt, // THIS IS THE DIFFERENCE FROM 1stROW ^^
			"updated_at": space1.UpdatedAt,
			"created_at": space1.CreatedAt,
		}
		expectedResult2 := testenv.Rows{expectedFirstRow, expectedSecondRow}

		fakeClient.ListSpacesReturnsOnCall(2, []cfclient.Space{
			space1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())
		Expect(tempdb.Query(`select * from spaces`)).To(MatchJSON(expectedResult2))

		By("not changing any data when the stored valid_from date matches the updated_at field during all subsequent operations")
		fakeClient.ListSpacesReturnsOnCall(3, []cfclient.Space{
			space1,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())
		Expect(tempdb.Query(`select * from spaces`)).To(MatchJSON(expectedResult2))

		By("storing updates to the space")
		space2 := cfclient.Space{
			Guid:      space1.Guid,
			Name:      "my-space",
			CreatedAt: "2001-01-01T01:01:01+00:00",
			UpdatedAt: "2003-03-03T03:03:03+00:00",
		}
		fakeClient.ListSpacesReturnsOnCall(4, []cfclient.Space{
			space2,
		}, nil)
		Expect(store.CollectSpaces()).To(Succeed())
		expectedThirdRow := testenv.Row{
			"guid":       space2.Guid,
			"name":       space2.Name,
			"valid_from": space2.UpdatedAt,
			"updated_at": space2.UpdatedAt,
			"created_at": space2.CreatedAt,
		}
		expectedResult3 := testenv.Rows{
			expectedFirstRow,
			expectedSecondRow,
			expectedThirdRow,
		}
		Expect(tempdb.Query(`select * from spaces`)).To(MatchJSON(expectedResult3))
	})

})
