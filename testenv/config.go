package testenv

import "github.com/alphagov/paas-billing/eventstore"

var BasicConfig = eventstore.Config{
	VATRates: []eventstore.VATRate{
		{
			Code:      "Standard",
			Rate:      0.2,
			ValidFrom: "epoch",
		},
	},
	CurrencyRates: []eventstore.CurrencyRate{
		{
			Code:      "GBP",
			Rate:      1,
			ValidFrom: "epoch",
		},
	},
	PricingPlans: []eventstore.PricingPlan{},
}
