package collector_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/collector"
	"github.com/alphagov/paas-billing/collector/fakes"
	"github.com/alphagov/paas-billing/store"
	storefakes "github.com/alphagov/paas-billing/store/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {

	var (
		config           *collector.Config
		logger           lager.Logger
		fakeEventFetcher *fakes.FakeEventFetcher
		fakeEventStore   *storefakes.FakeEventStorer
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
		fakeEventStore = &storefakes.FakeEventStorer{}

		coll = collector.New(config, logger, fakeEventFetcher, fakeEventStore)
		ctx, cancelFunc = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancelFunc()
	})

	It("should fetch events regularly", func() {
		go coll.Run(ctx)

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	It("should handle errors and retry again", func() {
		fakeEventFetcher.FetchEventsReturnsOnCall(0, []store.RawEvent{}, errors.New("some error"))

		go coll.Run(ctx)

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	It("should wait only the minimum wait time if collected FetchLimit number of events", func() {
		config.DefaultSchedule = time.Duration(1 * time.Minute)
		collectedEvents := make([]store.RawEvent, config.FetchLimit)
		fakeEventFetcher.FetchEventsReturnsOnCall(0, collectedEvents, nil)
		fakeEventStore.StoreEventsReturnsOnCall(0, nil)

		go coll.Run(ctx)

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5).Should(BeNumerically(">", 1))
	}, 5)

	It("should stop gracefully when sent an interrupt signal", func() {
		ctx, cancelFunc := context.WithCancel(context.Background())

		c := make(chan bool)
		go func() {
			coll.Run(ctx)
			c <- true
		}()

		cancelFunc()

		Expect(<-c).To(BeTrue())
	}, 5)

})
