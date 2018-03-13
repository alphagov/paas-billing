package api_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"time"

	"github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"
	"github.com/alphagov/paas-billing/fixtures"
	"github.com/alphagov/paas-billing/server"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Currency Conversion", func() {
	var (
		connstr       string
		sqlClient     *db.PostgresClient
		authenticator = AuthenticatedNonAdmin
	)

	ExpectJSON := func(actual interface{}, expected interface{}) {
		b1, err := json.Marshal(actual)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for expected value")
		b2, err := json.Marshal(expected)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for actual value")
		Expect(string(b1)).To(MatchJSON(string(b2)))
	}

	costOfUsage := func(days float64, currencyRate float64, fixedFee float64) float64 {
		return (days*24.0*3600.0 + fixedFee) * currencyRate * 100
	}

	doRequest := func(path string, v interface{}, params map[string]string) {
		u, err := url.Parse(path)
		Expect(err).ToNot(HaveOccurred())

		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequest("GET", u.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSONCharsetUTF8)
		req.Header.Set(echo.HeaderAuthorization, FakeBearerToken)

		e := server.New(sqlClient, authenticator, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		res := rec.Result()
		body, _ := ioutil.ReadAll(res.Body)
		Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))

		err = json.Unmarshal(body, v)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to unmarshal json: %s\nbody: %s", err, string(body)))
	}

	addCurrency := func(code string, rate float64, validFrom time.Time) {
		currencyRateFixtures := fixtures.CurrencyRates{{
			Code:      code,
			Rate:      rate,
			ValidFrom: validFrom,
		}}
		err := currencyRateFixtures.Insert(sqlClient)
		Expect(err).ToNot(HaveOccurred())
	}

	request := func(path string, from time.Time, to time.Time, v interface{}) {
		doRequest(path, v, map[string]string{
			"from": from.Format(time.RFC3339Nano),
			"to":   to.Format(time.RFC3339Nano),
		})
	}

	emptyOrgsFixture := func(guid string, spaceGUID string) fixtures.Orgs {
		gbpOrgsFixtures := fixtures.Orgs{}
		gbpOrgsFixtures[guid] = fixtures.Org{}
		gbpOrgsFixtures[guid][spaceGUID] = fixtures.Space{}
		return gbpOrgsFixtures
	}

	addEvent := func(fixture fixtures.Space, eventTime time.Time) fixtures.Space {
		newAppEvents := append(fixture.AppEvents, fixtures.AppEvent{
			AppGuid:               uuid.NewV4().String(),
			State:                 "STARTED",
			InstanceCount:         1,
			MemoryInMBPerInstance: 100,
			Time: eventTime,
		})
		fixture.AppEvents = newAppEvents

		// FIXME: Remove when fixed sql syntax error bug if there are
		// not service events.
		newServiceEvents := append(fixture.ServiceEvents, fixtures.ServiceEvent{
			ServiceInstanceGuid: uuid.NewV4().String(),
			State:               "SPAM",
			ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
			Time:                eventTime,
		})
		fixture.ServiceEvents = newServiceEvents

		return fixture
	}

	createComputePlan := func(planIdIndex int, validFrom time.Time, componentFormulas [][]string) fixtures.Plan {
		components := []fixtures.PricingPlanComponent{}
		for i, v := range componentFormulas {
			currency := v[0]
			formula := v[1]
			components = append(components,
				fixtures.PricingPlanComponent{
					ID:        101 + planIdIndex + i,
					Name:      "ComputePlanA/" + strconv.Itoa(i),
					Formula:   formula,
					VATRateID: 1,
					Currency:  currency,
				},
			)
		}

		return fixtures.Plan{
			ID:         10 + planIdIndex,
			Name:       "ComputePlanA",
			PlanGuid:   db.ComputePlanGuid,
			ValidFrom:  validFrom,
			Components: components,
		}
	}

	createScenario := func(
		guid string,
		spaceGUID string,
		startTime time.Time,
		appEventDeltas []time.Duration,
		planDeltas []time.Duration,
		componentFormulas [][]string,
	) {
		gbpOrgsFixtures := emptyOrgsFixture(guid, spaceGUID)
		for _, delta := range appEventDeltas {
			gbpOrgsFixtures[guid][spaceGUID] = addEvent(gbpOrgsFixtures[guid][spaceGUID], startTime.Add(delta))
		}

		gbpPlans := fixtures.Plans{}
		for i, delta := range planDeltas {
			gbpPlans = append(gbpPlans, createComputePlan(i, startTime.Add(delta), componentFormulas))
		}

		err := gbpOrgsFixtures.Insert(sqlClient, startTime)
		Expect(err).ToNot(HaveOccurred())

		err = gbpPlans.Insert(sqlClient)
		Expect(err).ToNot(HaveOccurred())

		err = sqlClient.UpdateViews()
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		connstr, err = dbhelper.CreateDB()
		Expect(err).ToNot(HaveOccurred())
		sqlClient, err = db.NewPostgresClient(connstr)
		Expect(err).ToNot(HaveOccurred())
		err = sqlClient.InitSchema()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := sqlClient.Close()
		Expect(err).ToNot(HaveOccurred())
		err = dbhelper.DropDB(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	It("does no conversion for resources that are billed in GBP for one event", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{0},
			[][]string{
				{"GBP", "$time_in_seconds + 1000"},
			},
		)

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice := costOfUsage(2, 1.0, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("does a conversion to GBP for resources billed in USD", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{0},
			[][]string{
				{"USD", "$time_in_seconds + 1000"},
			},
		)

		addCurrency("USD", 0.5, now)

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice := costOfUsage(2, 0.5, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("copes with multiple currency entries with same rate", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{0},
			[][]string{
				{"USD", "$time_in_seconds + 1000"},
			},
		)

		addCurrency("USD", 1, now)
		addCurrency("USD", 2, now.Add(24*1*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice :=
			costOfUsage(1, 1.0, 1000) +
				costOfUsage(1, 2.0, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("copes with multiple currency entries across multiple events", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{
				0,
				4 * 24 * time.Hour,
			},
			[]time.Duration{0},
			[][]string{
				{"USD", "$time_in_seconds + 1000"},
			},
		)

		addCurrency("USD", 1, now)
		addCurrency("USD", 0.5, now.Add(2*24*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(5*24*time.Hour),
			&out,
		)

		expectedPrice :=
			costOfUsage(2, 1.0, 1000) +
				costOfUsage(3, 0.5, 1000) +
				costOfUsage(1, 0.5, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("copes with multiple currency entries across multiple plans", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{
				0,
				31 * 24 * time.Hour,
			},
			[][]string{
				{"USD", "$time_in_seconds + 1000"},
			},
		)

		addCurrency("USD", 1, now)
		addCurrency("USD", 0.5, now.Add(2*24*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(60*24*time.Hour),
			&out,
		)

		expectedPrice :=
			costOfUsage(2, 1.0, 1000) +
				costOfUsage(29, 0.5, 1000) +
				costOfUsage(29, 0.5, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("applies only the currencies in the requested range", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{0},
			[][]string{
				{"USD", "$time_in_seconds + 1000"},
			},
		)

		addCurrency("USD", 1, now)
		addCurrency("USD", 0.5, now.Add(1*24*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now.Add(3*24*time.Hour),
			now.Add(4*24*time.Hour),
			&out,
		)

		expectedPrice := costOfUsage(1, 0.5, 1000)

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})

	It("applies different currencies for different components", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			now,
			[]time.Duration{0},
			[]time.Duration{0},
			[][]string{
				{"GBP", "$time_in_seconds + 1000"},
				{"USD", "$time_in_seconds + 2000"},
			},
		)

		addCurrency("USD", 2, now)
		addCurrency("USD", 4, now.Add(1*24*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			now,
			now.Add(3*24*time.Hour),
			&out,
		)

		expectedPriceFirstComponent := costOfUsage(3, 1, 1000)
		expectedPriceSecondComponent :=
			costOfUsage(1, 2, 2000) +
				costOfUsage(2, 4, 2000)

		expectedPrice := expectedPriceFirstComponent + expectedPriceSecondComponent

		ExpectJSON(out, []map[string]interface{}{
			{
				"org_guid":               guid,
				"space_guid":             spaceGUID,
				"price_in_pence_ex_vat":  expectedPrice,
				"price_in_pence_inc_vat": expectedPrice * 1.20,
			},
		})
	})
})
