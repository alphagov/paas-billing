package api_test

import (
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

	var request = func(path string, v interface{}) {
		doRequest(path, v, map[string]string{
			"from": ago(100 * time.Hour).Format(time.RFC3339Nano),
			"to":   now.Format(time.RFC3339Nano),
		})
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 633650,
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 185950,
					},
				})
			})

			It("should only return org total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 422450,
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 76850,
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
				request(path, &out)
				ExpectJSON(out, map[string]interface{}{
					"org_guid":       guid,
					"price_in_pence": 633650,
				})
			})

			It("should only return totals for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"org_guid":       guid,
					"price_in_pence": 422450,
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 563250,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":           "00000001-0001-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 563200,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0002-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 563250,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 185900,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000002-0002-0000-0000-000000000000",
					},
				})
			})

			It("should only return space total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 12800,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 409650,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 76800,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000002-0002-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":           "00000001-0001-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, map[string]interface{}{
					"space_guid":     guid,
					"org_guid":       "00000002-0000-0000-0000-000000000000",
					"price_in_pence": 185900,
				})
			})

			It("should only return space totals for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"space_guid":     guid,
					"org_guid":       "00000002-0000-0000-0000-000000000000",
					"price_in_pence": 76800,
				})
			})

		})

		Context("/resources", func() {

			var (
				path = "/resources"
			)

			It("should return totals for each resource", func() {
				var out interface{}
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":           "00000001-0001-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 563200,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0002-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0001-0001-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 19200,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0001-0002-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 166400,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0001-0003-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 100,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0001-0004-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 200,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0002-0001-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000002-0002-0000-0000-000000000000",
					},
				})
			})

			It("should only return resource total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"guid":           "00000001-0001-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 12800,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0001-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 409600,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":           "00000001-0002-0002-0000-000000000000",
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0001-0002-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 76800,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"guid":           "00000002-0002-0001-0000-000000000000",
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000002-0002-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, map[string]interface{}{
					"guid":           guid,
					"org_guid":       "00000002-0000-0000-0000-000000000000",
					"price_in_pence": 166400,
					"space_guid":     "00000002-0001-0000-0000-000000000000",
				})
			})

			It("should only return resource total for the given range", func() {
				var out interface{}
				requestRanged(path, ago(1*time.Hour), now, &out)
				ExpectJSON(out, map[string]interface{}{
					"guid":           guid,
					"org_guid":       "00000002-0000-0000-0000-000000000000",
					"price_in_pence": 76800,
					"space_guid":     "00000002-0001-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"space_guid":        "00000001-0001-0000-0000-000000000000",
						"pricing_plan_id":   10,
						"pricing_plan_name": "ComputePlanA",
						"from":              agoRFC3339(10 * time.Hour),
						"to":                agoRFC3339(1 * time.Hour),
						"price_in_pence":    9 * 64 * 1 * 100,
						"guid":              guid,
						"org_guid":          "00000001-0000-0000-0000-000000000000",
					},
					{
						"space_guid":        "00000001-0001-0000-0000-000000000000",
						"pricing_plan_id":   11,
						"pricing_plan_name": "ComputePlanB",
						"from":              agoRFC3339(1 * time.Hour),
						"to":                nowRFC3339,
						"price_in_pence":    1 * 64 * 2 * 100,
						"guid":              guid,
						"org_guid":          "00000001-0000-0000-0000-000000000000",
					},
				})
			})

			It("should only return events for the given range", func() {
				var out interface{}
				requestRanged(path, ago(10*time.Hour), ago(9*time.Hour), &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"space_guid":        "00000001-0001-0000-0000-000000000000",
						"from":              agoRFC3339(10 * time.Hour),
						"to":                agoRFC3339(9 * time.Hour),
						"price_in_pence":    1 * 64 * 1 * 100,
						"pricing_plan_id":   10,
						"pricing_plan_name": "ComputePlanA",
						"guid":              guid,
						"org_guid":          "00000001-0000-0000-0000-000000000000",
					},
				})
			})

		})

		Context("/events", func() {

			const (
				path = "/events"
			)

			It("should return events all events", func() {
				var out interface{}
				request(path, &out)
				// TODO: this is hard to test :)
			})

		})

		Context("/pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should fetch pricing plans", func() {
				var out interface{}
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"formula":    "($time_in_seconds / 60 / 60) * 0.5",
						"id":         20,
						"name":       "ServicePlanA",
						"plan_guid":  "00000000-0000-0000-0000-100000000000",
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * 1",
						"id":         30,
						"name":       "ServicePlanB",
						"plan_guid":  "00000000-0000-0000-0000-200000000000",
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * $memory_in_mb * 1",
						"id":         10,
						"name":       "ComputePlanA",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
						"id":         11,
						"name":       "ComputePlanB",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": agoRFC3339(1 * time.Hour),
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
				request(path, &out)
				ExpectJSON(out, map[string]interface{}{
					"formula":    "($time_in_seconds / 60 / 60) * 1",
					"id":         id,
					"name":       "ServicePlanB",
					"plan_guid":  "00000000-0000-0000-0000-200000000000",
					"valid_from": agoRFC3339(100 * time.Hour),
				})
			})
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
				form.Add("formula", "$memory_in_mb * 1")
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
				form.Add("formula", "10*10")
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 70400,
						"space_guid":     "00000001-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000001-0000-0000-0000-000000000000",
						"price_in_pence": 563250,
						"space_guid":     "00000001-0002-0000-0000-000000000000",
					},

					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 185900,
						"space_guid":     "00000002-0001-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 50,
						"space_guid":     "00000002-0002-0000-0000-000000000000",
					},
					{
						"org_guid":       "00000002-0000-0000-0000-000000000000",
						"price_in_pence": 1050,
						"space_guid":     "00000002-0003-0000-0000-000000000000",
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
				request(path, &out)
				ExpectJSON(out, []map[string]interface{}{
					{
						"formula":    "($time_in_seconds / 60 / 60) * 0.5",
						"id":         20,
						"name":       "ServicePlanA",
						"plan_guid":  "00000000-0000-0000-0000-100000000000",
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * 1",
						"id":         30,
						"name":       "ServicePlanB",
						"plan_guid":  "00000000-0000-0000-0000-200000000000",
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * $memory_in_mb * 1",
						"id":         10,
						"name":       "ComputePlanA",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": agoRFC3339(100 * time.Hour),
					},
					{
						"formula":    "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
						"id":         11,
						"name":       "ComputePlanB",
						"plan_guid":  db.ComputePlanGuid,
						"valid_from": agoRFC3339(1 * time.Hour),
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
				request(path, &out)
				ExpectJSON(out, map[string]interface{}{
					"formula":    "($time_in_seconds / 60 / 60) * 1",
					"id":         id,
					"name":       "ServicePlanB",
					"plan_guid":  "00000000-0000-0000-0000-200000000000",
					"valid_from": agoRFC3339(100 * time.Hour),
				})
			})
		})

		Context("POST /pricing_plans", func() {

			const (
				path = "/pricing_plans"
			)

			It("should create a pricing plan (form POST)", func() {
				form := url.Values{}
				form.Add("name", "NewPlan")
				form.Add("valid_from", agoRFC3339(1*time.Minute))
				form.Add("plan_guid", "aaaaaaa-bbbb-cccc-ddddddddddddd")
				form.Add("formula", "$memory_in_mb * 1")
				status, out := post(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"id":         1,
					"name":       "NewPlan",
					"valid_from": agoRFC3339(1 * time.Minute),
					"plan_guid":  "aaaaaaa-bbbb-cccc-ddddddddddddd",
					"formula":    "$memory_in_mb * 1",
				})
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
				form.Add("formula", "10*10")
				status, out := put(path, strings.NewReader(form.Encode()))
				Expect(status).To(Equal(http.StatusOK))
				ExpectJSON(out, map[string]interface{}{
					"formula":    "10*10",
					"id":         11,
					"name":       "UpdatedPlan",
					"plan_guid":  db.ComputePlanGuid,
					"valid_from": agoRFC3339(111 * time.Hour),
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
					"formula":    "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
					"id":         11,
					"name":       "ComputePlanB",
					"plan_guid":  db.ComputePlanGuid,
					"valid_from": agoRFC3339(1 * time.Hour),
				})
			})
		})

	})

})
