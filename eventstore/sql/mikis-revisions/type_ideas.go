// /app_usage – raw usage of CF apps, without any pricing calculated
// /service_usage – raw usage of fixed-price services, without any pricing calculated
// /on_demand_service_usage – assorted chargeable events (e.g., Cost Explorer costs of S3 calls)

// /costs – without the management fee, in original currency
// /costs/YYYY/MM?org_guid=_______________________
// /costs/YYYY/MM/DD?org_guid=_______________________

// /charges – with the management fee and VAT, in £ pounds
// /charges/YYYY/MM?org_guid=_______________________
// /charges/YYYY/MM/DD?org_guid=_______________________

// /vat_rates
// /currency_rates

type Resource struct {
  ResourceType string `json:"resource_type"`
	ResourceGUID string `json:"resource_guid"`
  ResourceName string `json:"resource_name"`
  PlanGUID     string `json:"plan_guid"`
  PlanName     string `json:"plan_name"`
  OrgGUID      string `json:"org_guid"`
  OrgName      string `json:"org_name"`
  SpaceGUID    string `json:"space_guid"`
  SpaceName    string `json:"space_name"`
}

type Cost struct {
  Resource
	CostGUID     string            `json:"guid"`
	Day          string            `json:"day"`
  Start        string            `json:"start_time"`
  Stop         string            `json:"stop_time"`
  Metadata     map[string]string `json:"metadata"`
  CurrencyCode string            `json:"currency_code"`
  BaseCost     string            `json:"base_cost"`
}

type Charge struct {
	Cost
  ManagementFee string `json:"management_fee"`
  VAT           string `json:"vat"`
  Total         string `json:"total"`
}


type BillableEvent struct {
  EventGUID           string `json:"event_guid"`
  EventStart          string `json:"event_start"`
  EventStop           string `json:"event_stop"`
  PlanGUID            string `json:"plan_guid"`
  PlanName            string `json:"plan_name"`
  QuotaDefinitionGUID string `json:"quota_definition_guid"`
  NumberOfNodes       int64  `json:"number_of_nodes"`
  MemoryInMB          int64  `json:"memory_in_mb"`
  StorageInMB         int64  `json:"storage_in_mb"`
  Price               Price  `json:"price"`
}
type Price struct {
  IncVAT  string           `json:"inc_vat"`
  ExVAT   string           `json:"ex_vat"`
  Details []PriceComponent `json:"details"`
}
type PriceComponent struct {
  Name         string `json:"name"`
  PlanName     string `json:"plan_name"`
  Start        string `json:"start"`
  Stop         string `json:"stop"`
  VatRate      string `json:"vat_rate"`
  VatCode      string `json:"vat_code"`
  CurrencyCode string `json:"currency_code"`
  IncVAT       string `json:"inc_vat"`
  ExVAT        string `json:"ex_vat"`
}
