package eventstore_test

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetCurrencyRates", func() {
	var (
		cfg eventstore.Config
	)

	It("should return all the configured currency rates", func() {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "1970-01-01T00:00:00+00:00",

				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "GBP",
					Rate:      1,
					ValidFrom: "1970-01-01T00:00:00+00:00",
				},
				{
					Code:      "USD",
					Rate:      0.8,
					ValidFrom: "1970-01-01T00:00:00+00:00",
				},
				{
					Code:      "USD",
					Rate:      0.74,
					ValidFrom: "2003-01-14T00:00:00+00:00",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "2001-01-01T00:00:00+00:00",
					ValidTo:   "2002-02-01T00:00:00+00:00",
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

		currencyRates, err := store.GetCurrencyRates(eventio.TimeRangeFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2018-01-01",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(currencyRates).To(ConsistOf(cfg.CurrencyRates), "expected returned currency rates to match expected data")
	})

})
