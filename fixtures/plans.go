package fixtures

import (
	"time"

	"github.com/alphagov/paas-billing/db"
)

type PricingPlanComponent struct {
	ID        int
	Name      string
	Formula   string
	VATRateID int
	Currency  string
}

type Plans []Plan

type Plan struct {
	ID          int
	Name        string
	PlanGuid    string
	ValidFrom   time.Time
	Components  []PricingPlanComponent
	MemoryInMb  uint
	StorageInMb uint
}

func (plans Plans) Insert(sqlClient *db.PostgresClient) error {
	for _, plan := range plans {
		_, err := sqlClient.Conn.Exec(`
            INSERT INTO pricing_plans(id, name, valid_from, plan_guid, memory_in_mb, storage_in_mb) VALUES (
                $1,
                $2,
                $3,
                $4,
                $5,
                $6
            );
        `, plan.ID, plan.Name, plan.ValidFrom, plan.PlanGuid, plan.MemoryInMb, plan.StorageInMb)
		if err != nil {
			return err
		}

		for _, component := range plan.Components {
			_, err := sqlClient.Conn.Exec(
				`INSERT INTO pricing_plan_components(id, pricing_plan_id, name, formula, vat_rate_id, currency) VALUES (
	          $1,
	          $2,
	          $3,
	          $4,
	          $5,
	          $6
	      );`,
				component.ID, plan.ID, component.Name, component.Formula, component.VATRateID, component.Currency,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
