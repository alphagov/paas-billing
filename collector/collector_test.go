package collector_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-usage-events-collector/cloudfoundry/mocks"
	. "github.com/alphagov/paas-usage-events-collector/collector"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type customTestReporter struct {
	t           gomock.TestReporter
	CatchErrors bool
	Error       error
}

func (c *customTestReporter) Errorf(format string, args ...interface{}) {
	if !c.CatchErrors {
		c.t.Errorf(format, args...)
	} else {
		c.Error = fmt.Errorf(format, args)
	}
}

func (c *customTestReporter) Fatalf(format string, args ...interface{}) {
	if !c.CatchErrors {
		c.t.Fatalf(format, args...)
	} else {
		c.Error = fmt.Errorf(format, args)
	}
}

var _ = Describe("Collector", func() {
	var (
		testReporter   *customTestReporter
		mockCtrl       *gomock.Controller
		logger         lager.Logger
		mockClient     *mocks.MockClient
		emptyUsageList string
	)

	BeforeEach(func() {
		testReporter = &customTestReporter{t: GinkgoT()}
		mockCtrl = gomock.NewController(testReporter)
		mockClient = mocks.NewMockClient(mockCtrl)
		logger = lager.NewLogger("test")
		emptyUsageList = `{
  "total_results": 0,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": []
}`
	})

	AfterEach(func() {
		testReporter.CatchErrors = false
		mockCtrl.Finish()
	})

	Context("When the collector gets an interrupt signal", func() {
		It("should stop gracefully", func(done Done) {
			defer close(done)

			collector, err := New(mockClient, Config{}, logger)
			Expect(err).To(BeNil())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				collector.Run(ctx)
				c <- true
			}()

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 5)
	})

	Context("When the collector is started", func() {

	})

	Context("When the collector is started", func() {

		It("should collect app and service usage metrics regularly", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "10",
				RecordMinAge:    "1s",
			}
			collector, err := New(mockClient, config, logger)
			Expect(err).To(BeNil())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			resp := func() *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
			}
			mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=10").Return(resp(), nil)
			mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=10").Return(resp(), nil)
			mockClient.EXPECT().Get("/v2/service_usage_events?results-per-page=10").Return(resp(), nil)
			mockClient.EXPECT().Get("/v2/service_usage_events?results-per-page=10").Return(resp(), nil)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 1).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)

		It("should ignore errors when API call fails", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "10",
				RecordMinAge:    "1s",
			}
			collector, err := New(mockClient, config, logger)
			Expect(err).To(BeNil())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			resp := func() *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(emptyUsageList)),
				}
			}
			mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=10").Return(nil, errors.New("some error"))
			mockClient.EXPECT().Get("/v2/app_usage_events?results-per-page=10").Return(nil, errors.New("some error"))
			mockClient.EXPECT().Get("/v2/service_usage_events?results-per-page=10").Return(resp(), nil)
			mockClient.EXPECT().Get("/v2/service_usage_events?results-per-page=10").Return(resp(), nil)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 1).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)
	})

	Context("When the collector is wrongly configured", func() {

		It("should error if DefaultSchedule is not a valid time duration", func() {
			config := Config{
				DefaultSchedule: "xxx",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("DefaultSchedule is invalid"))
		})

		It("should error if MinWaitTime is not a valid time duration", func() {
			config := Config{
				MinWaitTime: "xxx",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("MinWaitTime is invalid"))
		})

		It("should error if RecordMinAge is not a valid time duration", func() {
			config := Config{
				RecordMinAge: "xxx",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("RecordMinAge is invalid"))
		})

		It("should error if FetchLimit is not a valid time duration", func() {
			config := Config{
				FetchLimit: "xxx",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("FetchLimit is invalid"))
		})

		It("should error if FetchLimit is zero", func() {
			config := Config{
				FetchLimit: "0",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("FetchLimit must be a positive integer"))
		})

		It("should error if FetchLimit is negative", func() {
			config := Config{
				FetchLimit: "-1",
			}
			_, err := New(mockClient, config, logger)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("FetchLimit must be a positive integer"))
		})

	})

})
