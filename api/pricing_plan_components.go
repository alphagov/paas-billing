package api

import (
	"errors"
	"fmt"

	"github.com/alphagov/paas-billing/db"
	"github.com/labstack/echo"
)

type PricingPlanComponent struct {
	PricingPlanID int    `json:"pricing_plan_id" form:"pricing_plan_id"`
	Name          string `json:"name" form:"name"`
	Formula       string `json:"formula" form:"formula"`
	VATRateID     int    `json:"vat_rate_id" form:"vat_rate_id"`
}

func NewPricingPlanComponentFromContext(c echo.Context) (*PricingPlanComponent, error) {
	p := &PricingPlanComponent{}
	if err := c.Bind(p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %s", err.Error())
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %s", err.Error())
	}
	return p, nil
}

func (p *PricingPlanComponent) Validate() error {
	if p.PricingPlanID == 0 {
		return errors.New("pricing_plan_id is required")
	}
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.Formula == "" {
		return errors.New("formula is required")
	}
	if p.VATRateID == 0 {
		return errors.New("vat_rate_id is required")
	}
	return nil
}

func ListPricingPlanComponents(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render(Many, c, db, `
			select
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
			from
				pricing_plan_components
			order by
				pricing_plan_id, id
		`)
	}
}

func ListPricingPlanComponentsByPlan(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		planID, err := parseIntParam(c, "pricing_plan_id")
		if err != nil {
			return err
		}
		return render(Many, c, db, `
			select
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
			from
				pricing_plan_components
			where
				pricing_plan_id = $1
			order by
				id
		`, planID)
	}
}

func GetPricingPlanComponent(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		return render(Single, c, db, `
			select
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
			from
				pricing_plan_components
			where
				id = $1
		`, id)
	}
}

func CreatePricingPlanComponent(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		ppc, err := NewPricingPlanComponentFromContext(c)
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			insert into pricing_plan_components (
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
			) values (
				$1,
				$2,
				$3,
				$4
			) returning
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
		`, ppc.PricingPlanID, ppc.Name, ppc.Formula, ppc.VATRateID)
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}

func UpdatePricingPlanComponent(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		ppc, err := NewPricingPlanComponentFromContext(c)
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			update pricing_plan_components set
				pricing_plan_id = $2::numeric,
				name = $3,
				formula = $4,
				vat_rate_id = $5
			where
				id = $1
			returning
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
		`, id, ppc.PricingPlanID, ppc.Name, ppc.Formula, ppc.VATRateID)
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}

func DestroyPricingPlanComponent(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := parseIntParam(c, "id")
		if err != nil {
			return err
		}

		err = render(Single, c, db, `
			delete from
				pricing_plan_components
			where
				id = $1
			returning
				id,
				pricing_plan_id,
				name,
				formula,
				vat_rate_id
		`, id)
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}
