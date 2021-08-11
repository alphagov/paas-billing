package eventstore_test

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetPricingPlans", func() {

	var (
		cfg eventstore.Config
	)

	It("should return all the configured plans", func() {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "epoch",
					ValidTo:   "9999-12-31:00:00",
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "GBP",
					Rate:      1,
					ValidFrom: "epoch",
					ValidTo:   "9999-12-31:00:00",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "2001-01-01T00:00:00+00:00",
					Name:      "PLAN1",
					Components: []eventio.PricingPlanComponent{
						{
							Name:         "PLAN1COMPONENT1",
							Formula:      "1111 * 1",
							CurrencyCode: "GBP",
							VATCode:      "Standard",
						},
					},
				},
				{
					PlanGUID:      eventstore.ComputePlanGUID,
					ValidFrom:     "2002-02-01T00:00:00+00:00",
					ValidTo:       "9999-12-31:00:00:00+00:00",
					Name:          "PLAN2",
					NumberOfNodes: 2,
					MemoryInMB:    64,
					StorageInMB:   1024,
					Components: []eventio.PricingPlanComponent{
						{
							Name:         "PLAN2COMPONENT1",
							Formula:      "2222 * 1",
							CurrencyCode: "GBP",
							VATCode:      "Standard",
						},
						{
							Name:         "PLAN2COMPONENT2",
							Formula:      "2222 * 2",
							CurrencyCode: "GBP",
							VATCode:      "Standard",
						},
					},
				},
			},
		}

		env, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer env.Close()
		store := env.Schema

		Expect(store.Refresh()).To(Succeed())

		plans, err := store.GetPricingPlans(eventio.TimeRangeFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2018-01-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(plans)).To(BeNumerically("==", 2), "expected two plans to be returned")

		Expect(plans[0]).To(Equal(cfg.PricingPlans[0]), "expected first returned plan to match PLAN1 data")
		Expect(plans[1]).To(Equal(cfg.PricingPlans[1]), "expected second returned plan to match PLAN2 data")
	})

})
