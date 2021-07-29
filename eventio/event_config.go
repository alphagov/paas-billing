package eventio

type PricingPlan struct {
	Name          string                 `json:"name"`
	PlanGUID      string                 `json:"plan_guid"`
	ValidFrom     string                 `json:"valid_from"`
	ValidTo       string                 `json:"valid_to"`
	Components    []PricingPlanComponent `json:"components"`
	MemoryInMB    uint                   `json:"memory_in_mb"`
	StorageInMB   uint                   `json:"storage_in_mb"`
	NumberOfNodes uint                   `json:"number_of_nodes"`
}

type PricingPlanComponent struct {
	Name          string `json:"name"`
	Formula       string `json:"formula"`
  ExternalPrice string `json:"external_price"`
	VATCode       string `json:"vat_code"`
	CurrencyCode  string `json:"currency_code"`
}

type VATRate struct {
	Code      string  `json:"code"`
	ValidFrom string  `json:"valid_from"`
	ValidTo   string  `json:"valid_to"`
	Rate      float64 `json:"rate"`
}

type CurrencyRate struct {
	Code      string  `json:"code"`
	ValidFrom string  `json:"valid_from"`
	ValidTo   string  `json:"valid_to"`
	Rate      float64 `json:"rate"`
}
