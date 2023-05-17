package eventstore

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.TotalCostReader = &EventStore{}

func (s *EventStore) GetTotalCost() ([]eventio.TotalCost, error) {
	startTime := time.Now()
	rows, err := s.db.Query(`select plan_guid, split_part(plan_name, ' ', 1) as service, split_part(plan_name, ' ', 2) as plan, round(sum(cost_for_duration),2) as cost from billable_event_components group by plan_guid, plan_name order by plan_guid`)

	if err != nil {
		elapsed := time.Since(startTime)
		eventStorePerformanceGauge.WithLabelValues("GetTotalCost", err.Error()).Set(elapsed.Seconds())
		s.logger.Error("get-total-cost", err, lager.Data{
			"elapsed": int64(elapsed),
		})
		return nil, err
	}

	defer rows.Close()

	planGUIDSByCost := []eventio.TotalCost{}
	for rows.Next() {
		var planGUIDByCost eventio.TotalCost
		if err := rows.Scan(&planGUIDByCost.PlanGUID, &planGUIDByCost.Kind, &planGUIDByCost.PlanName, &planGUIDByCost.Cost); err != nil {
			return nil, err
		}
		planGUIDSByCost = append(planGUIDSByCost, planGUIDByCost)

	}
	elapsed := time.Since(startTime)
	eventStorePerformanceGauge.WithLabelValues("GetTotalCost", "").Set(elapsed.Seconds())
	s.logger.Info("get-total-cost", lager.Data{
		"elapsed": int64(elapsed),
	})
	return planGUIDSByCost, nil
}
