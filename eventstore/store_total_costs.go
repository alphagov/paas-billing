package eventstore

import (
	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.TotalCostReader = &EventStore{}

func (s *EventStore) GetTotalCost() ([]eventio.TotalCost, error) {
	rows, err := s.db.Query(`select plan_guid, round(sum(cost_for_duration),2) as cost from billable_event_components group by plan_guid order by plan_guid`)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	planGUIDSByCost := []eventio.TotalCost{}
	for rows.Next() {
		var planGUIDByCost eventio.TotalCost
		if err := rows.Scan(&planGUIDByCost.PlanGUID, &planGUIDByCost.Cost); err != nil {
			return nil, err
		}
		planGUIDSByCost = append(planGUIDSByCost, planGUIDByCost)

	}
	return planGUIDSByCost, nil
}
