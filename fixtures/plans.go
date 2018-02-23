package fixtures

import (
	"time"

	"github.com/alphagov/paas-billing/db"
)

type PricingPlanComponent struct {
	ID      int
	Name    string
	Formula string
}

type Plans []Plan

type Plan struct {
	ID         int
	Name       string
	PlanGuid   string
	ValidFrom  time.Time
	Components []PricingPlanComponent
}

func (plans Plans) Insert(sqlClient *db.PostgresClient) error {
	for _, plan := range plans {
		_, err := sqlClient.Conn.Exec(`
            INSERT INTO pricing_plans(id, name, valid_from, plan_guid) VALUES (
                $1,
                $2,
                $3,
                $4
            );
        `, plan.ID, plan.Name, plan.ValidFrom, plan.PlanGuid)
		if err != nil {
			return err
		}

		for _, component := range plan.Components {
			_, err := sqlClient.Conn.Exec(
				`INSERT INTO pricing_plan_components(id, pricing_plan_id, name, formula) VALUES (
	          $1,
	          $2,
	          $3,
	          $4
	      );`,
				component.ID, plan.ID, component.Name, component.Formula,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
