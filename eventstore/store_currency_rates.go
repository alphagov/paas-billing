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
        valid_currency_exchange_rates as (
            select
                *,
                tstzrange(valid_from, valid_to) as valid_for
            from
                currency_exchange_rates
        )
        select
	    vcer.from_ccy as code,
            vcer.valid_from,
	    vcer.valid_to,
            vcer.rate
        from
            valid_currency_exchange_rates vcer
        where
            vcer.valid_for && tstzrange($1, $2)
        group by
            vcer.from_ccy,
            vcer.valid_from,
	    vcer.valid_to,
            vcer.rate
        order by
            valid_from
    `, filter.RangeStart, filter.RangeStop)
	elapsed := time.Since(startTime)
	if err != nil {
		s.logger.Error("get-currency-rates-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return nil, err
	}
	s.logger.Info("get-currency-rates-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
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
