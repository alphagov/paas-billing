package cloudfoundry_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type clientFactory func(client Client, logger lager.Logger) UsageEventsAPI

var usageEventTests = func(eventType string, clientFactory clientFactory) func() {
	return func() {
		var (
			now                  time.Time
			mockCtrl             *gomock.Controller
			logger               lager.Logger
			mockClient           *mocks.MockClient
			usageEvents          UsageEventsAPI
			emptyUsageList       string
			usageListWithRecords string
		)

		BeforeEach(func() {
			now = time.Now()
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mocks.NewMockClient(mockCtrl)
			logger = lager.NewLogger("test")
			usageEvents = clientFactory(mockClient, logger)

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
			mockCtrl.Finish()
		})

		Context("When the client is created", func() {
			It("should have the right type", func() {
				client := clientFactory(mockClient, logger)
				Expect(client.Type()).To(Equal(eventType))
			})
		})

		Context("When the API is called", func() {
			It("should use the right API endpoint and build the url", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get(fmt.Sprintf("/v2/%s_usage_events?results-per-page=3", eventType)).Return(resp, nil)
				usageEvents.Get(GUIDNil, 3, 0)
			})
		})

		Context("When the afterGUID parameter is set", func() {
			It("the API url should contain the after_guid parameter", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get(fmt.Sprintf("/v2/%s_usage_events?results-per-page=3&after_guid=abcd", eventType)).Return(resp, nil)
				usageEvents.Get("abcd", 3, 0)
			})
		})

		Context("When the API result is empty", func() {
			It("should return with an empty usage event list", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 3, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(events).To(Equal(&UsageEventList{Resources: []UsageEvent{}}), "result should be empty list")
			})
		})

		Context("When the API returns with records", func() {
			It("should parse the result list correctly", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
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
		})

		Context("When the result list contains results newer than minimum age", func() {
			It("should not process records after the first item with newer than minimum age", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
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
		})

		Context("When all records are newer than minimum age", func() {
			It("should return an empty usage event list", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 10*time.Minute)
				Expect(err).ToNot(HaveOccurred())
				Expect(*events).To(Equal(UsageEventList{Resources: []UsageEvent{}}))
			})
		})

		Context("When the API request fails", func() {
			It("should handle client error", func() {
				mockClient.EXPECT().Get(gomock.Any()).Return(nil, errors.New("some error"))
				events, err := usageEvents.Get(GUIDNil, 10, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("error fetching /v2/%s_usage_events?results-per-page=10: some error", eventType)))
				Expect(events).To(BeNil())
			})
		})

		Context("When API returns with a non-200 http status code", func() {
			It("should handle it as an error", func() {
				resp := &http.Response{
					StatusCode: 500,
					Body:       ioutil.NopCloser(strings.NewReader("some error")),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("/v2/%s_usage_events?results-per-page=10 request failed: 500 some error", eventType)))
				Expect(events).To(BeNil())
			})
		})

		Context("When the API returns with a non-valid JSON", func() {
			It("should return with an error", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("non json")),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 0)
				Expect(err).To(HaveOccurred())
				Expect(events).To(BeNil())
			})
		})

		Context("When reading the response we encounter an IO error", func() {
			It("should return with an error", func() {
				bodyMock := mocks.NewMockReadCloser(mockCtrl)
				bodyMock.EXPECT().Read(gomock.Any()).Return(0, errors.New("some error"))
				bodyMock.EXPECT().Close()
				resp := &http.Response{
					StatusCode: 200,
					Body:       bodyMock,
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 0)
				Expect(err).To(HaveOccurred())
				Expect(events).To(BeNil())
			})
		})

		Context("When fetching the usage events afterGUID is an empty string", func() {
			It("should panic", func() {
				Expect(func() { usageEvents.Get("", 10, 0) }).To(Panic())
			})
		})

	}
}

var _ = Describe("The App Usage Events Handler", usageEventTests("app", func(client Client, logger lager.Logger) UsageEventsAPI {
	return NewAppUsageEventsAPI(client, logger)
}))

var _ = Describe("The Service Usage Events Handler", usageEventTests("service", func(client Client, logger lager.Logger) UsageEventsAPI {
	return NewServiceUsageEventsAPI(client, logger)
}))

// On Linux comparing two events fails while on OS X it succeeds
// Somehow the two OSes do the time.Time equality differently
func expectEventsToBeEqual(e1 UsageEvent, e2 UsageEvent) {
	Expect(e1.MetaData.GUID).To(Equal(e2.MetaData.GUID))
	Expect(e1.MetaData.CreatedAt).To(BeTemporally("==", e2.MetaData.CreatedAt))
	Expect(e1.EntityRaw).To(Equal(e2.EntityRaw))
}
