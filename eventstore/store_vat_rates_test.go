package eventstore_test

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetVATRates", func() {
	var (
		cfg eventstore.Config
	)

	It("should return all the configured VAT rates", func() {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Zero",
					ValidFrom: "1970-01-01T00:00:00+00:00",
					Rate:      0.0,
				},
				{
					Code:      "Reduced",
					ValidFrom: "1970-01-01T00:00:00+00:00",
					Rate:      0.05,
				},
				{
					Code:      "Standard",
					ValidFrom: "1970-01-01T00:00:00+00:00",
					Rate:      0.2,
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "GBP",
					Rate:      1,
					ValidFrom: "epoch",
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

		vatRates, err := store.GetVATRates(eventio.TimeRangeFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2018-01-01",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(vatRates).To(ConsistOf(cfg.VATRates), "expected returned VAT rates to match expected data")
	})

})
