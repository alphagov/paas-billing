package eventcollector_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/fakes"

	. "github.com/alphagov/paas-billing/eventcollector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {

	var (
		logger           = lager.NewLogger("test")
		fakeEventFetcher *fakes.FakeEventFetcher
		fakeEventStore   *fakes.FakeEventStore
		cfg              Config
		ctx              context.Context
		cancelFunc       context.CancelFunc
	)

	BeforeEach(func() {
		fakeEventFetcher = &fakes.FakeEventFetcher{}
		fakeEventStore = &fakes.FakeEventStore{}
		cfg = Config{
			Logger:      logger,
			Fetcher:     fakeEventFetcher,
			Store:       fakeEventStore,
			Schedule:    time.Duration(200 * time.Millisecond),
			MinWaitTime: time.Duration(100 * time.Millisecond),
		}
		ctx, cancelFunc = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancelFunc()
	})

	It("should fetch events regularly", func() {
		fakeEventFetcher.FetchEventsReturns([]eventio.RawEvent{}, nil)
		fakeEventStore.StoreEventsReturns(nil)
		fakeEventStore.GetEventsReturns([]eventio.RawEvent{}, nil)

		go New(cfg).Run(ctx)

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5*time.Second).Should(BeNumerically(">", 3))
		Eventually(fakeEventStore.StoreEventsCallCount, 5*time.Second).Should(BeNumerically(">", 3))
		Eventually(fakeEventStore.GetEventsCallCount, 5*time.Second).Should(BeNumerically(">", 3))
	})

	It("should handle errors and retry again", func() {
		fakeEventFetcher.FetchEventsReturnsOnCall(0, []eventio.RawEvent{}, errors.New("some error"))

		go New(cfg).Run(ctx)

		Eventually(fakeEventFetcher.FetchEventsCallCount, 5*time.Second).Should(BeNumerically(">", 1))
	})

	It("should wait only the MinWaitTime if we have not already seen the last fetched event", func() {
		cfg.Schedule = 999 * time.Minute
		cfg.MinWaitTime = 250 * time.Millisecond

		fakeEventStore.GetEventsReturns([]eventio.RawEvent{{GUID: "last-event"}}, nil)
		fakeEventFetcher.FetchEventsReturns([]eventio.RawEvent{{GUID: "unseen-event"}}, nil)
		fakeEventStore.StoreEventsReturns(nil)

		go New(cfg).Run(ctx)

		time.Sleep(900 * time.Millisecond)

		Eventually(fakeEventFetcher.FetchEventsCallCount(), 5*time.Second).Should(Equal(4))
	})

	It("should wait the full ScheduleWaitTime if no new events have been seen", func() {
		cfg.Schedule = 300 * time.Millisecond
		cfg.MinWaitTime = 0

		fakeEventStore.GetEventsReturns([]eventio.RawEvent{{GUID: "seen-event"}}, nil)
		fakeEventFetcher.FetchEventsReturns([]eventio.RawEvent{{GUID: "seen-event"}}, nil)
		fakeEventStore.StoreEventsReturns(nil)

		go New(cfg).Run(ctx)

		time.Sleep(650 * time.Millisecond)

		Eventually(fakeEventFetcher.FetchEventsCallCount(), 5*time.Second).Should(Equal(3))
	})

	It("should wait the full ScheduleWaitTime if no new events have been fetched with a lastEvent given", func() {
		cfg.Schedule = 300 * time.Millisecond
		cfg.MinWaitTime = 0

		fakeEventStore.GetEventsReturns([]eventio.RawEvent{{GUID: "seen-event"}}, nil)
		fakeEventFetcher.FetchEventsReturns([]eventio.RawEvent{}, nil)
		fakeEventStore.StoreEventsReturns(nil)

		go New(cfg).Run(ctx)

		time.Sleep(650 * time.Millisecond)

		Eventually(fakeEventFetcher.FetchEventsCallCount(), 5*time.Second).Should(Equal(3))
	})

	It("should wait the full ScheduleWaitTime if no new events have been fetched and no lastEvent given", func() {
		cfg.Schedule = 300 * time.Millisecond
		cfg.MinWaitTime = 0

		fakeEventStore.GetEventsReturns([]eventio.RawEvent{{GUID: "seen-event"}}, nil)
		fakeEventFetcher.FetchEventsReturns(nil, nil)
		fakeEventStore.StoreEventsReturns(nil)

		go New(cfg).Run(ctx)

		time.Sleep(650 * time.Millisecond)

		Eventually(fakeEventFetcher.FetchEventsCallCount(), 5*time.Second).Should(Equal(3))
	})

	It("should stop gracefully when context is cancelled", func() {
		ctx, cancelFunc := context.WithCancel(context.Background())

		c := make(chan bool)
		go func() {
			New(cfg).Run(ctx)
			c <- true
		}()

		cancelFunc()

		Eventually(<-c, 5*time.Second).Should(BeTrue())
	})
})
