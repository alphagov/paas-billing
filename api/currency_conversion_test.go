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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Currency Conversion", func() {
	var (
		connstr       string
		sqlClient     *db.PostgresClient
		authenticator = AuthenticatedNonAdmin
		start         = time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC)
	)

	ExpectJSON := func(actual interface{}, expected interface{}) {
		b1, err := json.Marshal(actual)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for expected value")
		b2, err := json.Marshal(expected)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for actual value")
		Expect(string(b1)).To(MatchJSON(string(b2)))
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
			AppGuid:               "00000001-0002-0001-0000-000000000000",
			State:                 "STARTED",
			InstanceCount:         1,
			MemoryInMBPerInstance: 100,
			Time: eventTime,
		})
		newServiceEvents := append(fixture.ServiceEvents, fixtures.ServiceEvent{
			ServiceInstanceGuid: "00000001-0002-0002-0000-000000000000",
			State:               "SPAM",
			ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
			Time:                eventTime,
		})
		fixture.AppEvents = newAppEvents
		fixture.ServiceEvents = newServiceEvents
		return fixture
	}

	createComputePlan := func(validFrom time.Time, componentFormulas [][]string) fixtures.Plans {
		components := []fixtures.PricingPlanComponent{}
		for i, v := range componentFormulas {
			currency := v[0]
			formula := v[1]
			components = append(components,
				fixtures.PricingPlanComponent{
					ID:        101 + i,
					Name:      "ComputePlanA/" + strconv.Itoa(i),
					Formula:   formula,
					VATRateID: 1,
					Currency:  currency,
				},
			)
		}

		return fixtures.Plans{
			{
				ID:         10,
				Name:       "ComputePlanA",
				PlanGuid:   db.ComputePlanGuid,
				ValidFrom:  validFrom,
				Components: components,
			},
		}
	}

	createScenario := func(
		guid string,
		spaceGUID string,
		startTime time.Time,
		componentFormulas [][]string,
	) {
		gbpOrgsFixtures := emptyOrgsFixture(guid, spaceGUID)
		gbpOrgsFixtures[guid][spaceGUID] = addEvent(gbpOrgsFixtures[guid][spaceGUID], startTime)

		gbpPlans := createComputePlan(start, componentFormulas)

		err := gbpOrgsFixtures.Insert(sqlClient, now)
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

	It("does no conversion for resources that are billed in GBP", func() {
		guid := "00000001-0000-0000-0000-000000000000"
		spaceGUID := "00000001-0001-0000-0000-000000000000"

		createScenario(
			guid,
			spaceGUID,
			start,
			[][]string{
				{"GBP", "$time_in_seconds"},
			},
		)

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			start,
			start.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice :=
			2 * // days
				24 * 3600 * // seconds in a day
				1.0 * // GBP => GBP
				100 // pences

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
			start,
			[][]string{
				{"USD", "$time_in_seconds"},
			},
		)

		addCurrency("USD", 0.5, start)

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			start,
			start.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice :=
			2 * // days
				24 * 3600 * // seconds in a day
				0.5 * // USD => GBP
				100 // pences

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
			start,
			[][]string{
				{"USD", "$time_in_seconds"},
			},
		)

		addCurrency("USD", 1, start)
		addCurrency("USD", 1, start.Add(24*1*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			start,
			start.Add(2*24*time.Hour),
			&out,
		)

		expectedPrice :=
			2 * // days
				24 * 3600 * // seconds in a day
				1.0 * // USD => GBP
				100 // pences

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
			start,
			[][]string{
				{"GBP", "$time_in_seconds * 1.0"},
				{"USD", "$time_in_seconds * 0.25"},
			},
		)

		addCurrency("USD", 2, start)
		addCurrency("USD", 4, start.Add(24*1*time.Hour))

		var out interface{}

		request(
			"/organisations/"+guid+"/spaces",
			start,
			start.Add(3*24*time.Hour),
			&out,
		)

		expectedPriceFirstComponent :=
			24 * 3600 * // seconds in a day
				1.0 * // price per second in component
				(1 + 1 + 1) * // GBP => GBP each day
				100 // pences

		expectedPriceSecondComponent :=
			24 * 3600 * // seconds in a day
				0.25 * // price per second in component
				(2 + 4 + 4) * // USD => GBP each day
				100 // pences

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
