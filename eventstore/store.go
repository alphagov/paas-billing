package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	AppUsageTableName     = "app_usage_events"
	ServiceUsageTableName = "service_usage_events"
	ComputePlanGUID       = "f4d4b95a-f55e-4593-8d54-3364c25798c4"
	ComputeServiceGUID    = "4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b"
	TaskPlanGUID          = "ebfa9453-ef66-450c-8c37-d53dfd931038"
	StagingPlanGUID       = "9d071c77-7a68-4346-9981-e8dafac95b6f"
	DefaultInitTimeout    = 25 * time.Minute
	DefaultRefreshTimeout = 700 * time.Minute
	DefaultStoreTimeout   = 45 * time.Second
	DefaultQueryTimeout   = 45 * time.Second
)

var (
	eventStorePerformanceGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "performance",
			Help:      "Elapsed time for EventStore functions (in seconds)",
		},
		[]string{"function", "error"})
)

var (
	missingPlansCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "dummy_plans_created",
			Help:      "Count of missing plans (for which dummy plans have been created)",
		}, []string{"guid", "name"})
	inconsistentPlansCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "inconsistent_plans",
			Help:      "Count of inconsistent plans",
		})

	currencyRatioGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "currency_configured_ratio",
			Help:      "Configured ratio for GBP:$code",
		}, []string{"code"})

	vatRateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "vat_configured_rate",
			Help:      "Configured vat rate for $code",
		}, []string{"code"})

	totalCostGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "paas_billing",
			Subsystem: "eventstore",
			Name:      "total_cost_gbp",
			Help:      "Total costs",
		}, []string{"kind", "plan", "plan_guid"})
)

var _ eventio.EventStore = &EventStore{}

type EventStore struct {
	db     *sql.DB
	cfg    Config
	logger lager.Logger
	ctx    context.Context
}

func New(ctx context.Context, db *sql.DB, logger lager.Logger, cfg Config) *EventStore {
	return &EventStore{
		db:     db,
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
	}
}

func NewFromConfig(ctx context.Context, db *sql.DB, logger lager.Logger, filename string) (*EventStore, error) {
	cfg, err := LoadConfig(filename)
	if err != nil {
		return nil, err
	}
	return New(ctx, db, logger, cfg), nil
}

type migrateLagerLogger struct {
	lagerLogger lager.Logger
}

func newMigrateLagerLogger(lagerLogger lager.Logger) migrateLagerLogger {
	mll := migrateLagerLogger{}
	mll.lagerLogger = lagerLogger
	return mll
}

func (mll migrateLagerLogger) Printf(format string, v ...interface{}) {
	mll.lagerLogger.Info(fmt.Sprintf(format, v...), lager.Data{})
}

func (mll migrateLagerLogger) Verbose() bool {
	return false
}

