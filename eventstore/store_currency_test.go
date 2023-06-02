package eventstore_test

import (
	"encoding/json"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Currency Conversion", func() {
	var (
		cfg            eventstore.Config
		db             *testenv.TempDB
		err            error
		app1EventStart = eventio.RawEvent{
			GUID:       "aa11a111-a111-11a1-11a1-11a1a1a11aaa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		app1EventStop = eventio.RawEvent{
			GUID:       "bb22b222-b222-22b2-22b2-22b2b2b22bbb",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 3, 1, 1, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STOPPED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
	)

	/*---------------------------------------------------------------------------------------*
	  2001-01-01T00:00                                                     2001-01-01T01:00  .
	         |                                                                       |   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [===============================APP1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==============================PLAN1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==========================GBP-CurrencyRate1============================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	----------------------------------------------------------------------------------------*/
	It("should not affect price when using a GBP rate of 1", func(ctx SpecContext) {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "epoch",
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "GBP",
					Rate:      1,
					ValidFrom: "epoch",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "epoch",
					Name:      "PLAN1",
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
		}
		db, err = testenv.OpenWithContext(cfg, ctx)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		Expect(store.StoreEvents([]eventio.RawEvent{
			app1EventStart,
			app1EventStop,
		})).To(Succeed())

		Expect(store.Refresh()).To(Succeed())

		events, err := store.GetBillableEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(events)).To(BeNumerically("==", 1), "expected a single event to be returned")
		Expect(events[0].Price.ExVAT).To(Equal("1"))
		Expect(events[0].Price.IncVAT).To(Equal("1.2"))
		Expect(len(events[0].Price.Details)).To(BeNumerically("==", 1), "expected a single event component to be returned")
	})

	/*---------------------------------------------------------------------------------------*
	  2001-01-01T00:00                                                     2001-01-01T01:00  .
	         |                                                                       |   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [===============================APP1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==============================PLAN1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==========================USD-CurrencyRate1============================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	----------------------------------------------------------------------------------------*/
	It("converts USD at the defined rate", func(ctx SpecContext) {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "epoch",
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "USD",
					Rate:      0.8,
					ValidFrom: "epoch",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "epoch",
					Name:      "PLAN1",
					Components: []eventio.PricingPlanComponent{
						{
							Name:         "compute",
							Formula:      "100",
							CurrencyCode: "USD",
							VATCode:      "Standard",
						},
					},
				},
			},
		}

		db, err = testenv.OpenWithContext(cfg, ctx)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		Expect(store.StoreEvents([]eventio.RawEvent{
			app1EventStart,
			app1EventStop,
		})).To(Succeed())

		Expect(store.Refresh()).To(Succeed())

		events, err := store.GetBillableEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(events)).To(BeNumerically("==", 1), "expected a single event to be returned")
		Expect(events[0].Price.ExVAT).To(Equal("80.0"))
		Expect(events[0].Price.IncVAT).To(Equal("96.00"))
		Expect(len(events[0].Price.Details)).To(BeNumerically("==", 1), "expected a single event component to be returned")
	})

	/*---------------------------------------------------------------------------------------*
	     2001-01-01                        2001-02-01                           2001-03-01   .
	         |                                 |                                     |   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [===============================APP1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==============================PLAN1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==========CurrencyRate1==========][==========CurrencyRate2=============]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	----------------------------------------------------------------------------------------*/
	It("should return one BillableEvent with multiple pricing components when range intercepts changing currency rates", func(ctx SpecContext) {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "epoch",
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "USD",
					Rate:      2,
					ValidFrom: "epoch",
				},
				{
					Code:      "USD",
					Rate:      4,
					ValidFrom: "2001-02-01",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "epoch",
					Name:      "PLAN1",
					Components: []eventio.PricingPlanComponent{
						{
							Name:         "compute",
							Formula:      "1",
							CurrencyCode: "USD",
							VATCode:      "Standard",
						},
					},
				},
			},
		}

		db, err = testenv.OpenWithContext(cfg, ctx)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		Expect(store.StoreEvents([]eventio.RawEvent{
			app1EventStart,
			app1EventStop,
		})).To(Succeed())

		Expect(store.Refresh()).To(Succeed())

		events, err := store.GetBillableEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-03-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(events)).To(BeNumerically("==", 1), "expected a single event to be returned")
		Expect(events[0].Price.ExVAT).To(Equal("6"))
		Expect(events[0].Price.IncVAT).To(Equal("7.2"))

		Expect(len(events[0].Price.Details)).To(BeNumerically("==", 2), "expected two event components to be returned")
		Expect(events[0].Price.Details[0].ExVAT).To(Equal("2"))
		Expect(events[0].Price.Details[0].IncVAT).To(Equal("2.4"))
		Expect(events[0].Price.Details[1].ExVAT).To(Equal("4"))
		Expect(events[0].Price.Details[1].IncVAT).To(Equal("4.8"))
	})

	/*---------------------------------------------------------------------------------------*
	     2001-01-01                        2001-02-01                           2001-03-01   .
	         |                                 |                                     |   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [===============================APP1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==============================PLAN1====================================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==========================GBP-CurrencyRate=============================]   .   .
	 .   .   [==========================USD-CurrencyRate=============================]   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	----------------------------------------------------------------------------------------*/
	It("should return single BillableEvent with two pricing components with different currencies", func(ctx SpecContext) {
		cfg = eventstore.Config{
			VATRates: []eventio.VATRate{
				{
					Code:      "Standard",
					Rate:      0.2,
					ValidFrom: "epoch",
				},
			},
			CurrencyRates: []eventio.CurrencyRate{
				{
					Code:      "GBP",
					Rate:      1,
					ValidFrom: "2001-01-01",
				},
				{
					Code:      "USD",
					Rate:      2,
					ValidFrom: "2001-01-01",
				},
			},
			PricingPlans: []eventio.PricingPlan{
				{
					PlanGUID:  eventstore.ComputePlanGUID,
					ValidFrom: "2001-01-01",
					Name:      "PLAN1",
					Components: []eventio.PricingPlanComponent{
						{
							Name:         "little-price",
							Formula:      "1",
							CurrencyCode: "GBP",
							VATCode:      "Standard",
						},
						{
							Name:         "big-price",
							Formula:      "100",
							CurrencyCode: "USD",
							VATCode:      "Standard",
						},
					},
				},
			},
		}

		db, err = testenv.OpenWithContext(cfg, ctx)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		Expect(store.StoreEvents([]eventio.RawEvent{
			app1EventStart,
			app1EventStop,
		})).To(Succeed())

		Expect(store.Refresh()).To(Succeed())

		events, err := store.GetBillableEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-03-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(events)).To(BeNumerically("==", 1), "expected a single event to be returned")
		Expect(len(events[0].Price.Details)).To(BeNumerically("==", 2), "expected two event components to be returned")

		Expect(events[0].Price.Details[0].ExVAT).To(Equal("1"))
		Expect(events[0].Price.Details[0].IncVAT).To(Equal("1.2"))
		Expect(events[0].Price.Details[0].CurrencyCode).To(Equal("GBP"))
		Expect(events[0].Price.Details[1].ExVAT).To(Equal("200"))
		Expect(events[0].Price.Details[1].IncVAT).To(Equal("240.0"))
		Expect(events[0].Price.Details[1].CurrencyCode).To(Equal("GBP"))
	})

})
