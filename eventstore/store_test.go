package eventstore_test

import (
	"encoding/json"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {

	var (
		cfg eventstore.Config
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
	})

	It("should be idempotent", func() {
		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		Expect(db.Schema.Init()).To(Succeed())
		Expect(db.Schema.Init()).To(Succeed())
	})








	It("should ensure plan has unique plan_guid + valid_from", func() {
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			ValidTo:   "9999-12-31",
			Name:      "APP_PLAN_1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			ValidTo:   "9999-12-31",
			Name:      "APP_PLAN_1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		db, err := testenv.Open(cfg)
		Expect(err).To(MatchError(ContainSubstring(`violates unique constraint`)))
		if err == nil {
			db.Close()
		}
	})

	DescribeTable("reject placing plans with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				PricingPlans: []eventio.PricingPlan{
					{
						PlanGUID:  uuid.NewV4().String(),
						ValidFrom: timestamp,
						ValidTo:   "9999-12-31",
						Name:      "bad-plan",
						Components: []eventio.PricingPlanComponent{
							{
								Name:         "compute",
								Formula:      "1",
								CurrencyCode: "GBP",
								VATCode:      "Standard",
							},
						},
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_month"`)))
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("reject vat_rates with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventio.VATRate{
					{
						ValidFrom: timestamp,
						ValidTo:   "9999-12-31",
						Code:      "Standard",
						Rate:      0,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_month"`)))
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("allow currency_rates with valid_from that isn't the first day of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventio.CurrencyRate{
					{
						ValidFrom: timestamp,
						ValidTo:   "9999-12-31",
						Code:      "USD",
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("first day of month", "2017-05-01T00:00:00Z"),
	)

	DescribeTable("reject currency_rates with valid_from that isn't the start of a day",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventio.CurrencyRate{
					{
						ValidFrom: timestamp,
						ValidTo:   "9999-12-31",
						Code:      "USD",
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_day"`)))
		},
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("allow whitelisted currency codes",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventio.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
						ValidTo:   "9999-12-31",
						Code:      code,
						Rate:      1,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer db.Close()
			}
		},
		Entry("£ UK Sterling", "GBP"),
		Entry("$ US Dollar", "USD"),
		Entry("€ Euro", "EUR"),
	)

	DescribeTable("reject unknown currency_codes",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventio.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
						ValidTo:   "9999-12-31",
						Code:      code,
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`invalid currency rate: invalid input value for enum currency_code`)))
		},
		Entry("no lowercase", "usd"),
		Entry("no symbols", "$"),
		Entry("no random codes", "UKP"),
		Entry("no unknown", "XXX"),
	)

	DescribeTable("allow whitelisted vat_rates",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventio.VATRate{
					{
						ValidFrom: "2001-01-01",
						ValidTo:   "9999-12-31",
						Code:      code,
						Rate:      0.1,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer db.Close()
			}
		},
		Entry("allow: Standard", "Standard"),
		Entry("allow: Reduced", "Reduced"),
		Entry("allow: Zero", "Zero"),
	)

	DescribeTable("reject unknown vat_rates",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventio.VATRate{
					{
						ValidFrom: "2001-01-01",
						ValidTo:   "9999-12-31",
						Code:      code,
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`invalid vat rate: invalid input value for enum vat_code`)))
		},
		Entry("no lowercase", "standard"),
		Entry("no uppercase", "ZERO"),
		Entry("no random codes", "myrate"),
	)

	DescribeTable("should store events of difference kinds",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("attempting to store a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching the stored events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				marshalledStoredEvents, err :=  json.Marshal(storedEvents)
				Expect(err).ToNot(HaveOccurred())
				marshalledInputEvents, err := json.Marshal([]eventio.RawEvent{
					event3,
					event2,
					event1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(marshalledStoredEvents)).To(Equal(string(marshalledInputEvents)))

			})
		},
		Entry("app usage event", "app"),
		Entry("service usage event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should not commit when batch contains invalid app event",
		func(kind string, expectedErr string, badEvent eventio.RawEvent) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("attempting to store a bad batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					badEvent,
					event3,
				})
				Expect(err).To(MatchError(ContainSubstring(expectedErr)))
			})
			By("fetching no events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{}))
			})
		},
		Entry("app event with no GUID", "app", "must have a GUID", eventio.RawEvent{
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:       "app",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no CreatedAt", "app", "must have a CreatedAt", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			Kind:       "app",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no Kind", "app", "must have a Kind", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no RawMessage", "app", "must have a RawMessage payload", eventio.RawEvent{
			GUID:      "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt: time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:      "app",
		}),
		Entry("compose event with no GUID", "compose", "must have a GUID", eventio.RawEvent{
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:       "compose",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no CreatedAt", "compose", "must have a CreatedAt", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			Kind:       "compose",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no Kind", "compose", "must have a Kind", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no RawMessage", "compose", "must have a RawMessage payload", eventio.RawEvent{
			GUID:      "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt: time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:      "compose",
		}),
	)

	DescribeTable("should be an error to GetEvents with invalid Kind",
		func(kind string, expectedErr string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
				Kind: kind,
			})
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
			Expect(storedEvents).To(BeNil())
		},
		Entry("unset kind", "", "you must supply a kind to filter events by"),
		Entry("unknown kind", "unknown", "cannot query events of kind 'unknown'"),
	)

	DescribeTable("should ignore events that already exist in the database",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("inserting new same batch again", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching all events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				marshalledStoredEvents, err :=  json.Marshal(storedEvents)
				Expect(err).ToNot(HaveOccurred())
				marshalledInputEvents, err := json.Marshal([]eventio.RawEvent{
					event3,
					event2,
					event1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(marshalledStoredEvents)).To(Equal(string(marshalledInputEvents)))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should be able to fetch only the LAST known event",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching back a single event", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind:  kind,
					Limit: 1,
				})
				Expect(err).ToNot(HaveOccurred())
				marshalledStoredEvents, err :=  json.Marshal(storedEvents)
				Expect(err).ToNot(HaveOccurred())
				marshalledInputEvents, err := json.Marshal([]eventio.RawEvent{
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(marshalledStoredEvents)).To(Equal(string(marshalledInputEvents)))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should be able to fetch only the FIRST known event",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching back a single event", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind:    kind,
					Reverse: true,
					Limit:   1,
				})
				Expect(err).ToNot(HaveOccurred())
				marshalledDBEvents, err :=  json.Marshal(storedEvents)
				Expect(err).ToNot(HaveOccurred())
				marshalledInputEvents, err := json.Marshal([]eventio.RawEvent{
					event1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(marshalledDBEvents)).To(Equal(string(marshalledInputEvents)))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

	Describe("pg_size_bytes", func() {
		var db *testenv.TempDB

		BeforeEach(func() {
			var err error
			db, err = testenv.Open(cfg)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			db.Close()
		})

		DescribeTable("valid inputs",
			func(input string, expected int) {
				var output int
				err := db.Conn.QueryRow(`select pg_size_bytes('` + input + `')`).Scan(&output)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(expected))
			},
			Entry("1", "1", 1),
			Entry("123bytes", "123bytes", 123),
			Entry("1kB", "1kB", 1024),
			Entry("1MB", "1MB", 1048576),
			Entry(" 1 GB", " 1 GB", 1073741824),
			Entry("1.5 GB", "1.5 GB", 1610612736),
			Entry("1TB", "1TB", 1099511627776),
			Entry("3000 TB", "3000 TB", 3298534883328000),
			Entry("1e6 MB", "1e6 MB", 1048576000000),

			// case-insensitive units are supported
			Entry("1", "1", 1),
			Entry("123bYteS", "123bYteS", 123),
			Entry("1kb", "1kb", 1024),
			Entry("1mb", "1mb", 1048576),
			Entry(" 1 Gb", " 1 Gb", 1073741824),
			Entry("1.5 gB", "1.5 gB", 1610612736),
			Entry("1tb", "1tb", 1099511627776),
			Entry("3000 tb", "3000 tb", 3298534883328000),
			Entry("1e6 mb", "1e6 mb", 1048576000000),

			// negative numbers are supported
			Entry("-1", "-1", -1),
			Entry("-123bytes", "-123bytes", -123),
			Entry("-1kb", "-1kb", -1024),
			Entry("-1mb", "-1mb", -1048576),
			Entry(" -1 Gb", " -1 Gb", -1073741824),
			Entry("-1.5 gB", "-1.5 gB", -1610612736),
			Entry("-1tb", "-1tb", -1099511627776),
			Entry("-3000 TB", "-3000 TB", -3298534883328000),
			Entry("-10e-1 MB", "-10e-1 MB", -1048576),

			// different cases with allowed points
			Entry("-1.", "-1.", -1),
			Entry("-1.kb", "-1.kb", -1024),
			Entry("-1. kb", "-1. kb", -1024),
			Entry("-0. gb", "-0. gb", 0),
			Entry("-.1", "-.1", 0),
			Entry("-.1kb", "-.1kb", -102),
			Entry("-.1 kb", "-.1 kb", -102),
			Entry("-.0 gb", "-.0 gb", 0),
		)

		DescribeTable("invalid inputs",
			func(input string) {
				_, err := db.Conn.Query(`select pg_size_bytes('` + input + `')`)
				Expect(err).To(HaveOccurred())
			},
			Entry("1 AB", "1 AB"),
			Entry("1 AB A", "1 AB A"),
			Entry("1 AB A    ", "1 AB A    "),
			Entry("9223372036854775807.9", "9223372036854775807.9"),
			Entry("1e100", "1e100"),
			Entry("1e1000000000000000000", "1e1000000000000000000"),
			Entry("1 byte", "1 byte"), // the singular "byte" is not supported
			Entry("", ""),
			Entry("kb", "kb"),
			Entry("..", ".."),
			Entry("-.", "-."),
			Entry("-.kb", "-.kb"),
			Entry("-. kb", "-. kb"),
			Entry(".+912", ".+912"),
			Entry("+912+ kB", "+912+ kB"),
			Entry("++123 kB", "++123 kB"),
		)
	})
})
