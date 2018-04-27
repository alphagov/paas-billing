package eventstore_test

import (
	"encoding/json"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetUsageEvents", func() {

	var (
		cfg eventstore.Config
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
	})

	/*-----------------------------------------------------------------------------------*
	     2001-01-01                                                      2002-01-01      .
	       00:00           01:00                                           00:00         .
	         |               |                                               |           .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   [======APP1=====]   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   [======DB1======]   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   :   .   .
	 .   .   |_____________________ request range ___________________________|   .   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	*-----------------------------------------------------------------------------------*/
	It("should return one UsageEvent for each STARTED/STOPPED pair of RawEvent", func() {
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom: "2001-01-01",
			Name:      "DB_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 1",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		app1EventStart := eventio.RawEvent{
			GUID:       "ae28a570-f485-48e1-87d0-98b7b8b66dfa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
		}
		app1EventStop := eventio.RawEvent{
			GUID:       "bd9036c5-8367-497d-bb56-94bfcac6621a",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STOPPED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
		}
		service1EventStart := eventio.RawEvent{
			GUID:       "c497eb13-f48a-4859-be53-5569f302b516",
			Kind:       "service",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "DB1", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := eventio.RawEvent{
			GUID:       "dd52b4f4-9e33-4504-8fca-fd9e33af11a6",
			Kind:       "service",
			CreatedAt:  time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "DB1", "service_instance_type": "managed_service_instance"}`),
		}

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		store := db.Schema

		Expect(store.StoreEvents([]eventio.RawEvent{
			app1EventStart,
			app1EventStop,
			service1EventStart,
			service1EventStop,
		})).To(Succeed())
		Expect(db.Schema.Refresh()).To(Succeed())

		usageEvents, err := store.GetUsageEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2002-01-01",
			OrgGUIDs:   []string{"51ba75ef-edc0-47ad-a633-a8f6e8770944"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(usageEvents)).To(Equal(2))

		Expect(usageEvents[0]).To(Equal(eventio.UsageEvent{
			EventGUID:     "ae28a570-f485-48e1-87d0-98b7b8b66dfa",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
			ResourceName:  "APP1",
			ResourceType:  "app",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:      eventstore.ComputePlanGUID,
			NumberOfNodes: 10,
			MemoryInMB:    1000,
			StorageInMB:   0,
		}))

		Expect(usageEvents[1]).To(Equal(eventio.UsageEvent{
			EventGUID:     "c497eb13-f48a-4859-be53-5569f302b516",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "f3f98365-6a95-4bbd-ab8f-527a7957a41f",
			ResourceName:  "DB1",
			ResourceType:  "postgres",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d",
			PlanGUID:      "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			NumberOfNodes: 0,
			MemoryInMB:    0,
			StorageInMB:   0,
		}))

	})

})
