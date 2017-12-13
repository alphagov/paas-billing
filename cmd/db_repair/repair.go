package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	"github.com/alphagov/paas-usage-events-collector/db"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

const fakeID = 0

func createEventsForAppsWithNoRecordedEvents(tx *sql.Tx, firstEventTimestamp *time.Time, spaces map[string]cfclient.Space, cfClient cf.Client) (err error) {
	// fetch list of app and service guids we have events for
	rows, err := tx.Query(`
		select distinct
			raw_message->>'app_guid'
		from
			app_usage_events
		where
			raw_message->>'app_guid' is not null
	`)
	if err != nil {
		return err
	}
	knownGuids := map[string]bool{}
	for rows.Next() {
		var guid string
		if err := rows.Scan(&guid); err != nil {
			return err
		}
		knownGuids[guid] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	apps, err := cfClient.GetApps()
	if err != nil {
		return err
	}
	fmt.Println("apps:", len(apps))
	for _, app := range apps {
		// Skip non running apps
		if app.State != db.StateStarted {
			continue
		}
		fmt.Println("app", app.Guid)

		// Skip if we already have events for this app
		if seen := knownGuids[app.Guid]; seen {
			continue
		}
		fmt.Println("detected missing app", app.Guid)

		// Skip anything updated since we started collecting events
		updated, err := time.Parse(time.RFC3339, app.UpdatedAt)
		if err != nil {
			return err
		}
		if updated.After(*firstEventTimestamp) {
			fmt.Println("not seen events for app", app.Guid, "but skipping since it was updated", updated, "which is after we started collecting data")
			continue
		}
		space, ok := spaces[app.SpaceGuid]
		if !ok {
			return errors.New("failed to find space: " + app.SpaceGuid)
		}
		// We have found a running app that we have no events for
		ev := map[string]interface{}{
			"state":                              "STARTED",
			"app_guid":                           app.Guid,
			"app_name":                           app.Name,
			"org_guid":                           space.OrganizationGuid,
			"task_guid":                          nil,
			"task_name":                          nil,
			"space_guid":                         space.Guid,
			"space_name":                         space.Name,
			"process_type":                       "web",
			"package_state":                      "STAGED",
			"buildpack_guid":                     app.DetectedBuildpackGuid,
			"buildpack_name":                     app.DetectedBuildpack,
			"instance_count":                     app.Instances,
			"previous_state":                     "STOPPED",
			"parent_app_guid":                    app.Guid,
			"parent_app_name":                    app.Name,
			"previous_package_state":             "UNKNOWN",
			"previous_instance_count":            0,
			"memory_in_mb_per_instance":          app.Memory,
			"previous_memory_in_mb_per_instance": app.Memory,
		}
		fmt.Println("adding missing app STARTED event", ev)
		evJSON, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			insert into app_usage_events (
				id,
				guid,
				created_at,
				raw_message
			) values (
				$1,
				$2::text,
				$3::timestamp,
				$4::jsonb
			)
		`, fakeID, app.Guid, firstEventTimestamp, string(evJSON))
		if err != nil {
			return err
		}
		knownGuids[app.Guid] = true
	}
	return nil

}

func createEventsForServicesWithNoRecordedEvents(tx *sql.Tx, firstEventTimestamp *time.Time, spaces map[string]cfclient.Space, cfClient cf.Client) (err error) {
	rows, err := tx.Query(`
		select distinct
			raw_message->>'service_instance_guid'
		from
			service_usage_events
		where
			raw_message->>'service_instance_guid' is not null
	`)
	if err != nil {
		return err
	}
	knownGuids := map[string]bool{}
	for rows.Next() {
		var guid string
		if err := rows.Scan(&guid); err != nil {
			return err
		}
		knownGuids[guid] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	orgs, err := cfClient.GetOrgs()
	if err != nil {
		return err
	}
	fmt.Println("orgs:", len(orgs))
	services, err := cfClient.GetServices()
	if err != nil {
		return err
	}
	fmt.Println("services:", len(services))
	servicePlans, err := cfClient.GetServicePlans()
	if err != nil {
		return err
	}
	fmt.Println("servicePlans:", len(servicePlans))

	srvs, err := cfClient.GetServiceInstances()
	if err != nil {
		return err
	}
	for _, serviceInstance := range srvs {
		// Skip anything updated since we started collecting events
		updated, err := time.Parse(time.RFC3339, serviceInstance.UpdatedAt)
		if err != nil {
			return err
		}
		if updated.After(*firstEventTimestamp) {
			continue
		}
		// Skip if we already have events for this service instance
		if seen := knownGuids[serviceInstance.Guid]; seen {
			continue
		}
		space, ok := spaces[serviceInstance.SpaceGuid]
		if !ok {
			return errors.New("failed to find space: " + serviceInstance.SpaceGuid)
		}
		org, ok := orgs[space.OrganizationGuid]
		if !ok {
			return errors.New("failed to find org: " + space.OrganizationGuid)
		}
		servicePlan, ok := servicePlans[serviceInstance.ServicePlanGuid]
		if !ok {
			return errors.New("failed to find service plan for:" + serviceInstance.ServicePlanGuid)
		}
		service, ok := services[serviceInstance.ServiceGuid]
		if !ok {
			return errors.New("failed to find service for plan:" + serviceInstance.ServiceGuid)
		}

		// We have found a running service instance that we have no events for
		ev := map[string]interface{}{
			"state":                 "CREATED",
			"org_guid":              org.Guid,
			"space_guid":            space.Guid,
			"space_name":            space.Name,
			"service_guid":          service.Guid,
			"service_label":         service.Label,
			"service_plan_guid":     servicePlan.Guid,
			"service_plan_name":     servicePlan.Name,
			"service_instance_guid": serviceInstance.Guid,
			"service_instance_name": serviceInstance.Name,
			"service_instance_type": "managed_service_instance",
		}
		evJSON, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		fmt.Println("adding missing service CREATED event", ev)
		_, err = tx.Exec(`
			insert into service_usage_events (
				id,
				guid,
				created_at,
				raw_message
			) values (
				$1,
				$2::text,
				$3::timestamp,
				$4::jsonb
			)
		`, fakeID, serviceInstance.Guid, firstEventTimestamp, string(evJSON))
		if err != nil {
			return err
		}
		knownGuids[serviceInstance.Guid] = true
	}
	return nil
}

func resetFakeEvents(tx *sql.Tx, cfClient cf.Client) (err error) {
	// remove any events with fake ID
	_, err = tx.Exec(`delete from app_usage_events where id = $1`, fakeID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from service_usage_events where id = $1`, fakeID)
	if err != nil {
		return err
	}
	return nil
}

func getCollectionEpoch(tx *sql.Tx) (*time.Time, error) {
	var firstEventTimestamp *time.Time
	err := tx.QueryRow(`
		select least((
			select min(created_at::timestamptz) from app_usage_events where created_at is not null
		),(
			select min(created_at::timestamptz) from service_usage_events where created_at is not null
		));
	`).Scan(&firstEventTimestamp)
	if err != nil {
		return nil, err
	}
	if firstEventTimestamp == nil {
		return nil, errors.New("Database appears to be empty and thus cannot be repaired.")
	}
	return firstEventTimestamp, nil
}

func createEventsForAppsWhereFirstRecordedEventIsStopped(tx *sql.Tx, firstEventTimestamp *time.Time) (err error) {
	result, err := tx.Exec(`
		WITH events AS (
			SELECT
				first_value(raw_message) OVER first_event AS first_event
			FROM app_usage_events
			WHERE
				raw_message->>'state' = 'STARTED'
			OR
				raw_message->>'state' = 'STOPPED'
			WINDOW
				first_event AS (partition by raw_message->>'app_guid' order by id rows between unbounded preceding and current row)
		),

		stop_events AS (
			SELECT
				first_event
			FROM
				events
			WHERE
				first_event->>'state' = 'STOPPED'
			GROUP BY
				first_event
		),

		missing_events AS (
			SELECT
				$1::int AS id,
				uuid_generate_v4() AS guid,
				$2::timestamptz AS created_at,
				(first_event || '{"state": "STARTED"}'::jsonb) as raw_message
			FROM
				stop_events
		)

		INSERT INTO app_usage_events (
			id,
			guid,
			created_at,
			raw_message
		)(
			SELECT
				id,
				guid,
				created_at,
				raw_message
			FROM
				missing_events
		)
		`, fakeID, firstEventTimestamp)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("Inserted %v missing STARTED events for apps\n", rowsAffected)
	return nil
}

func createEventsForServicesWhereFirstRecordedEventIsDeleted(tx *sql.Tx, firstEventTimestamp *time.Time) (err error) {
	result, err := tx.Exec(`
		WITH events AS (
			SELECT
				first_value(raw_message) OVER first_event AS first_event
			FROM service_usage_events
			WINDOW
				first_event AS (partition by raw_message->>'service_instance_guid' order by id rows between unbounded preceding and current row)
		),

		stop_events AS (
			SELECT
				first_event
			FROM
				events
			WHERE
				first_event->>'state' = 'DELETED'
			GROUP BY
				first_event
		),

		missing_events AS (
			SELECT
				$1::int AS id,
				uuid_generate_v4() AS guid,
				$2::timestamptz AS created_at,
				(first_event || '{"state": "CREATED"}'::jsonb) as raw_message
			FROM
				stop_events
		)

		INSERT INTO service_usage_events (
			id,
			guid,
			created_at,
			raw_message
		)(
			SELECT
				id,
				guid,
				created_at,
				raw_message
			FROM
				missing_events
		)
	`, fakeID, firstEventTimestamp)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("Inserted %v missing CREATED events for services\n", rowsAffected)
	return nil
}
