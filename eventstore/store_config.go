package eventstore

import (
	"encoding/json"

	"github.com/alphagov/paas-billing/eventio"
)

type Config struct {
	VATRates           []eventio.VATRate      `json:"vat_rates"`            // vat rate
	CurrencyRates      []eventio.CurrencyRate `json:"currency_rates"`       // exchange rates
	PricingPlans       []eventio.PricingPlan  `json:"pricing_plans"`        // dataset to generate prices from
	IgnoreMissingPlans bool                   `json:"ignore_missing_plans"` // if true, will generate missing plans that emit "£0", useful for testing
}

func (cfg *Config) AddPlan(p eventio.PricingPlan) {
	cfg.PricingPlans = append(cfg.PricingPlans, p)
}

func (cfg *Config) AddVATRate(v eventio.VATRate) {
	cfg.VATRates = append(cfg.VATRates, v)
}

func (cfg *Config) AddCurrencyRate(c eventio.CurrencyRate) {
	cfg.CurrencyRates = append(cfg.CurrencyRates, c)
}

var _ eventio.PricingPlanReader = &EventStore{}

func (s *EventStore) GetPricingPlans(filter eventio.TimeRangeFilter) ([]eventio.PricingPlan, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := queryJSON(tx, `
		with
		valid_pricing_plans as (
			select
				*,
				tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
					partition by plan_guid order by valid_from rows between current row and 1 following
				)) as valid_for
			from
				pricing_plans
		)
		select
			vpp.plan_guid,
			vpp.valid_from,
			vpp.name,
			vpp.memory_in_mb,
			vpp.number_of_nodes,
			vpp.storage_in_mb,
			json_agg(json_build_object(
				'plan_guid', ppc.plan_guid::text,
				'name', ppc.name,
				'formula', ppc.formula,
				'vat_code', ppc.vat_code,
				'currency_code', ppc.currency_code
			)) as components
		from
			valid_pricing_plans vpp
		left join
			pricing_plan_components ppc on ppc.plan_guid = vpp.plan_guid
			and ppc.valid_from = vpp.valid_from
		where
			vpp.valid_for && tstzrange($1, $2)
		group by
			vpp.plan_guid,
			vpp.valid_from,
			vpp.name,
			vpp.memory_in_mb,
			vpp.number_of_nodes,
			vpp.storage_in_mb
		order by
			valid_from
	`, filter.RangeStart, filter.RangeStop)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := []eventio.PricingPlan{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var plan eventio.PricingPlan
		if err := json.Unmarshal(b, &plan); err != nil {
			return nil, err
		}
		plans = append(plans, plan)

	}
	return plans, nil
}
