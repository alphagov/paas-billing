package collector_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	. "github.com/alphagov/paas-usage-events-collector/collector"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/alphagov/paas-usage-events-collector/mocks"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	event1GUID = "2C5D3E72-2082-43C1-9262-814EAE7E65AA"
	event2GUID = "968437F2-CCEE-4B8E-B29B-34EA701BA196"
	event3GUID = "0C586A5D-BFB7-4B31-91A2-7A7D06962D50"
	event4GUID = "F2133349-E9D5-47B6-AA0C-00A5AC96B703"
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
		testReporter      *customTestReporter
		mockCtrl          *gomock.Controller
		logger            lager.Logger
		mockAppClient     *mocks.MockUsageEventsAPI
		mockServiceClient *mocks.MockUsageEventsAPI
		mockSQLClient     *mocks.MockSQLClient
		emptyUsageEvents  *cloudfoundry.UsageEventList
		someAppEvents     *cloudfoundry.UsageEventList
		someServiceEvents *cloudfoundry.UsageEventList
	)

	BeforeEach(func() {
		testReporter = &customTestReporter{t: GinkgoT()}
		mockCtrl = gomock.NewController(testReporter)
		mockAppClient = mocks.NewMockUsageEventsAPI(mockCtrl)
		mockAppClient.EXPECT().Type().AnyTimes().Return("app")
		mockServiceClient = mocks.NewMockUsageEventsAPI(mockCtrl)
		mockServiceClient.EXPECT().Type().AnyTimes().Return("service")
		mockSQLClient = mocks.NewMockSQLClient(mockCtrl)
		logger = lager.NewLogger("test")
		emptyUsageEvents = &cloudfoundry.UsageEventList{[]cloudfoundry.UsageEvent{}}

		someAppEvents = &cloudfoundry.UsageEventList{
			Resources: []cloudfoundry.UsageEvent{
				cloudfoundry.UsageEvent{
					MetaData:  cloudfoundry.MetaData{GUID: event1GUID},
					EntityRaw: json.RawMessage(`{"field":"value1"}`),
				},
				cloudfoundry.UsageEvent{
					MetaData:  cloudfoundry.MetaData{GUID: event2GUID},
					EntityRaw: json.RawMessage(`{"field":"value2"}`),
				},
			},
		}
		someServiceEvents = &cloudfoundry.UsageEventList{
			Resources: []cloudfoundry.UsageEvent{
				cloudfoundry.UsageEvent{
					MetaData:  cloudfoundry.MetaData{GUID: event3GUID},
					EntityRaw: json.RawMessage(`{"field":"value3"}`),
				},
				cloudfoundry.UsageEvent{
					MetaData:  cloudfoundry.MetaData{GUID: event4GUID},
					EntityRaw: json.RawMessage(`{"field":"value4"}`),
				},
			},
		}
	})

	AfterEach(func() {
		testReporter.CatchErrors = false
		mockCtrl.Finish()
	})

	Context("When the collector gets an interrupt signal", func() {
		It("should stop gracefully", func(done Done) {
			defer close(done)

			collector, err := New(mockAppClient, mockServiceClient, mockSQLClient, &Config{}, logger)
			Expect(err).ToNot(HaveOccurred())

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

	Context("When the collector is running", func() {
		It("should collect app and service usage metrics regularly", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := &Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "10",
				RecordMinAge:    "1m",
			}
			collector, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).Return(cloudfoundry.GUIDNil, nil)
			mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).Return(event2GUID, nil)
			mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).Return(cloudfoundry.GUIDNil, nil)
			mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).Return(event4GUID, nil)

			mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(someAppEvents, nil)
			mockAppClient.EXPECT().Get(event2GUID, 10, time.Minute).Return(emptyUsageEvents, nil)

			mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(someServiceEvents, nil)
			mockServiceClient.EXPECT().Get(event4GUID, 10, time.Minute).Return(emptyUsageEvents, nil)

			mockSQLClient.EXPECT().InsertUsageEventList(someAppEvents, db.AppUsageTableName).Return(nil)
			mockSQLClient.EXPECT().InsertUsageEventList(someServiceEvents, db.ServiceUsageTableName).Return(nil)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 2).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)
	})

	Context("When the API call fails", func() {
		It("should handle the error and retry in the next iteration", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := &Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "10",
				RecordMinAge:    "1m",
			}
			collector, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).AnyTimes().Return(cloudfoundry.GUIDNil, nil)
			mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).AnyTimes().Return(cloudfoundry.GUIDNil, nil)
			gomock.InOrder(
				mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(nil, errors.New("some error")),
				mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(emptyUsageEvents, nil),
			)
			gomock.InOrder(
				mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(nil, errors.New("some error")),
				mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(emptyUsageEvents, nil),
			)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 2).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)
	})

	Context("When the FetchLastGUID call fails", func() {
		It("should handle the error and retry in the next iteration", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := &Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "10",
				RecordMinAge:    "1m",
			}
			collector, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			gomock.InOrder(
				mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).Return(nil, errors.New("some error")),
				mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).Return(cloudfoundry.GUIDNil, nil),
			)

			gomock.InOrder(
				mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).Return(nil, errors.New("some error")),
				mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).Return(cloudfoundry.GUIDNil, nil),
			)

			mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(emptyUsageEvents, nil)
			mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 10, time.Minute).Return(emptyUsageEvents, nil)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 2).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)
	})

	Context("When the InsertUsageEventList call fails", func() {
		It("should handle the error and retry in the next iteration", func(done Done) {
			defer close(done)

			testReporter.CatchErrors = true

			config := &Config{
				DefaultSchedule: "0.2s",
				MinWaitTime:     "0.1s",
				FetchLimit:      "2",
				RecordMinAge:    "1m",
			}
			collector, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
			Expect(err).ToNot(HaveOccurred())

			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				defer GinkgoRecover()
				collector.Run(ctx)
				c <- true
			}()

			mockSQLClient.EXPECT().FetchLastGUID(db.AppUsageTableName).AnyTimes().Return(cloudfoundry.GUIDNil, nil)
			mockSQLClient.EXPECT().FetchLastGUID(db.ServiceUsageTableName).AnyTimes().Return(cloudfoundry.GUIDNil, nil)

			mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 2, time.Minute).Return(someAppEvents, nil)
			mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 2, time.Minute).Return(someServiceEvents, nil)

			mockSQLClient.EXPECT().InsertUsageEventList(someAppEvents, db.AppUsageTableName).Return(errors.New("some error"))
			mockSQLClient.EXPECT().InsertUsageEventList(someServiceEvents, db.ServiceUsageTableName).Return(errors.New("some error"))

			mockAppClient.EXPECT().Get(cloudfoundry.GUIDNil, 2, time.Minute).Return(someAppEvents, nil)
			mockServiceClient.EXPECT().Get(cloudfoundry.GUIDNil, 2, time.Minute).Return(someServiceEvents, nil)

			mockSQLClient.EXPECT().InsertUsageEventList(someAppEvents, db.AppUsageTableName).Return(nil)
			mockSQLClient.EXPECT().InsertUsageEventList(someServiceEvents, db.ServiceUsageTableName).Return(nil)

			Eventually(func() error {
				mockCtrl.Finish()
				defer func() {
					testReporter.Error = nil
				}()
				return testReporter.Error
			}, 2).Should(BeNil())

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 3)
	})

	Describe("Wrong configuration values", func() {

		Context("When DefaultSchedule is not a valid time duration", func() {
			It("should return with error", func() {
				config := &Config{
					DefaultSchedule: "xxx",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DefaultSchedule is invalid"))
			})
		})

		Context("When MinWaitTime is not a valid time duration", func() {
			It("should return with error", func() {
				config := &Config{
					MinWaitTime: "xxx",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("MinWaitTime is invalid"))
			})
		})

		Context("When RecordMinAge is not a valid time duration", func() {
			It("should return with error", func() {
				config := &Config{
					RecordMinAge: "xxx",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("RecordMinAge is invalid"))
			})
		})

		Context("When FetchLimit is not a number", func() {
			It("should return with error", func() {
				config := &Config{
					FetchLimit: "xxx",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("FetchLimit is invalid"))
			})
		})

		Context("When FetchLimit is zero", func() {
			It("should return with error", func() {
				config := &Config{
					FetchLimit: "0",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("FetchLimit must be between 1 and 100"))
			})
		})

		Context("When FetchLimit is greater than 100", func() {
			It("should return with error", func() {
				config := &Config{
					FetchLimit: "101",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("FetchLimit must be between 1 and 100"))
			})
		})

	})

	Describe("Valid configuration values", func() {
		Context("When FetchLimit is set", func() {
			It("1 should be a valid value", func() {
				config := &Config{
					FetchLimit: "1",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).ToNot(HaveOccurred())
			})
			It("100 should be a valid value", func() {
				config := &Config{
					FetchLimit: "100",
				}
				_, err := New(mockAppClient, mockServiceClient, mockSQLClient, config, logger)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
