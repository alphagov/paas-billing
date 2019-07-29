package eventstore

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.CurrencyRateReader = &EventStore{}

func (s *EventStore) GetCurrencyRates(filter eventio.TimeRangeFilter) ([]eventio.CurrencyRate, error) {
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
        valid_currency_rates as (
            select
                *,
                tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                    partition by code order by valid_from rows between current row and 1 following
                )) as valid_for
            from
                currency_rates
        )
        select
            vcr.code,
            vcr.valid_from,
            vcr.rate
        from
            valid_currency_rates vcr
        where
            vcr.valid_for && tstzrange($1, $2)
        group by
            vcr.code,
            vcr.valid_from,
            vcr.rate
        order by
            valid_from
    `, filter.RangeStart, filter.RangeStop)
	elapsed := time.Since(startTime)
	if err != nil {
		s.logger.Error("get-currency-rates-query", err, lager.Data{
			"filter":        filter,
			"elapsed":       elapsed.String(),
			"elapse_millis": string(int64(elapsed / time.Millisecond)),
		})
		return nil, err
	}
	s.logger.Info("get-currency-rates-query", lager.Data{
		"filter":        filter,
		"elapsed":       elapsed.String(),
		"elapse_millis": string(int64(elapsed / time.Millisecond)),
	})

	defer rows.Close()
	currencyRates := []eventio.CurrencyRate{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var currencyRate eventio.CurrencyRate
		if err := json.Unmarshal(b, &currencyRate); err != nil {
			return nil, err
		}
		currencyRates = append(currencyRates, currencyRate)

	}
	return currencyRates, nil
}
