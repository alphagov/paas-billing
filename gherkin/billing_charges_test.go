package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	"github.com/cucumber/godog"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"
)

// This is a simplified test suite designed only to be run on a local database for now.
// Later on it can be run alongside the existing unit tests on either a local or a remote CloudFoundry database.

var db *sql.DB

var (
	tempdb *testenv.TempDB
	err    error
)

const (
	pathToSqlDefinitions = "../eventstore/sql/"

	defaultResourceGuid = "11111111-1111-1111-1111-123456789123"
	defaultOrgGuid      = "22222222-2222-2222-2222-123456789123"
	defaultOrgName      = "test-org-name"
	defaultSpaceGuid    = "33333333-3333-3333-3333-123456789123"
	defaultSpaceName    = "test-space-name"
)

var logger = getDefaultLogger()

// Month and year for which billing consolidation is being run
var startInterval time.Time
var endInterval time.Time

// Run at the start of the tests
func InitializeTestSuite(ctx *godog.TestSuiteContext) {

	ctx.BeforeSuite(func() {
		fmt.Println("Connecting to the database")
		conn := "user=postgres dbname=postgres host=localhost sslmode=disable"
		db, err = sql.Open("postgres", conn)
		if err != nil {
			logger.Fatal("unable to connect to the database", err)
			os.Exit(1)
		}

		err = db.Ping()
		if err != nil {
			logger.Fatal("unable ping the database", err)
			os.Exit(1)
		}

		ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		es, err := eventstore.NewFromConfig(ctx2, db, logger, "../config.json")
		if err != nil {
			logger.Error("new-eventstore", err)
			os.Exit(1)
		}
		if err := es.Init(); err != nil {
			logger.Error("init-eventstore", err)
			os.Exit(1)
		}
	})

	ctx.AfterSuite(func() { fmt.Println("After running test suite") })
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	// Background
	ctx.Step(`^clear out data from database tables: ([A-Za-z_ ,]+)$`, clearDatabaseTables)
	ctx.Step(`^a clean billing database$`, aCleanBillingDatabase)

	// Given
	ctx.Step(`^(?:a|the) tenant has a ([A-Za-z_\- \.0-9]+) between \'(\d+)-(\d+)-(\d+) (\d+):(\d+):(\d+)\' and \'(\d+)-(\d+)-(\d+) (\d+):(\d+):(\d+)\'$`, aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMMss)
	ctx.Step(`^(?:a|the) tenant has a ([A-Za-z_\- \.0-9]+) between \'(\d+)-(\d+)-(\d+) (\d+):(\d+):(\d+)\' and \'(\d+)-(\d+)-(\d+) (\d+):(\d+)\'$`, aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMM)
	ctx.Step(`^(?:a|the) tenant has a ([A-Za-z_\- \.0-9]+) between \'(\d+)-(\d+)-(\d+) (\d+):(\d+)\' and \'(\d+)-(\d+)-(\d+)\'$`, aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmdd)
	ctx.Step(`^(?:a|the) tenant has a ([A-Za-z_\- \.0-9]+) between \'(\d+)-(\d+)-(\d+) (\d+):(\d+)\' and \'(\d+)-(\d+)-(\d+) (\d+):(\d+)\'$`, aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmddHHMM)
	ctx.Step(`^(?:a|the) tenant has a ([A-Za-z_\- \.0-9]+) between \'(\d+)-(\d+)-(\d+)\' and \'(\d+)-(\d+)-(\d+)\'$`, aTenantHasSomethingBetweenyyyymmddAndyyyymmdd)

	// When
	ctx.Step(`^billing is run for ([A-Za-z 0-9]+)$`, billingIsRun)

	// Then
	ctx.Step(`^the bill, including VAT, should be £(\d+)\.(\d+)$`, theBillShouldBe)
}

// Background

// Assumes tables being passed in as a comma-separated list. This is in case we want to use this function directly from Gherkin.
func clearDatabaseTables(tables string) error {

	fmt.Print("\n\n#################################################################\n")
	fmt.Printf("\nClearing out any existing data from tables populated in the tests (%s).\n\n", tables)

	tables = strings.Replace(tables, " ", "", -1)
	tableList := strings.Split(tables, ",")

	for _, table := range tableList {
		sqlQuery := fmt.Sprintf("DELETE FROM %s;", table)
		_, err := db.Query(sqlQuery)
		if err != nil {
			logger.Error("failed to clear table", err, lager.Data{"table": table})
			return err
		}
	}

	return nil
}

