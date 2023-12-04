package acceptance_tests_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth-related acceptance tests", Ordered, func() {
	type eventCommon struct {
		OrgGUID   string `json:"org_guid"`
		EventGUID string `json:"event_guid"`
	}

	eventCommonTests := func(endpointPath string) {
		var (
			orgGUIDWithEvents string
			httpClient        = &http.Client{}
			rangeStart        = time.Now().AddDate(0, 0, -2).Format("2006-01-02")
			rangeStop         = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		)

		BeforeAll(func() {
			By("Selecting an arbitrary org id with events", func() {
				Eventually(func() bool {
					u, err := url.Parse(BillingAPIURLFromEnv)
					Expect(err).ToNot(HaveOccurred())
					u = u.JoinPath(endpointPath)
					q := u.Query()
					q.Set("range_start", rangeStart)
					q.Set("range_stop", rangeStop)
					u.RawQuery = q.Encode()

					req, err := http.NewRequest("GET", u.String(), nil)
					Expect(err).ToNot(HaveOccurred())
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFAdminBearerTokenFromEnv))
					res, err := httpClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					data, err := ioutil.ReadAll(res.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.StatusCode).To(Equal(200))

					var events []eventCommon
					err = json.Unmarshal(data, &events)
					Expect(err).ToNot(HaveOccurred())

					if len(events) < 1 {
						return false
					}

					// picking the same org as CFNonAdminBillingManagerOrgGUID
					// would screw up the non-admin tests
					orgGUIDWithEvents = ""
					for _, event := range events {
						if CFNonAdminBillingManagerOrgGUID == "" || event.OrgGUID != CFNonAdminBillingManagerOrgGUID {
							orgGUIDWithEvents = event.OrgGUID
							break
						}
					}
					if orgGUIDWithEvents == "" {
						return false
					}
					return true
				}, 4*time.Minute, 20*time.Second).Should(Equal(true), "Gave up waiting for events to be collected")
			})
		})

		Context("Without an auth token", func() {
			It("Returns 401 for single-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", orgGUIDWithEvents)
				q.Set("range_start", "2001-01-01")
				q.Set("range_stop", "2001-01-02")
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(`{
					"error": "no access_token in request"
				}`))
			})

			It("Returns 401 for any-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("range_start", "2001-01-01")
				q.Set("range_stop", "2001-01-02")
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(`{
					"error": "no access_token in request"
				}`))
			})
		})

		Context("With an admin auth token", func() {
			It("Returns events for single-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", orgGUIDWithEvents)
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				data, err := ioutil.ReadAll(res.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(200))

				var events []eventCommon
				err = json.Unmarshal(data, &events)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(events)).To(BeNumerically(">", 0))
				for _, ev := range events {
					Expect(ev.OrgGUID).To(Equal(orgGUIDWithEvents))
					Expect(ev.EventGUID).ToNot(BeEmpty())
				}
			})

			It("Returns events for multi-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", orgGUIDWithEvents)
				q.Add("org_guid", CFNonAdminBillingManagerOrgGUID)
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				data, err := ioutil.ReadAll(res.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(200))

				var events []eventCommon
				err = json.Unmarshal(data, &events)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(events)).To(BeNumerically(">", 2))
				foundOrgGUIDWithEvents := false
				foundCFNonAdminBillingManagerOrgGUID := false
				for _, ev := range events {
					Expect(ev.EventGUID).ToNot(BeEmpty())
					switch ev.OrgGUID {
					case orgGUIDWithEvents:
						foundOrgGUIDWithEvents = true
					case CFNonAdminBillingManagerOrgGUID:
						foundCFNonAdminBillingManagerOrgGUID = true
					default:
						Expect(false).To(BeTrue())
					}
				}
				Expect(foundOrgGUIDWithEvents).To(BeTrue())
				Expect(foundCFNonAdminBillingManagerOrgGUID).To(BeTrue())
			})

			It("Returns events for any-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				data, err := ioutil.ReadAll(res.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(200))

				var events []eventCommon
				err = json.Unmarshal(data, &events)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(events)).To(BeNumerically(">", 0))
			})
		})

		Context("With a non-admin auth token", func() {
			It("Returns events for single-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", CFNonAdminBillingManagerOrgGUID)
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFNonAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				data, err := ioutil.ReadAll(res.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(200))

				var events []eventCommon
				err = json.Unmarshal(data, &events)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(events)).To(BeNumerically(">", 0))
				for _, ev := range events {
					Expect(ev.OrgGUID).To(Equal(CFNonAdminBillingManagerOrgGUID))
					Expect(ev.EventGUID).ToNot(BeEmpty())
				}
			})

			It("Returns 401 for single-org requests for an unrelated org", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", orgGUIDWithEvents)
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFNonAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(fmt.Sprintf(`{
					"error": "invalid credentials: authorizer: no access to organisation: %s"
				}`, orgGUIDWithEvents)))
			})

			It("Returns 401 for multi-org requests covering an unrelated org", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("org_guid", orgGUIDWithEvents)
				q.Add("org_guid", CFNonAdminBillingManagerOrgGUID)
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFNonAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(fmt.Sprintf(`{
					"error": "invalid credentials: authorizer: no access to organisation: %s"
				}`, orgGUIDWithEvents)))
			})

			It("Returns 401 for any-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				q := u.Query()
				q.Set("range_start", rangeStart)
				q.Set("range_stop", rangeStop)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFNonAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(`{
					"error": "invalid credentials: only admins are allowed to access billing for all organisations"
				}`))
			})

			It("Returns 401 for sneaky any-org requests", func() {
				u, err := url.Parse(BillingAPIURLFromEnv)
				Expect(err).ToNot(HaveOccurred())
				u = u.JoinPath(endpointPath)
				u.RawQuery = fmt.Sprintf(
					"range_start=%s&range_stop=%s&org_guid=%%",
					rangeStart,
					rangeStop,
				)

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", CFNonAdminBearerTokenFromEnv))
				res, err := httpClient.Do(req)

				Expect(res.StatusCode).To(Equal(401))

				data, err := ioutil.ReadAll(res.Body)
				Expect(data).To(MatchJSON(`{
					"error": "invalid credentials: only admins are allowed to access billing for all organisations"
				}`))
			})
		})
	}

	Context("/usage_events", func() {
		eventCommonTests("usage_events")
	})

	Context("/billable_events", func() {
		eventCommonTests("billable_events")
	})
})
