package eventstore

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.TotalCostReader = &EventStore{}

func (s *EventStore) GetTotalCost() ([]eventio.TotalCost, error) {
	startTime := time.Now()
	rows, err := s.db.Query(`select plan_guid, round(sum(cost_for_duration),2) as cost from billable_event_components group by plan_guid order by plan_guid`)

	if err != nil {
		s.logger.Error("get-total-cost", err, lager.Data{
			"elapsed": int64(time.Since(startTime)),
		})
		return nil, err
	}

	defer rows.Close()

	planGUIDSByCost := []eventio.TotalCost{}
	for rows.Next() {
		var planGUIDByCost eventio.TotalCost
		if err := rows.Scan(&planGUIDByCost.PlanGUID, &planGUIDByCost.Cost); err != nil {
			return nil, err
		}
		planGUIDSByCost = append(planGUIDSByCost, planGUIDByCost)

	}

	s.logger.Info("get-total-cost", lager.Data{
		"elapsed": int64(time.Since(startTime)),
	})
	return planGUIDSByCost, nil
}
