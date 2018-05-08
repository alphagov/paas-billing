package eventstore_test

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ForecastBillingEvents", func() {

	var (
		cfg eventstore.Config
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
	})

	/*-----------------------------------------------------------------------------------*
	     2001-01-01                                                      2001-02-01      .
	       00:00           01:00                           03:00           00:00         .
	         |               |                               |               |           .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   [======APP1=====]   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   .   .   .   .   [==============SRV1=============]   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   :   .   .
	         [==========================APP-PLAN1=======================================>.
	         [==========================SRV-PLAN1=======================================>.
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |_____________________ request range ___________________________|   .   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	*-----------------------------------------------------------------------------------*/
	It("Should return one BillingEvent for each simulated app UsageEvent", func() {
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP-PLAN1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "node-cost",
					Formula:      "($time_in_seconds / 3600) * $number_of_nodes",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		dummyServicePlan := "d77af28f-735f-47d0-8a21-be3163baa0e9"
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  dummyServicePlan,
			ValidFrom: "2001-01-01",
			Name:      "SRV-PLAN1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "storage-cost",
					Formula:      "($time_in_seconds / 3600) * $storage_in_mb",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		filter := eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
			OrgGUIDs:   []string{eventstore.DummyOrgGUID},
		}

		inputEvents := []eventio.UsageEvent{
			{
				EventGUID:     "adf4df0c-5eee-4c38-a2da-486aedebf4fd",
				EventStart:    "2001-01-01T00:00:00+00:00",
				EventStop:     "2001-01-01T01:00:00+00:00",
				ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
				ResourceName:  "APP1",
				ResourceType:  "app",
				OrgGUID:       eventstore.DummyOrgGUID,
				SpaceGUID:     eventstore.DummySpaceGUID,
				PlanGUID:      eventstore.ComputePlanGUID,
				NumberOfNodes: 2,
				MemoryInMB:    64,
				StorageInMB:   0,
			},
			{
				EventGUID:     "be28a570-f485-48e1-87d0-98b7b8b66dfa",
				EventStart:    "2001-01-01T01:00:00+00:00",
				EventStop:     "2001-01-01T03:00:00+00:00",
				ResourceGUID:  "c232edeb-7e6f-4d07-a356-3ab521768b65",
				ResourceName:  "SRV1",
				ResourceType:  "service",
				OrgGUID:       eventstore.DummyOrgGUID,
				SpaceGUID:     eventstore.DummySpaceGUID,
				PlanGUID:      dummyServicePlan,
				NumberOfNodes: 0,
				MemoryInMB:    0,
				StorageInMB:   1024,
			},
		}

		outputEvents, err := store.ForecastBillableEvents(inputEvents, filter)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(outputEvents)).To(Equal(2))

		Expect(outputEvents[0]).To(Equal(eventio.BillableEvent{
			EventGUID:     "adf4df0c-5eee-4c38-a2da-486aedebf4fd",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
			ResourceName:  "APP1",
			ResourceType:  "app",
			OrgGUID:       eventstore.DummyOrgGUID,
			SpaceGUID:     eventstore.DummySpaceGUID,
			PlanGUID:      eventstore.ComputePlanGUID,
			NumberOfNodes: 2,
			MemoryInMB:    64,
			StorageInMB:   0,
			Price: eventio.Price{
				IncVAT: "2.400000000000000000000",
				ExVAT:  "2.00000000000000000000",
				Details: []eventio.PriceComponent{
					{
						Name:         "node-cost",
						PlanName:     "APP-PLAN1",
						Start:        "2001-01-01T00:00:00+00:00",
						Stop:         "2001-01-01T01:00:00+00:00",
						VatRate:      "0.2",
						VatCode:      "Standard",
						CurrencyCode: "GBP",
						CurrencyRate: "1",
						IncVAT:       "2.400000000000000000000",
						ExVAT:        "2.00000000000000000000",
					},
				},
			},
		}))

		Expect(outputEvents[1]).To(Equal(eventio.BillableEvent{
			EventGUID:     "be28a570-f485-48e1-87d0-98b7b8b66dfa",
			EventStart:    "2001-01-01T01:00:00+00:00",
			EventStop:     "2001-01-01T03:00:00+00:00",
			ResourceGUID:  "c232edeb-7e6f-4d07-a356-3ab521768b65",
			ResourceName:  "SRV1",
			ResourceType:  "service",
			OrgGUID:       eventstore.DummyOrgGUID,
			SpaceGUID:     eventstore.DummySpaceGUID,
			PlanGUID:      dummyServicePlan,
			NumberOfNodes: 0,
			MemoryInMB:    0,
			StorageInMB:   1024,
			Price: eventio.Price{
				IncVAT: "2457.60000000000000000",
				ExVAT:  "2048.0000000000000000",
				Details: []eventio.PriceComponent{
					{
						Name:         "storage-cost",
						PlanName:     "SRV-PLAN1",
						Start:        "2001-01-01T01:00:00+00:00",
						Stop:         "2001-01-01T03:00:00+00:00",
						VatRate:      "0.2",
						VatCode:      "Standard",
						CurrencyCode: "GBP",
						CurrencyRate: "1",
						IncVAT:       "2457.60000000000000000",
						ExVAT:        "2048.0000000000000000",
					},
				},
			},
		}))
	})

	It("should never persist simulated events to the store", func() {
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP-PLAN1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "node-cost",
					Formula:      "($time_in_seconds / 3600) * $number_of_nodes",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		filter := eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
			OrgGUIDs:   []string{eventstore.DummyOrgGUID},
		}

		inputEvents := []eventio.UsageEvent{
			{
				EventGUID:     "adf4df0c-5eee-4c38-a2da-486aedebf4fd",
				EventStart:    "2001-01-01T00:00:00+00:00",
				EventStop:     "2001-01-01T01:00:00+00:00",
				ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
				ResourceName:  "APP1",
				ResourceType:  "app",
				OrgGUID:       eventstore.DummyOrgGUID,
				SpaceGUID:     eventstore.DummySpaceGUID,
				PlanGUID:      eventstore.ComputePlanGUID,
				NumberOfNodes: 2,
				MemoryInMB:    64,
				StorageInMB:   0,
			},
		}

		outputEvents, err := store.ForecastBillableEvents(inputEvents, filter)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(outputEvents)).To(Equal(1))

		realEvents, err := store.GetBillableEvents(filter)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(realEvents)).To(Equal(0), "did not expect simulated events to persist as real events")
	})

})
