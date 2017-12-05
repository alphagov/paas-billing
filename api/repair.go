package api

import (
	"fmt"
	"net/http"

	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
)

func RepairEvents(dbClient db.SQLClient, cfClient cloudfoundry.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		if cfClient == nil {
			return fmt.Errorf("cfClient required for RepairEvents")
		}
		pgClient, ok := dbClient.(*db.PostgresClient)
		if !ok {
			return fmt.Errorf("%T does not support RepairEvents", dbClient)
		}
		err := pgClient.RepairEvents(cfClient)
		if err != nil {
			return err
		}
		go pgClient.UpdateViews()
		return c.Redirect(http.StatusFound, "/repair_events")
	}
}

func GetRepairedEvents(dbClient db.SQLClient, cfClient cloudfoundry.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		if cfClient == nil {
			return fmt.Errorf("cfClient required for RepairEvents")
		}
		pgClient, ok := dbClient.(*db.PostgresClient)
		if !ok {
			return fmt.Errorf("%T does not support RepairEvents", dbClient)
		}
		return render(Many, c, pgClient, `
			with events as (
				(
					select
						id,
						created_at::timestamptz as created_at,
						(raw_message->>'app_guid') as guid,
						(raw_message->>'app_name') as name,
						(raw_message->>'org_guid') as org_guid,
						(raw_message->>'space_guid') as space_guid,
						'f4d4b95a-f55e-4593-8d54-3364c25798c4'::text as plan_guid, -- fake plan id for compute plans
						'default-compute'::text as plan_name,                      -- fake plan name for compute plans
						coalesce(raw_message->>'instance_count', '0')::numeric as inst_count,
						coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
						raw_message->>'state' as state
					from
						app_usage_events
					where
						raw_message->>'state' = 'STARTED'
						or raw_message->>'state' = 'STOPPED'
				) union all (
					select
						id,
						created_at::timestamptz as created_at,
						(raw_message->>'service_instance_guid') as guid,
						(raw_message->>'service_instance_name') as name,
						(raw_message->>'org_guid') as org_guid,
						(raw_message->>'space_guid') as space_guid,
						(raw_message->>'service_plan_guid') as plan_guid,
						(raw_message->>'service_plan_name') as plan_name,
						'1'::numeric as inst_count,
						'0'::numeric as memory_in_mb,
						case
							when (raw_message->>'state') = 'CREATED' then 'STARTED'
							when (raw_message->>'state') = 'DELETED' then 'STOPPED'
							when (raw_message->>'state') = 'UPDATED' then 'STARTED'
						end as state
					from
						service_usage_events
					where
						raw_message->>'service_instance_type' = 'managed_service_instance'
				)
			)
			select * from events where id = 0
		`)
	}
}
