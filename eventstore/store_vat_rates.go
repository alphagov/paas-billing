package eventstore

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.VATRateReader = &EventStore{}

func (s *EventStore) GetVATRates(filter eventio.TimeRangeFilter) ([]eventio.VATRate, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	startTime := time.Now()
	rows, err := queryJSON(tx, `
        with
        valid_vat_rates as (
            select
                *,
                tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                    partition by code order by valid_from rows between current row and 1 following
                )) as valid_for
            from
                vat_rates
        )
        select
            vvr.code,
            vvr.valid_from,
            vvr.rate
        from
            valid_vat_rates vvr
        where
            vvr.valid_for && tstzrange($1, $2)
        group by
            vvr.code,
            vvr.valid_from,
            vvr.rate
        order by
            valid_from
    `, filter.RangeStart, filter.RangeStop)
	elapsed := time.Since(startTime)
	if err != nil {
		eventStorePerformanceGauge.WithLabelValues("GetVATRates", err.Error()).Set(elapsed.Seconds())
		s.logger.Error("get-vat-rates-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return nil, err
	}
	eventStorePerformanceGauge.WithLabelValues("GetVATRates", "").Set(elapsed.Seconds())
	s.logger.Info("get-vat-rates-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
	})

	defer rows.Close()
	vatRates := []eventio.VATRate{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var vatRate eventio.VATRate
		if err := json.Unmarshal(b, &vatRate); err != nil {
			return nil, err
		}
		vatRates = append(vatRates, vatRate)
	}
	return vatRates, nil
}
