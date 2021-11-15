package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/testenv"
	"github.com/cucumber/godog"
	"github.com/gofrs/uuid"
)

// This is a simplified test suite designed only to be run on a local database for now.
// Later on it can be run alongside the existing unit tests on either a local or a remote CloudFoundry database.

var db *sql.DB

var (
	tempdb *testenv.TempDB
	err    error
)

var pathToSqlDefinitions string
var pathToStaticTableData string

var defaultEventGuid string
var defaultResourceGuid string
var defaultResourceName string
var defaultResourceType string
var defaultOrgGuid string
var defaultOrgName string
var defaultSpaceGuid string
var defaultSpaceName string

// Month and year for which billing consolidation is being run
var startInterval time.Time
var endInterval time.Time

// The resource that the tenant is provisioning. This is what we are going to be calculating the bill for.
var tenantResource string

// Run at the start of the tests
func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	pathToSqlDefinitions = "../eventstore/sql/"
	pathToStaticTableData = "../billing-db/data/"

	defaultEventGuid = "00000000-0000-0000-0000-123456789123"
	defaultResourceGuid = "11111111-1111-1111-1111-123456789123"
	defaultResourceName = "gherkin_test_resource"
	defaultResourceType = "app"
	defaultOrgGuid = "22222222-2222-2222-2222-123456789123"
	defaultOrgName = "test-org-name"
	defaultSpaceGuid = "33333333-3333-3333-3333-123456789123"
	defaultSpaceName = "test-space-name"

	ctx.BeforeSuite(func() {
		fmt.Println("Connecting to the database")
		conn := "user=billinguser dbname=billing password=billinguser host=localhost sslmode=disable"
		db, err = sql.Open("postgres", conn)
		if err != nil {
			panic(err)
		}
		// defer db.Close()

		err = db.Ping()
		if err != nil {
			panic(err)
		}

		tables := []string{"../eventstore/sql/create_custom_types.sql",
			"../eventstore/sql/create_base_objects.sql",
			"../eventstore/sql/create_spaces.sql",
			"../eventstore/sql/create_service_usage_events.sql",
			"../eventstore/sql/create_app_usage_events.sql",
			"../eventstore/sql/create_services.sql",
			"../eventstore/sql/create_service_plans.sql",
			"../eventstore/sql/create_orgs.sql",
			"../eventstore/sql/create_compose_audit_events.sql",
			"../eventstore/sql/create_events.sql",
			"../eventstore/sql/create_custom_types.sql",
			"../eventstore/sql/create_consolidated_billable_events.sql",
			"../eventstore/sql/create_compose_audit_events.sql",
			"../billing-db/tables/resources.sql",
			"../billing-db/tables/charges.sql",
			"../billing-db/tables/vat_rates_new.sql",
			"../billing-db/tables/billing_formulae.sql",
			"../billing-db/tables/currency_rates.sql"}

		for i, table := range tables {
			_ = i
			// table = pathToSqlDefinitions + table
			fmt.Printf("Creating tables and other database objects in the file: %s...\n", table)
			content, err := ioutil.ReadFile(table)
			if err != nil {
				panic(err)
			}
			sql := string(content)
			rows, err := db.Query(sql)
			if err != nil {
				panic(err)
			}
			_ = rows
		}

		// defer db.Close()
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
	ctx.Step(`^billing is run$`, billingIsRun)

	// Then
	ctx.Step(`^the bill, including VAT, for ([A-Za-z 0-9]+) should be £(\d+)\.(\d+)$`, theBillShouldBe)
}

// Background

// Assumes tables being passed in as a comma-separated list. This is in case we want to use this function directly from Gherkin.
func clearDatabaseTables(tables string) error {

	fmt.Print("\n\n#################################################################\n")
	fmt.Printf("\nClearing out any existing data from tables populated in the tests (%s).\n\n", tables)

	tables = strings.Replace(tables, " ", "", -1)
	tableList := strings.Split(tables, ",")

	for i := 0; i < len(tableList); i++ {
		sql := fmt.Sprintf("TRUNCATE TABLE %s;", tableList[i])
		// fmt.Printf("Running '%s'\n", sql)
		rows, err := db.Query(sql)
		if err != nil {
			panic(err)
		}
		_ = rows
	}

	return nil
}

func aCleanBillingDatabase() error {
	// Clear out any data from previous tests.
	tableList := "resources, events, app_usage_events, service_usage_events, currency_exchange_rates, vat_rates_new, charges"
	clearDatabaseTables(tableList)

	// Add data to the following tables: vat_rates, currency_rates, pricing_plans, pricing_plan_components. Use the data in paas-billing/billing-db/data.
	tables := []string{"currency_exchange_rates",
		"vat_rates_new",
		"charges"}

	// TODO: We need to refresh a copy of the database using a copy of the data in paas-cf. Do not use the code below to refresh from data files.

	for i, table := range tables {
		_ = i
		fmt.Printf("Populating data in %s table...\n", table)
		table = pathToStaticTableData + table + ".dat"
		content, err := ioutil.ReadFile(table)
		if err != nil {
			panic(err)
		}
		sql := string(content)
		rows, err := db.Query(sql)
		if err != nil {
			panic(err)
		}
		_ = rows
	}

	return nil
}

// Given

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmdd(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmddHHMM(planName, fromYear, fromMonth, fromDay, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmddHHMMss(planName, fromYear, fromMonth, fromDay, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmdd(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMMss(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM:ss' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddHHMMssAndyyyymmddHHMM(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute, fromSecond), fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd HH:MM:ss'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmddHHMMss(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay, toHour, toMinute, toSecond string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s %s:%s:%s", toYear, toMonth, toDay, toHour, toMinute, toSecond))
}

// Datetime args are in the form 'yyyy-mm-dd HH:MM' and 'yyyy-mm-dd HH:MM'
func aTenantHasSomethingBetweenyyyymmddHHMMAndyyyymmddHHMM(planName, fromYear, fromMonth, fromDay, fromHour, fromMinute, toYear, toMonth, toDay, toHour, toMinute string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s %s:%s", fromYear, fromMonth, fromDay, fromHour, fromMinute)+":00", fmt.Sprintf("%s-%s-%s %s:%s", toYear, toMonth, toDay, toHour, toMinute)+":00")
}

// Datetime args are in the form 'yyyy-mm-dd' and 'yyyy-mm-dd'
func aTenantHasSomethingBetweenyyyymmddAndyyyymmdd(planName, fromYear, fromMonth, fromDay, toYear, toMonth, toDay string) error {
	return addEntryToBilling(planName, fmt.Sprintf("%s-%s-%s", fromYear, fromMonth, fromDay)+" 00:00:00", fmt.Sprintf("%s-%s-%s", toYear, toMonth, toDay)+" 00:00:00")
}

func addEntryToBilling(planName, fromDate, toDate string) error {
	// fmt.Printf("resource = '%s', from date = '%s', to date = '%s'\n", resource, fromDate, toDate)
	// Add an entry to the events table
	event_guid, err := uuid.NewV4()
	sql := fmt.Sprintf(`INSERT INTO resources
	(
		valid_from, 
		valid_to,
		resource_guid,
		resource_name,
		resource_type,
		org_guid,
		org_name,
		space_guid,
		space_name,
		plan_name,
		plan_guid,
		storage_in_mb
		memory_in_mb,
		number_of_nodes,
		cf_event_guid,
		last_updated
	)
	SELECT	'%s', -- valid_from
			'%s', -- valid_to
			'%s', -- resource_guid
			'%s', -- resource_name
			'%s', -- resource_type
			'%s', -- org_guid
			'%s', -- org_name
			'%s', -- space_guid
			'%s', -- space_name
			c.plan_name,
			c.plan_guid,
			c.storage_in_mb,
			c.memory_in_mb,
			c.number_of_nodes,
			'%s', -- cf_event_guid
			NOW()
	FROM charges c WHERE c.plan_name = '%s';`, // TODO: Also filter on valid_from/valid_to since may need more than one entry here. Can use range filters since performance of these tests not an issue.
		fromDate,
		toDate,
		defaultResourceGuid,
		defaultResourceName,
		defaultResourceType,
		defaultOrgGuid,
		defaultOrgName,
		defaultSpaceGuid,
		defaultSpaceName,
		event_guid.String(), /* cf_event_guid */
		planName)

	// fmt.Printf("Adding row to events table (%s)...\n", sql[0:400])

	rows, err := db.Query(sql)
	if err != nil {
		panic(err)
	}
	_ = rows

	fmt.Printf("Bill will be calculated for the interval: '%s' to '%s'.\n", startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	return nil
}

// When

// Empty function.
func billingIsRun() error {
	return nil
}

// Then

// The month and year must be passed in with a three-letter month. If the user is to pass in a full month name then need to write more code to convert it to three letters.
// We will need to enhance this to accept dates within a month. Currently, the code has only been written for complete months.
func theBillShouldBe(monthAndYear string, pounds, pence int) error {
	monthAndYear = strings.TrimSpace(monthAndYear)
	startInterval, err = time.Parse("Jan 2006", monthAndYear)
	if err != nil {
		panic(err)
	}

	// Add a month
	endInterval = startInterval.AddDate(0, 1, 0)

	// We are not using the golang function for this, given this version of billing is going to change in the near future. We are just replicating the SQL the current version of billing runs.
	sql := fmt.Sprintf(`SELECT SUM(charge_gbp_exc_vat) AS ex_vat_db, SUM(charge_gbp_inc_vat) AS ex_vat_db FROM calculate_bill();`)

	rows, err := db.Query(sql)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Examining billing bill for the time interval: '%s' to '%s'...\n", startInterval.Format("2006-01-02"), endInterval.Format("2006-01-02"))

	var inc_vat_db, ex_vat_db string
	var inc_vat, ex_vat float64
	for rows.Next() {
		err = rows.Scan(&ex_vat_db, &inc_vat_db)

		if err != nil {
			panic(err)
		}

		inc_vat_charge, err := strconv.ParseFloat(strings.Replace(inc_vat_db, "\"", "", -1), 64)
		if err != nil {
			panic(err)
		}

		ex_vat_charge, err := strconv.ParseFloat(strings.Replace(ex_vat_db, "\"", "", -1), 64)
		if err != nil {
			panic(err)
		}

		inc_vat += inc_vat_charge
		ex_vat += ex_vat_charge
	}

	if err = rows.Err(); err != nil {
		panic(err)
	}

	// Now examine the billing bill calculated by billing and check it's the same as that specified in the Gherkin test
	fmt.Printf("Bill calculated by billing excluding vat = £%f and including vat = £%f\n", ex_vat, inc_vat)

	ex_vat = math.Round(ex_vat*100) / 100
	inc_vat = math.Round(inc_vat*100) / 100

	// TODO: Investigate rounding in golang. The number 6.44448 is rounded to 6.44 not 6.45.

	expectedBill := float64((pounds*100)+pence) / 100
	if inc_vat != expectedBill {
		return fmt.Errorf("Billing calculation is not as expected. Expected bill (from Gherkin test) = £%f, bill calculated by Paas billing = £%f\n", expectedBill, inc_vat)
	} else {
		// Print in green
		fmt.Print(string("\033[32m"), "\n*** Test passed ***\n\n")
	}

	return nil
}
