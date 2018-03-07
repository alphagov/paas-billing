package collector_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/collector"
	"github.com/alphagov/paas-billing/collector/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {

	var (
		config           *collector.Config
		logger           lager.Logger
		fakeEventFetcher *fakes.FakeEventFetcher
		coll             *collector.Collector
		ctx              context.Context
		cancelFunc       context.CancelFunc
	)

	BeforeEach(func() {
		config = &collector.Config{
			DefaultSchedule: time.Duration(200 * time.Millisecond),
			MinWaitTime:     time.Duration(100 * time.Millisecond),
			FetchLimit:      10,
			RecordMinAge:    time.Duration(1 * time.Minute),
		}
		logger = lager.NewLogger("test")
		fakeEventFetcher = &fakes.FakeEventFetcher{}
		coll = collector.New(config, logger, fakeEventFetcher)
		ctx, cancelFunc = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancelFunc()
	})

	It("should fetch events regularly", func() {
		go func() {
			coll.Run(ctx)
		}()

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	It("should handle errors and retry again", func() {
		fakeEventFetcher.FetchEventsReturnsOnCall(0, 0, errors.New("some error"))

		go func() {
			coll.Run(ctx)
		}()

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	It("should wait only the minimum wait time it it fetched a full page", func() {
		config.DefaultSchedule = time.Duration(1 * time.Minute)
		fakeEventFetcher.FetchEventsReturnsOnCall(0, config.FetchLimit, nil)

		go func() {
			coll.Run(ctx)
		}()

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	Context("When the collector gets an interrupt signal", func() {
		It("should stop gracefully", func() {
			ctx, cancelFunc := context.WithCancel(context.Background())

			c := make(chan bool, 0)
			go func() {
				coll.Run(ctx)
				c <- true
			}()

			cancelFunc()

			Expect(<-c).To(BeTrue())
		}, 5)
	})

})
