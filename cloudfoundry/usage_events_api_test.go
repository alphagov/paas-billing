package cloudfoundry_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-usage-events-collector/cloudfoundry/mocks"
	"github.com/golang/mock/gomock"

	. "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("The Usage Events Handler", func() {
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
		usageEvents = NewAppUsageEventsAPI(mockClient, logger)

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
        "url": "/v2/app_usage_events/a000",
        "created_at": "` + now.Add(-2*time.Minute).Format("2006-01-02T15:04:05Z07:00") + `"
      },
      "entity": {
        "field": "foo1"
      }
    },
    {
      "metadata": {
        "guid": "b000",
        "url": "/v2/app_usage_events/b000",
        "created_at": "` + now.Add(-1*time.Minute).Format("2006-01-02T15:04:05Z07:00") + `"
      },
      "entity": {
        "field": "foo2"
      }
    },
    {
      "metadata": {
        "guid": "c000",
        "url": "/v2/app_usage_events/c000",
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

	Describe("Usage events API client", func() {

		Context("When created for the apps", func() {
			It("should use the app API point", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=3").Return(resp, nil)
				client := NewAppUsageEventsAPI(mockClient, logger)
				Expect(client.Type()).To(BeEquivalentTo("app"))
				client.Get(GUIDNil, 3, 0)
			})
		})

		Context("When created for the service", func() {
			It("should use the service API point", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get("/v2/service_usage_events?results-per-page=3").Return(resp, nil)
				client := NewServiceUsageEventsAPI(mockClient, logger)
				Expect(client.Type()).To(BeEquivalentTo("service"))
				client.Get(GUIDNil, 3, 0)
			})
		})

		Context("When the API is called", func() {

			It("should set the results-per-page parameter", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=3").Return(resp, nil)
				usageEvents.Get(GUIDNil, 3, 0)
			})

			It("should set the after_guid parameter if set", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=3&after_guid=abcd").Return(resp, nil)
				usageEvents.Get("abcd", 3, 0)
			})

			It("should handle empty results", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 3, 0)
				Expect(err).To(BeNil(), "there should be no error")
				Expect(events).To(BeEquivalentTo(&UsageEventList{Resources: []UsageEvent{}}), "result should be empty list")
			})

			It("should parse result list correctly", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 0)
				Expect(err).To(BeNil(), "there should be no error")

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

				Expect(events).To(BeEquivalentTo(expected))
			})

			It("should return results only with minimum age", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 2*time.Minute)
				Expect(err).To(BeNil(), "there should be no error")

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

				Expect(events).To(BeEquivalentTo(expected))
			})

			It("should return empty list if every record is newer than minimum age", func() {
				resp := &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(usageListWithRecords)),
				}
				mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
				events, err := usageEvents.Get(GUIDNil, 10, 10*time.Minute)
				Expect(err).To(BeNil(), "there should be no error")
				Expect(events).To(BeEquivalentTo(&UsageEventList{Resources: []UsageEvent{}}))
			})

		})

	})

	Context("When the API request fails", func() {

		It("should handle client error", func() {
			mockClient.EXPECT().Get(gomock.Any()).Return(nil, errors.New("some error"))
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err.Error()).ToNot(BeNil())
			Expect(err.Error()).To(BeEquivalentTo("error fetching /v2/app_usage_events?results-per-page=10: some error"))
			Expect(events).To(BeNil())
		})

		It("should handle non-200 http status code", func() {
			resp := &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(strings.NewReader("some error")),
			}
			mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err.Error()).ToNot(BeNil())
			Expect(err.Error()).To(BeEquivalentTo("/v2/app_usage_events?results-per-page=10 request failed: 500 some error"))
			Expect(events).To(BeNil())
		})

		It("should handle non valid JSON response", func() {
			resp := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("non json")),
			}
			mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err.Error()).ToNot(BeNil())
			Expect(events).To(BeNil())
		})

		It("should handle IO error", func() {
			bodyMock := mocks.NewMockReadCloser(mockCtrl)
			bodyMock.EXPECT().Read(gomock.Any()).Return(0, errors.New("some error"))
			bodyMock.EXPECT().Close()
			resp := &http.Response{
				StatusCode: 200,
				Body:       bodyMock,
			}
			mockClient.EXPECT().Get(gomock.Any()).Return(resp, nil)
			events, err := usageEvents.Get(GUIDNil, 10, 0)
			Expect(err.Error()).ToNot(BeNil())
			Expect(events).To(BeNil())
		})

	})

	Context("When the method is not used correctly", func() {

		It("should panic if GUID is empty", func() {
			Expect(func() { usageEvents.Get("", 10, 0) }).To(Panic())
		})

	})

})
