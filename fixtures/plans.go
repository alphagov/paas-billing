package fixtures

import (
	"time"

	"github.com/alphagov/paas-billing/db"
)

type Plans map[string]Plan

type Plan struct {
	ID        int
	PlanGuid  string
	ValidFrom time.Time
	Formula   string
}

func (plans Plans) Insert(sqlClient *db.PostgresClient) error {
	for planName, plan := range plans {
		_, err := sqlClient.Conn.Exec(`
            INSERT INTO pricing_plans(id, name, valid_from, plan_guid, formula) VALUES (
                $1,
                $2,
                $3,
                $4,
                $5
            );
        `, plan.ID, planName, plan.ValidFrom, plan.PlanGuid, plan.Formula)
		if err != nil {
			return err
		}
	}
	return nil
}
