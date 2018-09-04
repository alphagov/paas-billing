package eventstore_test

import (
	"fmt"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetConsolidatedBillableEvents", func() {
	var (
		cfg      eventstore.Config
		scenario *testenv.TestScenario
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
		scenario = testenv.NewTestScenario("2001-01-01T00:00")
	})

	It("should match the output of GetBillableEvents for a complex scenario across multiple months", func() {
		scenario.AddComputePlan()

		cfg.AddVATRate(eventio.VATRate{
			Code:      "Standard",
			Rate:      0,
			ValidFrom: "2017-03-01",
		})
		cfg.AddCurrencyRate(eventio.CurrencyRate{
			Code:      "GBP",
			Rate:      2,
			ValidFrom: "2017-02-01",
		})
		cfg.AddCurrencyRate(eventio.CurrencyRate{
			Code:      "GBP",
			Rate:      4,
			ValidFrom: "2017-04-01",
		})

		scenario.AppLifeCycle("org1", "space1", "app1",
			testenv.EventInfo{Delta: "+0h", State: "STARTED"},
			testenv.EventInfo{Delta: "+3600h", State: "STOPPED"},
		)
		scenario.AppLifeCycle("org2", "space2", "app2",
			testenv.EventInfo{Delta: "+12h", State: "STARTED"},
			testenv.EventInfo{Delta: "+3600h", State: "STOPPED"},
		)
		scenario.AppLifeCycle("org2", "space2", "app3",
			testenv.EventInfo{Delta: "+24h", State: "STARTED"},
			testenv.EventInfo{Delta: "+36h", State: "STOPPED"},
		)

		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Schema.Refresh()).To(Succeed())

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})

		for month := 1; month <= 4; month++ {
			filter := eventio.EventFilter{
				RangeStart: fmt.Sprintf("2017-%02d-01", month),
				RangeStop:  fmt.Sprintf("2017-%02d-01", month+1),
			}
			err = db.Schema.Consolidate(filter)
			Expect(err).NotTo(HaveOccurred())

			allBillableEvents, err := db.Schema.GetBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			allConsolidatedBillableEvents, err := db.Schema.GetConsolidatedBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			Expect(allBillableEvents).To(Equal(allConsolidatedBillableEvents))

			filter.OrgGUIDs = []string{scenario.GetOrgGUID("org1")}
			allOrg1BillableEvents, err := db.Schema.GetBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			allOrg1ConsolidatedBillableEvents, err := db.Schema.GetConsolidatedBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			Expect(allOrg1BillableEvents).To(Equal(allOrg1ConsolidatedBillableEvents))

			filter.OrgGUIDs = []string{scenario.GetOrgGUID("org2")}
			allOrg2BillableEvents, err := db.Schema.GetBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			allOrg2ConsolidatedBillableEvents, err := db.Schema.GetConsolidatedBillableEvents(filter)
			Expect(err).ToNot(HaveOccurred())
			Expect(allOrg2BillableEvents).To(Equal(allOrg2ConsolidatedBillableEvents))
		}
	})

	It("Should fail to GetBillableEvents if query range is not one month", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = db.Schema.GetConsolidatedBillableEvents(eventio.EventFilter{
			RangeStart: "2001-02-01",
			RangeStop:  "2001-02-02",
		})
		Expect(err).To(MatchError("consolidation only works with ranges starting and ending on month boundaries"))

		_, err = db.Schema.GetConsolidatedBillableEvents(eventio.EventFilter{
			RangeStart: "2001-01-15",
			RangeStop:  "2001-02-15",
		})
		Expect(err).To(MatchError("consolidation only works with ranges starting and ending on month boundaries"))
	})
})

var _ = Describe("Consolidate", func() {
	var (
		cfg      eventstore.Config
		scenario *testenv.TestScenario
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
		scenario = testenv.NewTestScenario("2001-01-01T00:00")
	})

	It("Should fail to Consolidate if organisation filter provided", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-02",
			RangeStop:  "2001-02-02",
			OrgGUIDs:   []string{"banana", "pear"},
		})
		Expect(err).To(MatchError(
			"consolidate must be called without an organisations filter (i.e. for all orgs)",
		))
	})

	It("Should fail to Consolidate if query range is not exactly one month", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-02",
			RangeStop:  "2001-02-02",
		})
		Expect(err).To(MatchError(
			MatchRegexp("violates check constraint"),
		))

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-07-01",
		})
		Expect(err).To(MatchError(
			MatchRegexp("violates check constraint"),
		))
	})

	It("Should fail if consolidate called twice for the same range", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())
		err = db.Schema.Consolidate(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).To(MatchError(MatchRegexp("duplicate key value violates unique constraint")))
	})
})

var _ = Describe("IsRangeConsolidated", func() {
	var (
		cfg      eventstore.Config
		scenario *testenv.TestScenario
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
		scenario = testenv.NewTestScenario("2001-01-01T00:00")
	})

	It("Should return false if range has not been consolidated", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		result, err := db.Schema.IsRangeConsolidated(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Should return true if range has been consolidated", func() {
		db, err := scenario.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		filter := eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		}

		db.Schema.Consolidate(filter)
		result, err := db.Schema.IsRangeConsolidated(filter)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeTrue())
	})

})
