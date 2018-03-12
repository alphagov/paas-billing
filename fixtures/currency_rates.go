package fixtures

import (
	"time"

	"github.com/alphagov/paas-billing/db"
)

type CurrencyRate struct {
	ID        int
	Code      string
	ValidFrom time.Time
	Rate      float64
}

type CurrencyRates []CurrencyRate

func (currencyRates CurrencyRates) Insert(sqlClient *db.PostgresClient) error {
	for _, currencyRate := range currencyRates {
		_, err := sqlClient.Conn.Exec(`
		INSERT INTO currency_rates(code, valid_from, rate) VALUES ($1, $2, $3);
	`, currencyRate.Code, currencyRate.ValidFrom, currencyRate.Rate)

		if err != nil {
			return err
		}
	}
	return nil
}
