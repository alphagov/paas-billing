package testenv

import "github.com/alphagov/paas-billing/schema"

var BasicConfig = schema.Config{
	VATRates: []schema.VATRate{
		{
			Code:      "Standard",
			Rate:      0.2,
			ValidFrom: "epoch",
		},
	},
	CurrencyRates: []schema.CurrencyRate{
		{
			Code:      "GBP",
			Rate:      1,
			ValidFrom: "epoch",
		},
	},
	PricingPlans: []schema.PricingPlan{},
}
