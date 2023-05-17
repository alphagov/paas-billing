package cffetcher_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alphagov/paas-billing/eventfetchers/cffetcher/cffetcherfakes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-billing/eventfetchers/cffetcher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type clientFactory func(client UsageEventsClient, logger lager.Logger) UsageEventsAPI

var usageEventTests = func(eventType string, clientFactory clientFactory) func() {
	return func() {
		var (
			now                   time.Time
			logger                = lager.NewLogger("test")
			fakeUsageEventsClient *cffetcherfakes.FakeUsageEventsClient
			usageEvents           UsageEventsAPI
			emptyUsageList        string
			usageListWithRecords  string
		)

		BeforeEach(func() {
			now = time.Now()
			fakeUsageEventsClient = &cffetcherfakes.FakeUsageEventsClient{}
			usageEvents = clientFactory(fakeUsageEventsClient, logger)

			emptyUsageList = `{
				"total_results": 0,
				"total_pages": 1,
				"prev_url": null,
				"next_url": null,
				"resources": []
			}`

			usageListWithRecords = `{
				"total_results": 3,
				"total_pages": 1,
				"prev_url": null,
				"next_url": null,
				"resources": [
				{
					"metadata": {
						"guid": "a000",
						"url": "/v2/` + eventType + `_usage_events/a000",
						"created_at": "` + now.Add(-2*time.Minute).Format("2006-01-02T15:04:05Z07:00") + `"
					},
					"entity": {
						"field": "foo1"
					}
				},
				{
					"metadata": {
						"guid": "b000",
						"url": "/v2/` + eventType + `_usage_events/b000",
						"created_at": "` + now.Add(-1*time.Minute).Format("2006-01-02T15:04:05Z07:00") + `"
					},
					"entity": {
						"field": "foo2"
					}
				},
				{
					"metadata": {
						"guid": "c000",
						"url": "/v2/` + eventType + `_usage_events/c000",
						"created_at": "` + now.Format("2006-01-02T15:04:05Z07:00") + `"
					},
					"entity": {
						"field": "foo3"
					}
				}
				]
			}`
		})

		AfterEach(func() {
			// mockCtrl.Finish()
		})

		It("should have the right type", func() {
			client := clientFactory(fakeUsageEventsClient, logger)
			Expect(client.Type()).To(Equal(eventType))
		})

		It("should use the right API endpoint and build the URL when calling the API", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			usageEvents.Get(GUIDNil, 3, 0)
			url := fakeUsageEventsClient.GetArgsForCall(0)
			expectedURL := fmt.Sprintf("/v2/%s_usage_events?results-per-page=3", eventType)
			Expect(url).To(Equal(expectedURL))
		})

		It("should use after_guid parameter in URL when afterGUID arg is set", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			expectedURL := fmt.Sprintf("/v2/%s_usage_events?results-per-page=3&after_guid=abcd", eventType)
			usageEvents.Get("abcd", 3, 0)
			url := fakeUsageEventsClient.GetArgsForCall(0)
			Expect(url).To(Equal(expectedURL))
		})

		It("should return with an empty usage event list when API result is empty", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 3, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(events).To(Equal(&UsageEventList{Resources: []UsageEvent{}}), "result should be empty list")
		})

		It("should parse the result response correctly", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err).ToNot(HaveOccurred())

			expected := &UsageEventList{
				Resources: []UsageEvent{
					{
						MetaData: MetaData{
							GUID:      "a000",
							CreatedAt: now.Add(-2 * time.Minute).Truncate(time.Second),
						},
						EntityRaw: json.RawMessage([]byte("{\n        \"field\": \"foo1\"\n      }")),
					},
					{
						MetaData: MetaData{
							GUID:      "b000",
							CreatedAt: now.Add(-1 * time.Minute).Truncate(time.Second),
						},
						EntityRaw: json.RawMessage([]byte("{\n        \"field\": \"foo2\"\n      }")),
					},
					{
						MetaData: MetaData{
							GUID:      "c000",
							CreatedAt: now.Truncate(time.Second),
						},
						EntityRaw: json.RawMessage([]byte("{\n        \"field\": \"foo3\"\n      }")),
					},
				},
			}

			Expect(len(events.Resources)).To(Equal(len(expected.Resources)))
			for i, resource := range events.Resources {
				expectEventsToBeEqual(resource, expected.Resources[i])
			}
		})

		It("should not process records after the first item with newer than minimum age", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			expected := &UsageEventList{
				Resources: []UsageEvent{
					{
						MetaData: MetaData{
							GUID:      "a000",
							CreatedAt: now.Add(-2 * time.Minute).Truncate(time.Second),
						},
						EntityRaw: json.RawMessage([]byte("{\n        \"field\": \"foo1\"\n      }")),
					},
				},
			}

			Expect(len(events.Resources)).To(Equal(len(expected.Resources)))
			for i, resource := range events.Resources {
				expectEventsToBeEqual(resource, expected.Resources[i])
			}
		})

		It("should return an empty usage event list when all records are newer than minimum age", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(*events).To(Equal(UsageEventList{Resources: []UsageEvent{}}))
		})

		It("should handle client error when API request fails", func() {
			getErr := errors.New("some error")
			fakeUsageEventsClient.GetReturns(nil, getErr)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("error fetching /v2/%s_usage_events?results-per-page=10: some error", eventType)))
			Expect(events).To(BeNil())
		})

		It("should return an error for non-200 response codes", func() {
			resp := &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(strings.NewReader("some error")),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("/v2/%s_usage_events?results-per-page=10 request failed: 500 some error", eventType)))
			Expect(events).To(BeNil())
		})

		It("should return an error when response contains invalid JSON", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("non json")),
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err).To(HaveOccurred())
			Expect(events).To(BeNil())
		})

		It("should return with an error if reading body returns an IO error", func() {
			r, w := io.Pipe()
			w.CloseWithError(errors.New("io-error"))
			resp := &http.Response{
				StatusCode: 200,
				Body:       r,
			}
			fakeUsageEventsClient.GetReturns(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err).To(HaveOccurred())
			Expect(events).To(BeNil())
		})

		It("should panic when afterGUID is an empty string", func() {
			Expect(func() { usageEvents.Get("", 10, 0) }).To(Panic())
		})

	}
}

var _ = Describe("The App Usage Events Handler", usageEventTests("app", func(client UsageEventsClient, logger lager.Logger) UsageEventsAPI {
	return NewAppUsageEventsAPI(client, logger)
}))

var _ = Describe("The Service Usage Events Handler", usageEventTests("service", func(client UsageEventsClient, logger lager.Logger) UsageEventsAPI {
	return NewServiceUsageEventsAPI(client, logger)
}))

// On Linux comparing two events fails while on OS X it succeeds
// Somehow the two OSes do the time.Time equality differently
func expectEventsToBeEqual(e1 UsageEvent, e2 UsageEvent) {
	Expect(e1.MetaData.GUID).To(Equal(e2.MetaData.GUID))
	Expect(e1.MetaData.CreatedAt).To(BeTemporally("==", e2.MetaData.CreatedAt))
	Expect(string(e1.EntityRaw)).To(MatchJSON(string(e2.EntityRaw)))
}
