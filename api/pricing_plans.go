package api

import (
	"errors"

	"github.com/alphagov/paas-billing/db"
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
		err := render(Single, c, db, `
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
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
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
		err := render(Single, c, db, `
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
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}

func DestroyPricingPlan(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("pricing_plan_id")
		if id == "" {
			return errors.New("missing pricing_plan_id")
		}
		err := render(Single, c, db, `
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
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}

// CreateMissingPricingPlans inserts "free" pricing plans for any plan_guids that don't have them yet
func CreateMissingPricingPlans(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := render(Single, c, db, `
			insert into pricing_plans (
				name,
				valid_from,
				plan_guid,
				formula
			) (
				select distinct
					raw_message->>'service_plan_name' as name,
					'2001-01-01'::timestamptz as valid_from,
					raw_message->>'service_plan_guid' as plan_guid,
					'0'::text as formula
				from
					service_usage_events
				where
					raw_message->>'service_plan_guid' is not null
					and not raw_message->>'service_plan_name' ~* 'CATS-|fake'
					and raw_message->>'service_plan_guid' not in (
						select plan_guid from pricing_plans
					)
			) returning name
		`)
		if err != nil {
			return err
		}
		go db.UpdateViews()
		return nil
	}
}
