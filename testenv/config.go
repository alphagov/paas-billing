package testenv

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
)

var BasicConfig = eventstore.Config{
	VATRates: []eventio.VATRate{
		{
			Code:      "Standard",
			Rate:      0.2,
			ValidFrom: "epoch",
		},
	},
	CurrencyRates: []eventio.CurrencyRate{
		{
			Code:      "GBP",
			Rate:      1,
			ValidFrom: "epoch",
		},
	},
	PricingPlans: []eventio.PricingPlan{},
}
