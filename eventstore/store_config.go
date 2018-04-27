package eventstore

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

type Config struct {
	VATRates           []VATRate      `json:"vat_rates"`            // vat rate
	CurrencyRates      []CurrencyRate `json:"currency_rates"`       // exchange rates
	PricingPlans       []PricingPlan  `json:"pricing_plans"`        // dataset to generate prices from
	IgnoreMissingPlans bool           `json:"ignore_missing_plans"` // if true, will generate missing plans that emit "£0", useful for testing
}

func (cfg *Config) AddPlan(p PricingPlan) {
	cfg.PricingPlans = append(cfg.PricingPlans, p)
}

func (cfg *Config) AddVATRate(v VATRate) {
	cfg.VATRates = append(cfg.VATRates, v)
}

func (cfg *Config) AddCurrencyRate(c CurrencyRate) {
	cfg.CurrencyRates = append(cfg.CurrencyRates, c)
}
