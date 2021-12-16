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
			ValidTo:   "9999-12-31T23:59:59Z",
		},
	},
	CurrencyRates: []eventio.CurrencyRate{
		{
			Code:      "GBP",
			Rate:      1,
			ValidFrom: "epoch",
			ValidTo:   "9999-12-31T23:59:59Z",
		},
	},
	PricingPlans: []eventio.PricingPlan{},
}
