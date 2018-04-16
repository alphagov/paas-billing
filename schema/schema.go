package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/store"
	"github.com/lib/pq"
)

const (
	AppUsageTableName     = "app_usage_events"
	ServiceUsageTableName = "service_usage_events"
	ComputePlanGUID       = "f4d4b95a-f55e-4593-8d54-3364c25798c4"
	DefaultInitTimeout    = 5 * time.Minute
	DefaultRefreshTimeout = 5 * time.Minute
	DefaultStoreTimeout   = 30 * time.Second
	DefaultQueryTimeout   = 30 * time.Second
)

var _ store.EventStorer = &Schema{}

type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Schema struct {
	db  DB
	cfg Config
	ctx context.Context
}

func New(ctx context.Context, db *sql.DB, cfg Config) *Schema {
	return &Schema{
		db:  db,
		cfg: cfg,
		ctx: ctx,
	}
}

func NewFromConfig(ctx context.Context, db *sql.DB, filename string) (*Schema, error) {
	cfg, err := LoadConfig(filename)
	if err != nil {
		return nil, err
	}
	return New(ctx, db, cfg), nil
}

// Init initialises the database tables and functions
func (s *Schema) Init() error {
	ctx, _ := context.WithTimeout(s.ctx, DefaultInitTimeout)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// drop all the projection data
	if err := execFile(tx, "drop_ephemeral_objects.sql"); err != nil {
		return err
	}
	defer tx.Rollback()
	// execute the collector's migrations
	if err := execFile(tx, "create_app_usage_events.sql"); err != nil {
		return err
	}
	if err := execFile(tx, "create_service_usage_events.sql"); err != nil {
		return err
	}
	if err := execFile(tx, "create_compose_audit_events.sql"); err != nil {
		return err
	}
	// reset / create the ephemeral report data
	if err := s.refresh(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Refresh triggers regeneration of the cached normalized view of the event dat and rebuilds the
// billable components. Ideally you should do this once a day
func (s *Schema) Refresh() error {
	ctx, _ := context.WithTimeout(s.ctx, DefaultRefreshTimeout)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.refresh(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// refresh rebuilds the schema in the given transaction
func (s *Schema) refresh(tx *sql.Tx) error {
	// drop all the projection data
	if err := execFile(tx, "drop_ephemeral_objects.sql"); err != nil {
		return err
	}
	// create the ephemeral configuration objects (pricing/plans/etc)
	if err := execFile(tx, "create_ephemeral_objects.sql"); err != nil {
		return err
	}
	// reset the event normalization
	if err := execFile(tx, "create_events.sql"); err != nil {
		return err
	}
	// populate the config
	if err := s.initVATRates(tx); err != nil {
		return err
	}
	if err := s.initCurrencyRates(tx); err != nil {
		return err
	}
	if err := s.initPlans(tx); err != nil {
		return err
	}
	// create the billable components view of the data
	if err := execFile(tx, "create_billable_event_components.sql"); err != nil {
		return err
	}
	return nil
}

func (s *Schema) initVATRates(tx *sql.Tx) error {
	for _, vr := range s.cfg.VATRates {
		_, err := tx.Exec(`
			insert into vat_rates (
				code, valid_from, rate
			) values (
				$1, $2, $3
			)
		`, vr.Code, vr.ValidFrom, vr.Rate)
		if err != nil {
			return wrapPqError(err, "invalid vat rate")
		}
	}
	return nil
}

func (s *Schema) initCurrencyRates(tx *sql.Tx) error {
	for _, cr := range s.cfg.CurrencyRates {
		_, err := tx.Exec(`
			insert into currency_rates (
				code, valid_from, rate
			) values (
				$1, $2, $3
			)
		`, cr.Code, cr.ValidFrom, cr.Rate)
		if err != nil {
			return wrapPqError(err, "invalid currency rate")
		}
	}
	return nil
}

// InitPlans destroys all existing plans and replaces them with those specified
// by pricingPlans if the new set of plans does not satisfy the existing data
// (for example if you are missing plans for services found in the events then
// it will fail to update plans and rollback the transaction
func (s *Schema) initPlans(tx *sql.Tx) (err error) {
	for _, pp := range s.cfg.PricingPlans {
		_, err := tx.Exec(`insert into pricing_plans (
			plan_guid, valid_from, name,
			memory_in_mb, storage_in_mb, number_of_nodes
		) values (
			$1, $2, $3,
			$4, $5, $6
		)`, pp.PlanGUID, pp.ValidFrom, pp.Name,
			pp.MemoryInMB, pp.StorageInMB, pp.NumberOfNodes,
		)
		if err != nil {
			return wrapPqError(err, "invalid pricing plan")
		}
		for _, ppc := range pp.Components {
			_, err := tx.Exec(`insert into pricing_plan_components (
				plan_guid, valid_from, name,
				formula, currency_code, vat_code
			) values (
				$1, $2, $3,
				$4, $5, $6
			)`, pp.PlanGUID, pp.ValidFrom, ppc.Name, ppc.Formula, ppc.CurrencyCode, ppc.VATCode)
			if err != nil {
				return wrapPqError(err, "invalid pricing plan component")
			}
		}
	}

	if s.cfg.IgnoreMissingPlans {
		if err := generateMissingPlans(tx); err != nil {
			return err
		}
	}

	if err := checkPricingComponents(tx); err != nil {
		return err
	}

	if err := checkPlanConsistancy(tx); err != nil {
		return err
	}

	if err := checkVATRates(tx); err != nil {
		return err
	}

	if err := checkCurrencyRates(tx); err != nil {
		return err
	}

	return nil
}

func (s *Schema) StoreEvents(events []store.RawEvent) error {
	ctx, _ := context.WithTimeout(s.ctx, DefaultStoreTimeout)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		switch event.Kind {
		case "app", "service":
			if err := s.storeUsageEvent(tx, event); err != nil {
				return err
			}
		case "compose":
			if err := s.storeComposeEvent(tx, event); err != nil {
				return err
			}
		default:
			return fmt.Errorf("cannot store event without a Kind: %v", event)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Schema) storeUsageEvent(tx *sql.Tx, event store.RawEvent) error {
	tableName := ""
	switch event.Kind {
	case "app":
		tableName = "app_usage_events"
	case "service":
		tableName = "service_usage_events"
	default:
		return fmt.Errorf("storeUsageEvent cannot store event of type %s", event.Kind)
	}
	stmt := fmt.Sprintf(`
		insert into %s (
			guid, created_at, raw_message
		) values (
			$1, $2, $3
		) on conflict do nothing
	`, tableName)
	_, err := tx.Exec(stmt, event.GUID, event.CreatedAt, event.RawMessage)
	return err
}

func (s *Schema) storeComposeEvent(tx *sql.Tx, event store.RawEvent) error {
	if event.Kind != "compose" {
		return fmt.Errorf("storeComposeEvent cannot store event of type %s", event.Kind)
	}
	stmt := fmt.Sprintf(`
		insert into compose_audit_events (
			event_id, created_at, raw_message
		) values (
			$1, $2, $3
		) on conflict do nothing
	`)
	_, err := tx.Exec(stmt, event.GUID, event.CreatedAt, event.RawMessage)
	return err
}

// GetEvents returns the RawEvents filtered using RawEventFilter if present
func (s *Schema) GetEvents(filter store.RawEventFilter) ([]store.RawEvent, error) {
	if filter.Kind == "" {
		return nil, fmt.Errorf("you must supply a kind to filter events by")
	}
	switch filter.Kind {
	case "app", "service":
		return s.getUsageEvents(filter)
	case "compose":
		return s.getComposeEvents(filter)
	}
	return nil, fmt.Errorf("cannot query events of kind '%s'", filter.Kind)
}

func (s *Schema) getComposeEvents(filter store.RawEventFilter) ([]store.RawEvent, error) {
	events := []store.RawEvent{}
	sortDirection := "desc"
	if filter.Reverse {
		sortDirection = "asc"
	}
	limit := ""
	if filter.Limit > 0 {
		limit = fmt.Sprintf(`limit %d`, filter.Limit)
	}
	if filter.Kind != "compose" {
		return nil, fmt.Errorf("getComposeEvents can not filter events of kind: %s", filter.Kind)
	}
	ctx, _ := context.WithTimeout(s.ctx, DefaultQueryTimeout)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`
		select
			event_id,
			created_at,
			raw_message
		from
			compose_audit_events
		order by
			id ` + sortDirection + `
		` + limit + `
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var event = store.RawEvent{Kind: filter.Kind}
		err := rows.Scan(
			&event.GUID,
			&event.CreatedAt,
			&event.RawMessage,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil

}

func (s *Schema) getUsageEvents(filter store.RawEventFilter) ([]store.RawEvent, error) {
	events := []store.RawEvent{}
	sortDirection := "desc"
	if filter.Reverse {
		sortDirection = "asc"
	}
	limit := ""
	if filter.Limit > 0 {
		limit = fmt.Sprintf(`limit %d`, filter.Limit)
	}
	tableName := ""
	switch filter.Kind {
	case "service":
		tableName = "service_usage_events"
	case "app":
		tableName = "app_usage_events"
	default:
		return nil, fmt.Errorf("getUsageEvents unknown kind: %s", filter.Kind)
	}
	ctx, _ := context.WithTimeout(s.ctx, DefaultQueryTimeout)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`
		select
			guid,
			created_at,
			raw_message
		from
			` + tableName + `
		order by
			id ` + sortDirection + `
		` + limit + `
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		event := store.RawEvent{Kind: filter.Kind}
		err := rows.Scan(
			&event.GUID,
			&event.CreatedAt,
			&event.RawMessage,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func checkVATRates(tx *sql.Tx) error {
	rows, err := tx.Query(`
		select distinct
			vat_code,
			valid_from,
			plan_guid
		from
			pricing_plan_components ppc
		where
			ppc.vat_code not in (
				select code 
				from vat_rates vr
				where vr.valid_from <= ppc.valid_from
			)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var valid string
		var guid string
		if err := rows.Scan(&code, &valid, &guid); err != nil {
			return err
		}
		return fmt.Errorf("missing vat_rate for '%s' for period '%s' required by plan '%s'", code, valid, guid)
	}

	return rows.Err()
}

func checkCurrencyRates(tx *sql.Tx) error {
	rows, err := tx.Query(`
		select distinct
			currency_code,
			valid_from,
			plan_guid
		from
			pricing_plan_components ppc
		where
			ppc.currency_code not in (
				select code 
				from currency_rates cr
				where cr.valid_from <= ppc.valid_from
			)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var valid string
		var guid string
		if err := rows.Scan(&code, &valid, &guid); err != nil {
			return err
		}
		return fmt.Errorf("missing currency_rate for '%s' for period '%s' required by plan '%s'", code, valid, guid)
	}

	return rows.Err()
}

// generateMissingPlans creates dummy plans with 0 cost at the epoch time
// useful for getting the system up with an existing dataset without configuring it properly
func generateMissingPlans(tx *sql.Tx) error {
	rows, err := tx.Query(`
		insert into pricing_plans (
			plan_guid, valid_from, name
		) (select distinct
			plan_guid,
			'epoch'::timestamptz,
			first_value(resource_type || ' ' || plan_name) over (partition by plan_guid order by lower(duration) desc)
		from events)
		returning plan_guid, name
	`)
	if err != nil {
		return wrapPqError(err, "generate-service-plan")
	}
	defer rows.Close()
	generated := []string{}
	for rows.Next() {
		var planGUID string
		var planName string
		if err := rows.Scan(&planGUID, &planName); err != nil {
			return err
		}
		generated = append(generated, fmt.Sprintf("%s (%s)", planName, planGUID))
	}
	if _, err := tx.Exec(`
		insert into pricing_plan_components (
			plan_guid, valid_from, name, formula, vat_code, currency_code
		) select distinct
			plan_guid,
			'epoch'::timestamptz,
			'pending',
			'0',
			'Standard'::vat_code,
			'GBP'::currency_code
		from events
	`); err != nil {
		return wrapPqError(err, "generate-service-plan-component")
	}
	for _, gen := range generated {
		fmt.Println("WARNING: generated epoch plan for:", gen)
	}
	return nil
}

// checkPricingComponents checks that all pricing plans have at least one pricing component
func checkPricingComponents(tx *sql.Tx) error {
	rows, err := tx.Query(`
		select
			pp.plan_guid,
			pp.valid_from,
			count(ppc.*) as component_count
		from
			pricing_plans pp
		left join
			pricing_plan_components ppc on pp.plan_guid = ppc.plan_guid
			and pp.valid_from = ppc.valid_from
		group by
			pp.plan_guid, pp.valid_from
		having
			count(ppc.*) < 0
	`)
	if err != nil {
		return wrapPqError(err, "unable to check pricing components")
	}
	defer rows.Close()
	missingComponents := []string{}
	for rows.Next() {
		var name string
		var guid string
		if err := rows.Scan(&name, &guid); err != nil {
			return err
		}
		missingComponents = append(missingComponents, fmt.Sprintf("%s: %s", guid, name))
	}
	if len(missingComponents) > 0 {
		return fmt.Errorf("%d existing services are not accounted for by the given pricing plans:\n    %s", len(missingComponents), strings.Join(missingComponents, "\n    "))
	}
	return nil
}

// checkPlanConsistancy reports an error if there are any plans in use in the
// the existing service_usage_events data that do not have corrosponding
// pricing_plans configured
func checkPlanConsistancy(tx *sql.Tx) error {
	rows, err := tx.Query(`
		with valid_pricing_plans as (
			select
				*,
				tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
					partition by plan_guid order by valid_from rows between current row and 1 following
				)) as valid_for
			from
				pricing_plans
		)
		select distinct
			plan_guid,	
			plan_name,
			resource_type
		from
			events
		where
			events.plan_guid not in (
				select plan_guid
				from valid_pricing_plans pp
				where pp.plan_guid = events.plan_guid
				and events.duration && pp.valid_for
			)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var planGUID string
		var planName string
		var resourceType string
		if err := rows.Scan(&planGUID, &planName, &resourceType); err != nil {
			return err
		}
		return fmt.Errorf("missing '%s' pricing plan configuration for '%s' (%s)", resourceType, planName, planGUID)
	}
	return nil
}

// execFile executes an sql file in the given transaction
func execFile(tx *sql.Tx, filename string) error {
	schemaFilename := schemaFile(filename)
	sql, err := ioutil.ReadFile(schemaFilename)
	if err != nil {
		return fmt.Errorf("failed to execute sql file %s: %s", schemaFilename, err)
	}
	_, err = tx.Exec(string(sql))
	if err != nil {
		return wrapPqError(err, schemaFilename)
	}
	return nil
}

func wrapPqError(err error, prefix string) error {
	msg := err.Error()
	if err, ok := err.(*pq.Error); ok {
		msg = err.Message
		if err.Detail != "" {
			msg += ": " + err.Detail
		}
		if err.Hint != "" {
			msg += ": " + err.Hint
		}
		if err.Where != "" {
			msg += ": " + err.Where
		}
	}
	return fmt.Errorf("%s: %s", prefix, msg)
}

func schemaDir() string {
	root := os.Getenv("APP_ROOT")
	if root == "" {
		root = os.Getenv("PWD")
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, "schema", "sql")
}

func schemaFile(filename string) string {
	return filepath.Join(schemaDir(), filename)
}

func LoadConfig(filename string) (Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
