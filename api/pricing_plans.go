package api

import (
	"errors"

	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
)

func ListPricingPlans(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render(Many, c, db, `
			select
				id,
				iso8601(valid_from) as valid_from,
				name,
				plan_guid,
				formula
			from
				pricing_plans	
			order by
				valid_from, plan_guid
		`)
	}
}

func GetPricingPlan(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("pricing_plan_id")
		if id == "" {
			return errors.New("missing pricing_plan_id")
		}
		return render(Single, c, db, `
			select
				id,
				iso8601(valid_from) as valid_from,
				name,
				plan_guid,
				formula
			from
				pricing_plans
			where
				id = $1::integer
			order by
				valid_from, plan_guid
			limit 1
		`, id)
	}
}

func CreatePricingPlan(db db.SQLClient) echo.HandlerFunc {
	type CreatePricingPlan struct {
		Name      string `json:"name" form:"name"`
		ValidFrom string `json:"valid_from" form:"valid_from"`
		PlanGuid  string `json:"plan_guid" form:"plan_guid"`
		Formula   string `json:"formula" form:"formula"`
	}
	return func(c echo.Context) error {
		pp := CreatePricingPlan{}
		if err := c.Bind(&pp); err != nil {
			return err
		}
		if pp.Name == "" {
			return errors.New("name is required")
		}
		if pp.ValidFrom == "" {
			return errors.New("valid_from is required")
		}
		if pp.PlanGuid == "" {
			return errors.New("plan_guid is required")
		}
		if pp.Formula == "" {
			return errors.New("formula is required")
		}
		return render(Single, c, db, `
			insert into pricing_plans (
				name,
				valid_from,
				plan_guid,
				formula
			) values (
				$1,
				$2,
				$3,
				$4
			) returning
				id,
				name,
				iso8601(valid_from) as valid_from,
				plan_guid,
				formula
		`, pp.Name, pp.ValidFrom, pp.PlanGuid, pp.Formula)
	}
}

func UpdatePricingPlan(db db.SQLClient) echo.HandlerFunc {
	type UpdatePricingPlan struct {
		Name      string `json:"name" form:"name"`
		ValidFrom string `json:"valid_from" form:"valid_from"`
		PlanGuid  string `json:"plan_guid" form:"plan_guid"`
		Formula   string `json:"formula" form:"formula"`
	}
	return func(c echo.Context) error {
		id := c.Param("pricing_plan_id")
		if id == "" {
			return errors.New("missing pricing_plan_id")
		}
		pp := UpdatePricingPlan{}
		if err := c.Bind(&pp); err != nil {
			return err
		}
		if pp.Name == "" {
			return errors.New("name is required")
		}
		if pp.ValidFrom == "" {
			return errors.New("valid_from is required")
		}
		if pp.PlanGuid == "" {
			return errors.New("plan_guid is required")
		}
		if pp.Formula == "" {
			return errors.New("formula is required")
		}
		return render(Single, c, db, `
			update pricing_plans set
				name = $1,
				valid_from = $2,
				plan_guid = $3,
				formula = $4
			where
				id = $5
			returning
				id,
				name,
				iso8601(valid_from) as valid_from,
				plan_guid,
				formula
		`, pp.Name, pp.ValidFrom, pp.PlanGuid, pp.Formula, id)
	}
}

func DestroyPricingPlan(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("pricing_plan_id")
		if id == "" {
			return errors.New("missing pricing_plan_id")
		}
		return render(Single, c, db, `
			delete from
				pricing_plans
			where
				id = $1::integer
			returning
				id,
				name,
				iso8601(valid_from) as valid_from,
				plan_guid,
				formula
		`, id)
	}
}