// Init initialises the database tables and functions
func (s *EventStore) Init() error {
	s.logger.Info("initializing")
	ctx, cancel := context.WithTimeout(s.ctx, DefaultInitTimeout)
	defer cancel()

	migrateConnection, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer migrateConnection.Close()

	// `WithInstance` was causing issues: having a deferred `Close()` on the driver would close the underlying SQL
	// instance, not just the connection that migrate created.
	migrateDriver, err := migrate_postgres.WithConnection(ctx, migrateConnection, &migrate_postgres.Config{})

	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsDir()),
		"postgres",
		migrateDriver,
	)
	if err != nil {
		return err
	}

	m.Log = newMigrateLagerLogger(s.logger.Session("migrate", lager.Data{}))

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.initVATRates(tx); err != nil {
		return fmt.Errorf("failed to init VAT rates: %s", err)
	}
	if err := s.initCurrencyRates(tx); err != nil {
		return fmt.Errorf("failed to init currency rates: %s", err)
	}
	if err := s.initPlans(tx); err != nil {
		return fmt.Errorf("failed to init plans: %s", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	s.logger.Info("initialized")
	return nil
}

func (s *EventStore) Ping() error {
	s.logger.Debug("Ping DB")
	return s.db.Ping()
}

// Refresh triggers regeneration of the cached normalized view of the event dat and rebuilds the
// billable components. Ideally you should do this once a day
func (s *EventStore) Refresh() error {
	return s.regenerateEvents()
}

func (s *EventStore) RecordPeriodicMetrics() error {
	return errors.Join(
		s.recordTotalCostMetrics(),
	)
}

func (s *EventStore) regenerateEvents() error {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultRefreshTimeout)
	defer cancel()

	if err := s.runSQLFilesInTransaction(
		ctx,
		"create_events.sql",
	); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if s.cfg.IgnoreMissingPlans {
		if err := s.generateMissingPlans(tx); err != nil {
			return err
		}
	}

	if err := checkPlanConsistency(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := s.runSQLFilesInTransaction(
		ctx,
		"create_billable_event_components.sql",
	); err != nil {
		return err
	}

	if err := s.runSQLFilesInTransaction(
		ctx,
		"create_view_billable_event_components_by_day.sql",
	); err != nil {
		return err
	}

	return nil
}

func (s *EventStore) initVATRates(tx *sql.Tx) error {
	if _, err := tx.Exec("DELETE FROM vat_rates"); err != nil {
		return wrapPqError(err, "error deleting existing vat_rates")
	}

	for _, vr := range s.cfg.VATRates {
		vatRateGauge.With(prometheus.Labels{"code": vr.Code}).Set(vr.Rate)
		s.logger.Info("configuring-vat-rate", lager.Data{
			"code":       vr.Code,
			"valid_from": vr.ValidFrom,
			"rate":       vr.Rate,
		})
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

func (s *EventStore) initCurrencyRates(tx *sql.Tx) error {
	if _, err := tx.Exec("DELETE FROM currency_rates"); err != nil {
		return wrapPqError(err, "error deleting existing currency_rates")
	}

	for _, cr := range s.cfg.CurrencyRates {
		currencyRatioGauge.With(prometheus.Labels{"code": cr.Code}).Set(cr.Rate)
		s.logger.Info("configuring-currency-rate", lager.Data{
			"code":       cr.Code,
			"valid_from": cr.ValidFrom,
			"rate":       cr.Rate,
		})
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
// it will fail to update plans and return an error, indicating the transaction
// should be rolled back
func (s *EventStore) initPlans(tx *sql.Tx) (err error) {
	if _, err := tx.Exec("DELETE FROM pricing_plan_components"); err != nil {
		return wrapPqError(err, "error deleting existing pricing_plan_components")
	}

	if _, err := tx.Exec("DELETE FROM pricing_plans"); err != nil {
		return wrapPqError(err, "error deleting existing pricing_plans")
	}

	for _, pp := range s.cfg.PricingPlans {
		s.logger.Info("configuring-pricing-plan", lager.Data{
			"plan_guid":  pp.PlanGUID,
			"name":       pp.Name,
			"valid_from": pp.ValidFrom,
		})
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
			s.logger.Info("configuring-pricing-plan-component", lager.Data{
				"plan_guid":  pp.PlanGUID,
				"name":       ppc.Name,
				"valid_from": pp.ValidFrom,
			})
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

	if err := checkPricingComponents(tx); err != nil {
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

func (s *EventStore) StoreEvents(events []eventio.RawEvent) error {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultStoreTimeout)
	defer cancel()
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
	return tx.Commit()
}

func (s *EventStore) storeUsageEvent(tx *sql.Tx, event eventio.RawEvent) error {
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

func (s *EventStore) storeComposeEvent(tx *sql.Tx, event eventio.RawEvent) error {
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

// GetEvents returns the eventio.RawEvents filtered using eventio.RawEventFilter if present
func (s *EventStore) GetEvents(filter eventio.RawEventFilter) ([]eventio.RawEvent, error) {
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

func (s *EventStore) getComposeEvents(filter eventio.RawEventFilter) ([]eventio.RawEvent, error) {
	events := []eventio.RawEvent{}
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
	ctx, cancel := context.WithTimeout(s.ctx, DefaultQueryTimeout)
	defer cancel()
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
		var event = eventio.RawEvent{Kind: filter.Kind}
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

func (s *EventStore) getUsageEvents(filter eventio.RawEventFilter) ([]eventio.RawEvent, error) {
	events := []eventio.RawEvent{}
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
	ctx, cancel := context.WithTimeout(s.ctx, DefaultQueryTimeout)
	defer cancel()
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
		event := eventio.RawEvent{Kind: filter.Kind}
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
// for every single plan in events, unless there is already one.
// Useful for getting the system up with an existing dataset without
// configuring it properly
func (s *EventStore) generateMissingPlans(tx *sql.Tx) error {
	rows, err := tx.Query(`
		insert into pricing_plans (
			plan_guid, valid_from, name
		) (
			select
				distinct plan_guid,
				'epoch'::timestamptz,
				first_value(resource_type || ' ' || plan_name)
				over (
					partition by plan_guid
					order by lower(duration) desc
				)
			from events
			where plan_guid not in (
				select distinct plan_guid
				from pricing_plans pp
				where pp.plan_guid = events.plan_guid
				and valid_from = 'epoch'::timestamptz
			)
		)
		returning plan_guid, name
	`)
	if err != nil {
		return wrapPqError(err, "generate-service-plan")
	}
	defer rows.Close()
	for rows.Next() {
		var planGUID string
		var planName string
		if err := rows.Scan(&planGUID, &planName); err != nil {
			return err
		}
		missingPlansCounter.WithLabelValues(planGUID, planName).Inc()
		s.logger.Info("generate-missing-plan", lager.Data{
			"message":    "generating dummy pricing plan",
			"hint":       "disable IgnoreMissingPlans and ensure all pricing plans are configure correctly",
			"plan_guid":  planGUID,
			"plan_name":  planName,
			"valid_from": "epoch",
		})
	}
	if _, err := tx.Exec(`
		insert into pricing_plan_components (
			plan_guid, valid_from, name, formula, vat_code, currency_code
		) (
			select distinct
				plan_guid,
				'epoch'::timestamptz,
				'pending',
				'0',
				'Standard'::vat_code,
				'GBP'::currency_code
			from events
			where plan_guid not in (
				select distinct plan_guid
				from pricing_plan_components ppc
				where ppc.plan_guid = events.plan_guid
				and valid_from = 'epoch'::timestamptz
			)
		)`,
	); err != nil {
		return wrapPqError(err, "generate-service-plan-component")
	}
	return nil
}

func (s *EventStore) recordPeriodicMetrics() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
		s.recordTotalCostMetrics()
	}
}

func (s *EventStore) recordTotalCostMetrics() error {
	costs, err := s.GetTotalCost()
	if err != nil {
		return err
	}
	for _, c := range costs {
		totalCostGauge.With(prometheus.Labels{
			"plan_guid": c.PlanGUID,
			"kind":      c.Kind,
			"plan":      c.PlanName,
		}).Set(float64(c.Cost))
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

// checkPlanConsistency reports an error if there are any plans in use in the
// the existing service_usage_events data that do not have corresponding
// pricing_plans configured
func checkPlanConsistency(tx *sql.Tx) error {
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
		inconsistentPlansCounter.Inc()
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

func (s *EventStore) runSQLFilesInTransaction(ctx context.Context, filenames ...string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, filename := range filenames {
		if err := s.runSQLFile(tx, filename); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *EventStore) runSQLFile(tx *sql.Tx, filename string) error {
	startTime := time.Now()
	s.logger.Info("run-sql-file", map[string]interface{}{"sqlFile": filename})

	sqlFilename := sqlFile(filename)
	sql, err := ioutil.ReadFile(sqlFilename)
	if err != nil {
		err = fmt.Errorf("failed to execute sql file %s: %s", sqlFilename, err)
		elapsed := time.Since(startTime)
		eventStorePerformanceGauge.WithLabelValues(
			fmt.Sprintf("runSQLFile:%s", filename), err.Error()).Set(elapsed.Seconds())
		s.logger.Error("finish-sql-file", err, lager.Data{
			"sqlFile": filename,
			"elapsed": int64(elapsed),
		})
		return err
	}

	_, err = tx.Exec(string(sql))
	elapsed := time.Since(startTime)
	if err != nil {
		err = wrapPqError(err, sqlFilename)
		elapsed := time.Since(startTime)
		eventStorePerformanceGauge.WithLabelValues(
			fmt.Sprintf("runSQLFile:%s", filename), err.Error()).Set(elapsed.Seconds())
		s.logger.Error("finish-sql-file", err, lager.Data{
			"sqlFile": filename,
			"elapsed": int64(elapsed),
		})
		return err
	}
	eventStorePerformanceGauge.WithLabelValues(
		fmt.Sprintf("runSQLFile:%s", filename), "").Set(elapsed.Seconds())

	s.logger.Info("finish-sql-file", lager.Data{
		"sqlFile": filename,
		"elapsed": int64(elapsed),
	})
	return nil
}

// queryJSON returns rows as a json blobs, which makes it easier to decode into structs.
func queryJSON(tx *sql.Tx, q string, args ...interface{}) (*sql.Rows, error) {
	return tx.Query(fmt.Sprintf(`
		with q as ( %s )
		select row_to_json(q.*) from q;
	`, q), args...)
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

func appRoot() string {
	root := os.Getenv("APP_ROOT")
	if root == "" {
		root = os.Getenv("PWD")
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	return root
}

func migrationsDir() string {
	return filepath.Join(appRoot(), "eventstore", "migrations")
}

func sqlDir() string {
	return filepath.Join(appRoot(), "eventstore", "sql")
}

func sqlFile(filename string) string {
	return filepath.Join(sqlDir(), filename)
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
