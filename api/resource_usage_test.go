package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/auth"
	"github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"
	"github.com/alphagov/paas-billing/server"
	"github.com/labstack/echo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func agoRFC3339(d time.Duration) string {
	return ago(d).Format(time.RFC3339)
}

func monthsAgoRFC3339(months int) string {
	return monthsAgo(months).Format(time.RFC3339)
}

var nowRFC3339 = now.Format(time.RFC3339)

var _ = Describe("API", func() {

	var (
		sqlClient     *db.PostgresClient
		connstr       string
		authenticator auth.Authenticator
	)

	BeforeEach(func() {
		var err error
		connstr, err = dbhelper.CreateDB()
		Expect(err).ToNot(HaveOccurred())
		sqlClient, err = db.NewPostgresClient(connstr)
		Expect(err).ToNot(HaveOccurred())
		err = sqlClient.InitSchema()
		Expect(err).ToNot(HaveOccurred())

		err = planFixtures.Insert(sqlClient)
		Expect(err).ToNot(HaveOccurred())

		err = orgsFixtures.Insert(sqlClient, now)
		Expect(err).ToNot(HaveOccurred())

		err = sqlClient.UpdateViews()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := sqlClient.Close()
		Expect(err).ToNot(HaveOccurred())
		err = dbhelper.DropDB(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	var doRequest = func(path string, v interface{}, params map[string]string) {
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

	var requestRanged = func(url string, from time.Time, to time.Time, v interface{}) {
		doRequest(url, v, map[string]string{
			"from": from.Format(time.RFC3339Nano),
			"to":   to.Format(time.RFC3339Nano),
		})
	}

	var send = func(method, ct, path string, data io.Reader) (int, interface{}) {
		var v interface{}
		req, err := http.NewRequest(method, path, data)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSONCharsetUTF8)
		req.Header.Set(echo.HeaderContentType, ct)
		req.Header.Set(echo.HeaderAuthorization, FakeBearerToken)

		e := server.New(sqlClient, authenticator, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		res := rec.Result()
		body, _ := ioutil.ReadAll(res.Body)

		err = json.Unmarshal(body, &v)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to unmarshal json: %s\nbody: %s", err, string(body)))
		return res.StatusCode, v
	}

	var get = func(path string) (int, interface{}) {
		return send("GET", echo.MIMEApplicationForm, path, bytes.NewReader(nil))
	}

	var post = func(path string, data io.Reader) (int, interface{}) {
		return send("POST", echo.MIMEApplicationForm, path, data)
	}

	var del = func(path string, data io.Reader) (int, interface{}) {
		return send("DELETE", echo.MIMEApplicationForm, path, data)
	}

	var put = func(path string, data io.Reader) (int, interface{}) {
		return send("PUT", echo.MIMEApplicationForm, path, data)
	}

	var ExpectJSON = func(v1 interface{}, v2 interface{}) {
		b1, err := json.Marshal(v1)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for expected value")
		b2, err := json.Marshal(v2)
		Expect(err).ToNot(HaveOccurred(), "ExpectJSON failed for actual value")
		Expect(string(b1)).To(MatchJSON(string(b2)))
	}

	var itShouldFetchPricingPlanComponents = func() {
		path := "/pricing_plan_components"
		var out interface{}
		doRequest(path, &out, map[string]string{})
		ExpectJSON(out, []map[string]interface{}{
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
				"id":              101,
				"name":            "ComputePlanA/1",
				"pricing_plan_id": 10,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.3",
				"id":              102,
				"name":            "ComputePlanA/2",
				"pricing_plan_id": 10,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
				"id":              111,
				"name":            "ComputePlanB/1",
				"pricing_plan_id": 11,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * 0.2",
				"id":              201,
				"name":            "ServicePlanA/1",
				"pricing_plan_id": 20,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * 0.3",
				"id":              202,
				"name":            "ServicePlanA/2",
				"pricing_plan_id": 20,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * 1",
				"id":              301,
				"name":            "ServicePlanB/1",
				"pricing_plan_id": 30,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * 0.2",
				"id":              401,
				"name":            "With standard VAT",
				"pricing_plan_id": 40,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * 0.3",
				"id":              402,
				"name":            "With zero VAT",
				"pricing_plan_id": 40,
				"vat_rate_id":     2,
			},
		})
	}

	var itShouldFetchPricingPlanComponentsByPlan = func() {
		path := "/pricing_plans/10/components"
		var out interface{}
		doRequest(path, &out, map[string]string{})
		ExpectJSON(out, []map[string]interface{}{
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
				"id":              101,
				"name":            "ComputePlanA/1",
				"pricing_plan_id": 10,
				"vat_rate_id":     1,
			},
			{
				"currency":        "GBP",
				"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.3",
				"id":              102,
				"name":            "ComputePlanA/2",
				"pricing_plan_id": 10,
				"vat_rate_id":     1,
			},
		})
	}

	var itShouldFetchPricingPlanComponentById = func() {
		path := "/pricing_plan_components/101"
		var out interface{}
		doRequest(path, &out, map[string]string{})
		ExpectJSON(out, map[string]interface{}{
			"currency":        "GBP",
			"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
			"id":              101,
			"name":            "ComputePlanA/1",
			"pricing_plan_id": 10,
			"vat_rate_id":     1,
		})
	}

	var itShouldFetchVATRates = func() {
		path := "/vat_rates"
		var out interface{}
		doRequest(path, &out, map[string]string{})
		ExpectJSON(out, []map[string]interface{}{
			{
				"id":   1,
				"name": "Standard",
				"rate": 0.2,
			},
			{
				"id":   2,
				"name": "Zero rate",
				"rate": 0,
			},
		})
	}

	var itShouldFetchVATRateByID = func() {
		path := "/vat_rates/1"
		var out interface{}
		doRequest(path, &out, map[string]string{})
		ExpectJSON(out, map[string]interface{}{
			"id":   1,
			"name": "Standard",
			"rate": 0.2,
		})
	}

	Context("As non admin", func() {

		BeforeEach(func() {
			authenticator = AuthenticatedNonAdmin
		})

		Context("/organisations", func() {

			var (
				path = "/organisations"
			)

			It("should return pricing totals for each org", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9932850,
						"price_in_pence_inc_vat": 9932850 * 1.2,
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  294750,
						"price_in_pence_inc_vat": 294750 * 1.2,
					},
					{
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (24*0.2 + 24*0.3) * 100,
						"price_in_pence_inc_vat": (24*0.2*1.2 + 24*0.3) * 100,
					},
				})
			})

			It("should only return org total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  422450,
						"price_in_pence_inc_vat": 422450 * 1.2,
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  76850,
						"price_in_pence_inc_vat": 76850 * 1.2,
					},
					{
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (1*0.2 + 1*0.3) * 100,
						"price_in_pence_inc_vat": (1*0.2*1.2 + 1*0.3) * 100,
					},
				})
			})

		})

		Context("/organisations/:org_id", func() {

			var (
				guid = "00000001-0000-0000-0000-000000000000"
				path = "/organisations/" + guid
			)

			It("should return org total", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, map[string]interface{}{
					"org_guid":               guid,
					"price_in_pence_ex_vat":  9932850,
					"price_in_pence_inc_vat": 9932850 * 1.2,
				})
			})

			It("should only return totals for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"org_guid":               guid,
					"price_in_pence_ex_vat":  422450,
					"price_in_pence_inc_vat": 422450 * 1.2,
				})
			})

		})

		Context("/organisations/:org_id/spaces", func() {

			var (
				guid = "00000001-0000-0000-0000-000000000000"
				path = "/organisations/" + guid + "/spaces"
			)

			It("should return space totals for the given org", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  716850,
						"price_in_pence_inc_vat": 716850 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/organisations/:org_id/resources", func() {

			var (
				guid = "00000001-0000-0000-0000-000000000000"
				path = "/organisations/" + guid + "/resources"
			)

			It("should return resource totals for the given org", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":                   "00000001-0001-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  716800,
						"price_in_pence_inc_vat": 716800 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0002-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/spaces", func() {

			var (
				path = "/spaces"
			)

			It("should return pricing totals for each space", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  716850,
						"price_in_pence_inc_vat": 716850 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  294700,
						"price_in_pence_inc_vat": 294700 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000002-0002-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (24*0.2 + 24*0.3) * 100,
						"price_in_pence_inc_vat": (24*0.2*1.2 + 24*0.3) * 100,
						"space_guid":             "00000003-0005-0000-0000-000000000000",
					},
				})
			})

			It("should only return space total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  12800,
						"price_in_pence_inc_vat": 12800 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  409650,
						"price_in_pence_inc_vat": 409650 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  76800,
						"price_in_pence_inc_vat": 76800 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000002-0002-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (1*0.2 + 1*0.3) * 100,
						"price_in_pence_inc_vat": (1*0.2*1.2 + 1*0.3) * 100,
						"space_guid":             "00000003-0005-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/spaces/:space_id/resources", func() {

			var (
				guid = "00000001-0001-0000-0000-000000000000"
				path = "/spaces/" + guid + "/resources"
			)

			It("should return resource totals for the given space", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":                   "00000001-0001-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/spaces/:space_guid", func() {

			var (
				guid = "00000002-0001-0000-0000-000000000000"
				path = "/spaces/" + guid
			)

			It("should return space total", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, map[string]interface{}{
					"space_guid":             guid,
					"org_guid":               "00000002-0000-0000-0000-000000000000",
					"price_in_pence_ex_vat":  294700,
					"price_in_pence_inc_vat": 294700 * 1.2,
				})
			})

			It("should only return space totals for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"space_guid":             guid,
					"org_guid":               "00000002-0000-0000-0000-000000000000",
					"price_in_pence_ex_vat":  76800,
					"price_in_pence_inc_vat": 76800 * 1.2,
				})
			})

		})

		Context("/resources", func() {

			var (
				path = "/resources"
			)

			It("should return totals for each resource", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":                   "00000001-0001-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  716800,
						"price_in_pence_inc_vat": 716800 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0002-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0001-0001-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  38400,
						"price_in_pence_inc_vat": 38400 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0001-0002-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  256000,
						"price_in_pence_inc_vat": 256000 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0001-0003-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  100,
						"price_in_pence_inc_vat": 100 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0001-0004-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  200,
						"price_in_pence_inc_vat": 200 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0002-0001-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000002-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000003-0005-0001-0000-000000000000",
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (24*0.2 + 24*0.3) * 100,
						"price_in_pence_inc_vat": (24*0.2*1.2 + 24*0.3) * 100,
						"space_guid":             "00000003-0005-0000-0000-000000000000",
					},
				})
			})

			It("should only return resource total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":                   "00000001-0001-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  12800,
						"price_in_pence_inc_vat": 12800 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0001-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  409600,
						"price_in_pence_inc_vat": 409600 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000001-0002-0002-0000-000000000000",
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0001-0002-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  76800,
						"price_in_pence_inc_vat": 76800 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":                   "00000002-0002-0001-0000-000000000000",
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000002-0002-0000-0000-000000000000",
					},
					{
						"guid":                   "00000003-0005-0001-0000-000000000000",
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (1*0.2 + 1*0.3) * 100,
						"price_in_pence_inc_vat": (1*0.2*1.2 + 1*0.3) * 100,
						"space_guid":             "00000003-0005-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/resources/:guid", func() {

			var (
				guid = "00000002-0001-0002-0000-000000000000"
				path = "/resources/" + guid
			)

			It("should return totals for each resource", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, map[string]interface{}{
					"guid":                   guid,
					"org_guid":               "00000002-0000-0000-0000-000000000000",
					"price_in_pence_ex_vat":  256000,
					"price_in_pence_inc_vat": 256000 * 1.2,
					"space_guid":             "00000002-0001-0000-0000-000000000000",
				})
			})

			It("should only return resource total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"guid":                   guid,
					"org_guid":               "00000002-0000-0000-0000-000000000000",
					"price_in_pence_ex_vat":  76800,
					"price_in_pence_inc_vat": 76800 * 1.2,
					"space_guid":             "00000002-0001-0000-0000-000000000000",
				})
			})

		})

		Context("/resources/:resource_guid/events", func() {

			const (
				guid = "00000001-0001-0001-0000-000000000000"
				path = "/resources/" + guid + "/events"
			)

			It("should return event details for a resource", func() {
				var out interface{}
				from := monthsAgo(3)
				to := now
				computePlanBoundry := monthsAgo(1)
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"space_guid":        "00000001-0001-0000-0000-000000000000",
						"pricing_plan_id":   10,
						"pricing_plan_name": "ComputePlanA",
						"from":              from.Format(time.RFC3339),
						"to":                computePlanBoundry.Format(time.RFC3339),
						"price_in_pence_ex_vat":  int(computePlanBoundry.Sub(from).Hours()) * 64 * 1 * 100,
						"price_in_pence_inc_vat": float64(int(computePlanBoundry.Sub(from).Hours())*64*1*100) * 1.2,
						"guid":     guid,
						"org_guid": "00000001-0000-0000-0000-000000000000",
					},
					{
						"space_guid":        "00000001-0001-0000-0000-000000000000",
						"pricing_plan_id":   11,
						"pricing_plan_name": "ComputePlanB",
						"from":              computePlanBoundry.Format(time.RFC3339),
						"to":                to.Format(time.RFC3339),
						"price_in_pence_ex_vat":  int(to.Sub(computePlanBoundry).Hours()) * 64 * 2 * 100,
						"price_in_pence_inc_vat": float64(int(to.Sub(computePlanBoundry).Hours())*64*2*100) * 1.2,
						"guid":     guid,
						"org_guid": "00000001-0000-0000-0000-000000000000",
					},
				})
			})

			It("should only return events for the given range", func() {
				var out interface{}
				from := monthsAgo(3)
				to := monthsAgo(3).Add(1 * time.Hour)
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"space_guid": "00000001-0001-0000-0000-000000000000",
						"from":       from.Format(time.RFC3339),
						"to":         to.Format(time.RFC3339),
						"price_in_pence_ex_vat":  1 * 64 * 1 * 100,
						"price_in_pence_inc_vat": 1 * 64 * 1 * 100 * 1.2,
						"pricing_plan_id":        10,
						"pricing_plan_name":      "ComputePlanA",
						"guid":                   guid,
						"org_guid":               "00000001-0000-0000-0000-000000000000",
					},
				})
			})

			It("should return event details with different VAT rates", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				guid := "00000003-0005-0001-0000-000000000000"
				path := "/resources/" + guid + "/events"
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"space_guid":        "00000003-0005-0000-0000-000000000000",
						"pricing_plan_id":   40,
						"pricing_plan_name": "VATTest",
						"from":              to.Add(-24 * time.Hour).Format(time.RFC3339),
						"to":                to.Format(time.RFC3339),
						"price_in_pence_ex_vat":  (24*0.2 + 24*0.3) * 100,
						"price_in_pence_inc_vat": (24*0.2*1.2 + 24*0.3) * 100,
						"guid":     guid,
						"org_guid": "00000003-0000-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should fetch pricing plans", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"id":         20,
						"name":       "ServicePlanA",
						"plan_guid":  "00000000-0000-0000-0000-100000000000",
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         30,
						"name":       "ServicePlanB",
						"plan_guid":  "00000000-0000-0000-0000-200000000000",
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         40,
						"name":       "VATTest",
						"plan_guid":  "00000000-0000-0000-0000-300000000000",
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         10,
						"name":       "ComputePlanA",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         11,
						"name":       "ComputePlanB",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": monthsAgoRFC3339(1),
					},
				})
			})

		})

		Context("/pricing_plans/:pricing_plan_id", func() {

			var (
				id   = 30
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should fetch pricing plan by id", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, map[string]interface{}{
					"id":         id,
					"name":       "ServicePlanB",
					"plan_guid":  "00000000-0000-0000-0000-200000000000",
					"valid_from": monthsAgoRFC3339(3),
				})
			})
		})

		Context("/pricing_plans/:pricing_plan_id/components", func() {
			It("should fetch the pricing plan components by plan", itShouldFetchPricingPlanComponentsByPlan)
		})

		Context("POST /pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should be unauthorized", func() {
				form := url.Values{}
				form.Add("name", "NewPlan")
				form.Add("valid_from", agoRFC3339(1*time.Minute))
				form.Add("plan_guid", "aaaaaaa-bbbb-cccc-ddddddddddddd")
				status, _ := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/pricing_plans/:id", func() {

			var (
				id   = 11
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should update a pricing plan (via form PUT)", func() {
				form := url.Values{}
				form.Add("name", "UpdatedPlan")
				form.Add("valid_from", agoRFC3339(111*time.Hour))
				form.Add("plan_guid", db.ComputePlanGuid)
				status, _ := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/pricing_plans/:id", func() {

			var (
				id   = 11
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should delete a pricing plan (via form DELETE)", func() {
				form := url.Values{}
				status, _ := del(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/pricing_plan_components", func() {
			It("should fetch the pricing plan components", itShouldFetchPricingPlanComponents)
		})

		Context("/pricing_plan_components/:id", func() {

			It("should fetch pricing plan component by id", itShouldFetchPricingPlanComponentById)

			It("should return 404 for non-existing plan component", func() {
				path := "/pricing_plan_components/999"
				status, res := get(path)
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(res).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("POST /pricing_plan_components", func() {

			const (
				path = "/pricing_plan_components"
			)

			It("should only allow create for admins", func() {
				status, _ := post(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/pricing_plan_components/:id", func() {

			var path = "/pricing_plan_components/123"

			It("should only allow update for admins", func() {
				status, _ := put(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})

			It("should only allow delete for admins", func() {
				status, _ := del(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/vat_rates", func() {
			It("should fetch the VAT rates", itShouldFetchVATRates)
		})

		Context("/vat_rates/:id", func() {

			It("should fetch VAT rate by ID", itShouldFetchVATRateByID)

			It("should return 404 for non-existing VAT rate", func() {
				path := "/vat_rates/999"
				status, res := get(path)
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(res).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("POST /vat_rates", func() {

			const (
				path = "/vat_rates"
			)

			It("should only allow create for admins", func() {
				status, _ := post(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/vat_rates/:id", func() {

			var path = "/vat_rates/2"

			It("should only allow update for admins", func() {
				status, _ := put(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})

			It("should only allow delete for admins", func() {
				status, _ := del(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("/seed_pricing_plans", func() {
			It("should be unauthorized", func() {
				path := "/seed_pricing_plans"
				status, _ := post(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusUnauthorized))
			})
		})

	})

	Context("as admin", func() {

		BeforeEach(func() {
			authenticator = AuthenticatedAdmin
		})

		Context("/spaces", func() {

			var (
				path = "/spaces"
			)

			It("should let admin see ALL spaces", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  9216000,
						"price_in_pence_inc_vat": 9216000 * 1.2,
						"space_guid":             "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000001-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  716850,
						"price_in_pence_inc_vat": 716850 * 1.2,
						"space_guid":             "00000001-0002-0000-0000-000000000000",
					},

					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  294700,
						"price_in_pence_inc_vat": 294700 * 1.2,
						"space_guid":             "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  50,
						"price_in_pence_inc_vat": 50 * 1.2,
						"space_guid":             "00000002-0002-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000002-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  1850,
						"price_in_pence_inc_vat": 1850 * 1.2,
						"space_guid":             "00000002-0003-0000-0000-000000000000",
					},
					{
						"org_guid":               "00000003-0000-0000-0000-000000000000",
						"price_in_pence_ex_vat":  (24*0.2 + 24*0.3) * 100,
						"price_in_pence_inc_vat": (24*0.2*1.2 + 24*0.3) * 100,
						"space_guid":             "00000003-0005-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should fetch pricing plans", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"id":         20,
						"name":       "ServicePlanA",
						"plan_guid":  "00000000-0000-0000-0000-100000000000",
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         30,
						"name":       "ServicePlanB",
						"plan_guid":  "00000000-0000-0000-0000-200000000000",
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         40,
						"name":       "VATTest",
						"plan_guid":  "00000000-0000-0000-0000-300000000000",
						"valid_from": "2017-07-01T00:00:00Z",
					},
					{
						"id":         10,
						"name":       "ComputePlanA",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": monthsAgoRFC3339(3),
					},
					{
						"id":         11,
						"name":       "ComputePlanB",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": monthsAgoRFC3339(1),
					},
				})
			})

		})

		Context("/pricing_plans/:pricing_plan_id", func() {

			var (
				id   = 30
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should fetch pricing plan by id", func() {
				var out interface{}
				from := monthsAgo(1)
				to := now
				requestRanged(path, from, to, &out)
				ExpectJSON(out, map[string]interface{}{
					"id":         id,
					"name":       "ServicePlanB",
					"plan_guid":  "00000000-0000-0000-0000-200000000000",
					"valid_from": monthsAgoRFC3339(3),
				})
			})
		})

		Context("/pricing_plans/:pricing_plan_id/components", func() {
			It("should fetch the pricing plan components by plan", itShouldFetchPricingPlanComponentsByPlan)
		})

		Context("POST /pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should create a pricing plan (form POST)", func() {
				newValidFrom := monthsAgoRFC3339(4)
				form := url.Values{}
				form.Add("name", "NewPlan")
				form.Add("valid_from", newValidFrom)
				form.Add("plan_guid", "aaaaaaa-bbbb-cccc-ddddddddddddd")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":         1,
					"name":       "NewPlan",
					"valid_from": newValidFrom,
					"plan_guid":  "aaaaaaa-bbbb-cccc-ddddddddddddd",
				})
			})

			It("should not create a pricing plan that violates valid_from constraint (form POST)", func() {
				invalidFrom := "2017-04-04T00:00:00Z"
				form := url.Values{}
				form.Add("name", "NewPlan")
				form.Add("valid_from", invalidFrom)
				form.Add("plan_guid", "aaaaaaa-bbbb-cccc-ddddddddddddd")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusBadRequest))
				ExpectJSON(out, map[string]interface{}{
					"error":      "constraint violation",
					"constraint": "valid_from_start_of_month",
				})
			})
		})

		Context("/pricing_plans/:id", func() {

			var (
				id   = 11
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should update a pricing plan (via form PUT)", func() {
				newValidFrom := monthsAgoRFC3339(5)
				form := url.Values{}
				form.Add("name", "UpdatedPlan")
				form.Add("valid_from", newValidFrom)
				form.Add("plan_guid", db.ComputePlanGuid)
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":         11,
					"name":       "UpdatedPlan",
					"plan_guid":  db.ComputePlanGuid,
					"valid_from": newValidFrom,
				})
			})
		})

		Context("/pricing_plans/:id", func() {

			var (
				id   = 11
				path = "/pricing_plans/" + strconv.Itoa(id)
			)

			It("should delete a pricing plan (via form DELETE)", func() {
				form := url.Values{}
				status, out := del(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":         11,
					"name":       "ComputePlanB",
					"plan_guid":  db.ComputePlanGuid,
					"valid_from": monthsAgoRFC3339(1),
				})
			})
		})

		Context("/pricing_plan_components", func() {
			It("should fetch the pricing plan components", itShouldFetchPricingPlanComponents)
		})

		Context("/pricing_plan_components/:id", func() {
			It("should fetch pricing plan component by id", itShouldFetchPricingPlanComponentById)
		})

		Context("POST /pricing_plan_components", func() {

			const (
				path = "/pricing_plan_components"
			)

			It("should create a pricing plan component (form POST)", func() {
				form := url.Values{}
				form.Add("name", "NewPlanComp")
				form.Add("pricing_plan_id", "10")
				form.Add("formula", "$memory_in_mb * 1")
				form.Add("vat_rate_id", "2")
				form.Add("currency", "GBP")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"currency":        "GBP",
					"id":              1,
					"pricing_plan_id": 10,
					"name":            "NewPlanComp",
					"formula":         "$memory_in_mb * 1",
					"vat_rate_id":     2,
				})
			})

			It("should fail creaing a pricing plan component wit an invalid currency (form POST)", func() {
				form := url.Values{}
				form.Add("name", "NewPlanComp")
				form.Add("pricing_plan_id", "10")
				form.Add("formula", "$memory_in_mb * 1")
				form.Add("vat_rate_id", "2")
				form.Add("currency", "ISK")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusBadRequest))
				Expect(out).To(Equal(map[string]interface{}{
					"error":      "constraint violation",
					"constraint": "currency_valid",
				}))
			})
		})

		Context("/pricing_plan_components/:id", func() {

			var (
				id   = 101
				path = "/pricing_plan_components/" + strconv.Itoa(id)
			)

			It("should update a pricing plan component (via form PUT)", func() {
				form := url.Values{}
				form.Add("name", "UpdatedPlan")
				form.Add("pricing_plan_id", "20")
				form.Add("formula", "10*10")
				form.Add("vat_rate_id", "2")
				form.Add("currency", "GBP")
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"currency":        "GBP",
					"formula":         "10*10",
					"id":              101,
					"name":            "UpdatedPlan",
					"pricing_plan_id": 20,
					"vat_rate_id":     2,
				})
			})

			It("should return with 404 when trying to update non-existing component", func() {
				path = "/pricing_plan_components/999"
				form := url.Values{}
				form.Add("name", "UpdatedPlan")
				form.Add("pricing_plan_id", "20")
				form.Add("formula", "10*10")
				form.Add("vat_rate_id", "2")
				form.Add("currency", "GBP")
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(out).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("/pricing_plan_components/:id", func() {

			var (
				id   = 101
				path = "/pricing_plan_components/" + strconv.Itoa(id)
			)

			It("should delete a pricing plan component (via form DELETE)", func() {
				form := url.Values{}
				status, out := del(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"currency":        "GBP",
					"formula":         "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
					"id":              101,
					"name":            "ComputePlanA/1",
					"pricing_plan_id": 10,
					"vat_rate_id":     1,
				})

				status, _ = get(path)
				Expect(status).To(Equal(http.StatusNotFound))
			})

			It("should return with 404 when trying to delete non-existing component", func() {
				path = "/pricing_plan_components/999"
				status, out := del(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(out).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("/vat_rates", func() {
			It("should fetch the vat rates", itShouldFetchVATRates)
		})

		Context("/vat_rates/:id", func() {
			It("should fetch the vat rate by id", itShouldFetchVATRateByID)
		})

		Context("POST /vat_rates", func() {

			const (
				path = "/vat_rates"
			)

			It("should create a VAT rate (form POST)", func() {
				form := url.Values{}
				form.Add("name", "New rate")
				form.Add("rate", "0.25")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":   3,
					"name": "New rate",
					"rate": 0.25,
				})
			})
		})

		Context("/vat_rates/:id", func() {

			var (
				id   = 2
				path = "/vat_rates/" + strconv.Itoa(id)
			)

			It("should update a VAT rate (via form PUT)", func() {
				form := url.Values{}
				form.Add("name", "Updated rate")
				form.Add("rate", "0.25")
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":   2,
					"name": "Updated rate",
					"rate": 0.25,
				})
			})

			It("should return with 404 when trying to update non-existing VAT rate", func() {
				path = "/vat_rates/999"
				form := url.Values{}
				form.Add("name", "Updated rate")
				form.Add("rate", "0.25")
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(out).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("/vat_rates/:id", func() {

			var (
				id   = 3
				path = "/vat_rates/" + strconv.Itoa(id)
			)

			BeforeEach(func() {
				_, err := sqlClient.Exec("INSERT INTO vat_rates (name, rate) VALUES ('delete me', 0.5)")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should delete a VAT rate (via form DELETE)", func() {
				form := url.Values{}
				status, out := del(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":   3,
					"name": "delete me",
					"rate": 0.5,
				})

				status, _ = get(path)
				Expect(status).To(Equal(http.StatusNotFound))
			})

			It("should return with 404 when trying to delete non-existing VAT rate", func() {
				path = "/vat_rates/999"
				status, out := del(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusNotFound))
				Expect(out).To(Equal(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "not found",
					}},
				))
			})
		})

		Context("/seed_pricing_plans", func() {
			It("should be able to call and return with 200", func() {
				path := "/seed_pricing_plans"
				status, out := post(path, strings.NewReader(url.Values{}.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"success": true,
				})
			})
		})

	})

})