func aCleanBillingDatabase() error {
	// Clear out any data from previous tests.
	tableList := "compose_audit_events, consolidated_billable_events, consolidation_history, events, app_usage_events, service_usage_events, currency_rates, vat_rates, pricing_plans, pricing_plan_components"
	clearDatabaseTables(tableList)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// (re-)populate "static" tables
	es, err := eventstore.NewFromConfig(ctx, db, logger, "../config.json")
	if err != nil {
		return err
	}
	if err := es.Init(); err != nil {
		return err
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

// Given

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmdd(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmddHHMM(resource, fromYear, fromMonth, fromDay, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmddHHMMss(resource, fromYear, fromMonth, fromDay, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmdd(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMMss(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMM(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmddHHMMss(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmddHHMM(resource, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmdd(resource, fromYear, fromMonth, fromDay, toYear, toMonth, toDay string) error {
	return addEntryToBillableEventComponents(resource, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

func addEntryToBillableEventComponents(resource, fromDate, toDate string) error {
	// Add an entry to the events table
	eventGUID, err := uuid.NewV4()
	sqlQuery := fmt.Sprintf(`INSERT INTO events (event_guid,
		resource_guid,
		resource_name,
		resource_type,
		org_guid,
		org_name,
		space_guid,
		space_name,
		duration,
		plan_guid,
		plan_name,
		memory_in_mb,
		storage_in_mb,
		number_of_nodes)
		SELECT '%s', -- event-guid
		'%s', -- resource_guid
		'raw-msg-service-instance-name',
		'service', -- resource_type
		'%s', -- org-guid
		'%s',
		'%s', -- space-guid
		'%s',
		TSTZRANGE('%s', '%s'),
		p.plan_guid,
		p.name,
		p.memory_in_mb,
		p.storage_in_mb,
		p.number_of_nodes
		FROM pricing_plans p
		WHERE p.name = '%s';`, eventGUID.String(), defaultResourceGuid, defaultOrgGuid, defaultOrgName, defaultSpaceGuid, defaultSpaceName, fromDate, toDate, resource)

	_, err = db.Query(sqlQuery)
	if err != nil {
		panic(err)
	}

	return nil
}

// When

func billingIsRun(monthAndYear string) error {
	fmt.Printf("Running billing consolidation for %s\n", startInterval.String())
	monthAndYear = strings.TrimSpace(monthAndYear)
	startInterval, err = time.Parse("Jan 2006", monthAndYear)
	if err != nil {
		panic(err)
	}

	endInterval = startInterval.AddDate(0, 1, 0)

	// Need to add an entry to consolidation_history first.
	// We are not using the golang function for this, given this version of billing is going to change in the near future. We are just replicating the SQL the current version of billing runs.
	sqlQuery := fmt.Sprintf(`insert into consolidation_history (
		consolidated_range,
		created_at
	) values (
		tstzrange('%s', '%s'),
		NOW()
	);`, startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	_, err := db.Query(sqlQuery)
	if err != nil {
		panic(err)
	}

	// Run code to populate billable_event_components.
	content, err := ioutil.ReadFile(pathToSqlDefinitions + "create_billable_event_components.sql")
	if err != nil {
		panic(err)
	}
	sqlQuery = string(content)
	_, err = db.Query(sqlQuery)
	if err != nil {
		panic(err)
	}

	return nil
}

// Then

// The month and year must be passed in with a three-letter month. If the user is to pass in a full month name then need to write more code to convert it to three letters.
func theBillShouldBe(pounds, pence int) error {
	fmt.Printf("Running billing consolidation for interval: '%s' to '%s'.\n", startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	// We need to run the billing consolidation here.
	// Original code taken from paas-billing/eventstore/store_consolidated_billable_events.go:consolidate() and paas-billing/eventstore/store_billable_events.go:WithBillableEvents()
	sqlQuery := fmt.Sprintf(`with
		filtered_range as (
			select tstzrange('%s', '%s') as filtered_range -- durationArgPosition
		),
		components_with_price as (
			select
				b.event_guid,
				b.resource_guid,
				b.resource_name,
				b.resource_type,
				b.org_guid,
				b.org_name,
				b.space_guid,
				b.space_name,
				b.plan_guid,
				b.plan_name,
				b.duration * filtered_range as duration,
				b.number_of_nodes,
				b.memory_in_mb,
				b.storage_in_mb,
				b.component_name,
				b.component_formula,
				b.vat_code,
				b.vat_rate,
				'GBP' as currency_code,
				(eval_formula(
					b.memory_in_mb,
					b.storage_in_mb,
					b.number_of_nodes,
					b.duration * filtered_range,
					b.component_formula
				) * b.currency_rate) as price_ex_vat
			from
				filtered_range,
				billable_event_components b
			where
				duration && filtered_range
				-- filterQuery
			order by
				lower(duration) asc
		),
		billable_events as (
			select
				event_guid,
				min(lower(duration)) as event_start,
				max(upper(duration)) as event_stop,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				org_name,
				null::uuid as quota_definition_guid,
				space_guid,
				space_name,
				plan_guid,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb,
				json_build_object(
					'ex_vat', (sum(price_ex_vat))::text,
					'inc_vat', (sum(price_ex_vat * (1 + vat_rate)))::text,
					'details', json_agg(json_build_object(
						'name', component_name,
						'start', lower(duration),
						'stop', upper(duration),
						'plan_name', plan_name,
						'ex_vat', (price_ex_vat)::text,
						'inc_vat', (price_ex_vat * (1 + vat_rate))::text,
						'vat_rate', (vat_rate)::text,
						'vat_code', vat_code,
						'currency_code', currency_code
					))
				) as price
			from
				components_with_price
			group by
				event_guid,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				org_name,
				quota_definition_guid,
				space_guid,
				space_name,
				plan_guid,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb
			order by
				event_guid
	  )
	  `+`insert into consolidated_billable_events (
			consolidated_range,

			event_guid,
			duration,
			resource_guid,
			resource_name,
			resource_type,
			org_guid,
			org_name,
			space_guid,
			space_name,
			plan_guid,
			quota_definition_guid,
			number_of_nodes,
			memory_in_mb,
			storage_in_mb,
			price
		)
		select
			filtered_range,

			billable_events.event_guid,
			tstzrange(billable_events.event_start, billable_events.event_stop),
			billable_events.resource_guid,
			billable_events.resource_name,
			billable_events.resource_type,
			billable_events.org_guid,
			billable_events.org_name,
			billable_events.space_guid,
			billable_events.space_name,
			billable_events.plan_guid,
			billable_events.quota_definition_guid,
			billable_events.number_of_nodes,
			billable_events.memory_in_mb,
			billable_events.storage_in_mb,
			billable_events.price
		from
			billable_events,
			filtered_range;`, startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	_, err := db.Query(sqlQuery)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Examining billing bill for time interval: '%s' to '%s'...\n", startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	// Get the billing bill from the database
	rows, err := db.Query(`SELECT price->'ex_vat' AS ex_vat, price->'inc_vat' AS inc_vat FROM consolidated_billable_events;`)
	if err != nil {
		panic(err)
	}

	var intVATdb, exVATDB string
	var incVAT, exVAT float64
	for rows.Next() {
		err = rows.Scan(&exVATDB, &intVATdb)

		if err != nil {
			panic(err)
		}

		incVATCharge, err := strconv.ParseFloat(strings.Replace(intVATdb, "\"", "", -1), 64)
		if err != nil {
			panic(err)
		}

		exVATCharge, err := strconv.ParseFloat(strings.Replace(exVATDB, "\"", "", -1), 64)
		if err != nil {
			panic(err)
		}

		incVAT += incVATCharge
		exVAT += exVATCharge
	}

	if err = rows.Err(); err != nil {
		panic(err)
	}

	// Now examine the billing bill calculated by billing and check it's the same as that specified in the Gherkin test
	fmt.Printf("Bill calculated by billing excluding vat = £%f and including vat = £%f\n", exVAT, incVAT)

	exVAT = math.Round(exVAT*100) / 100
	incVAT = math.Round(incVAT*100) / 100

	// TODO: Investigate rounding in golang. The number 6.44448 is rounded to 6.44 not 6.45.

	expectedBill := float64((pounds*100)+pence) / 100
	if incVAT != expectedBill {
		return fmt.Errorf("Billing calculation is not as expected. Expected bill (from Gherkin test) = £%f, bill calculated by Paas billing = £%f\n", expectedBill, incVAT)
	} else {
		// Print in green
		fmt.Print("\033[32m", "\n*** Test passed ***\n\n")
	}

	return nil
}

func getDefaultLogger() lager.Logger {
	logger := lager.NewLogger("paas-billing-test")
	logLevel := lager.INFO
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		logLevel = lager.DEBUG
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
}
