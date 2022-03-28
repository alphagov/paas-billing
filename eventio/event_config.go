package eventio

type PricingPlan struct {
	Name          string                 `json:"name"`
	PlanGUID      string                 `json:"plan_guid"`
	ValidFrom     string                 `json:"valid_from"`
	Components    []PricingPlanComponent `json:"components"`
	MemoryInMB    uint                   `json:"memory_in_mb"`
	StorageInMB   uint                   `json:"storage_in_mb"`
	NumberOfNodes uint                   `json:"number_of_nodes"`
}

type PricingPlanComponent struct {
	Name         string `json:"name"`
	Formula      string `json:"formula"`
	VATCode      string `json:"vat_code"`
	CurrencyCode string `json:"currency_code"`
}

type VATRate struct {
	Code      string  `json:"code"`
	ValidFrom string  `json:"valid_from"`
	Rate      float64 `json:"rate"`
}

type CurrencyRate struct {
	Code      string  `json:"code"`
	ValidFrom string  `json:"valid_from"`
	Rate      float64 `json:"rate"`
}

type Org struct {
	GUID                string    `json:"guid"`
	Name                string    `json:"name"`
	ValidFrom           time.Time `json:"valid_from"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	QuotaDefinitionGUID string    `json:"quota_definition_guid"`
}
