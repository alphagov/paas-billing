package api

import (
	"errors"
	"fmt"

	"github.com/alphagov/paas-billing/db"
	"github.com/labstack/echo"
)

type VATRate struct {
	Name string  `json:"name" form:"name"`
	Rate float64 `json:"rate" form:"rate"`
}

func NewVATRateFromContext(c echo.Context) (*VATRate, error) {
	v := &VATRate{}
	if err := c.Bind(v); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %s", err.Error())
	}
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %s", err.Error())
	}
	return v, nil
}

func (v *VATRate) Validate() error {
	if v.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func ListVATRates(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render(Many, c, db, `
			select
				id,
				name,
				rate
			from
				vat_rates
			order by
				id
		`)
	}
}

func GetVATRate(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		return render(Single, c, db, `
			select
				id,
				name,
				rate
			from
				vat_rates
			where
				id = $1
		`, id)
	}
}

func CreateVATRate(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		vr, err := NewVATRateFromContext(c)
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			insert into vat_rates (
				name,
				rate
			) values (
				$1,
				$2
			) returning
				id,
				name,
				rate
		`, vr.Name, vr.Rate)
		if err != nil {
			return err
		}
		return nil
	}
}

func UpdateVATRate(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		vr, err := NewVATRateFromContext(c)
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			update vat_rates set
				name = $2,
				rate = $3::numeric
			where
				id = $1
			returning
				id,
				name,
				rate
		`, id, vr.Name, vr.Rate)
		if err != nil {
			return err
		}
		return nil
	}
}

func DestroyVATRate(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			delete from
				vat_rates
			where
				id = $1
			returning
				id,
				name,
				rate
		`, id)
		if err != nil {
			return err
		}
		return nil
	}
}
