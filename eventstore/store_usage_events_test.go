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
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
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
			PlanGUID:  eventstore.StagingPlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "STAGING_PLAN_1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.03",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom: "2001-01-01",
			Name:      "DB_PLAN_1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 1",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		app1EventBuildpackSet := eventio.RawEvent{
			GUID:       "ae28a570-f485-48e1-87d0-98b7b8b66dfa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "BUILDPACK_SET", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
		}
		app1EventStagingStart := eventio.RawEvent{
			GUID:       "ae28a571-f485-48e1-87d0-98b7b8b66dfa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STAGING_STARTED", "parent_app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "parent_app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
		}
		app1EventStart := eventio.RawEvent{
			GUID:       "ae28a572-f485-48e1-87d0-98b7b8b66dfa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
		}
		app1EventStagingStop := eventio.RawEvent{
			GUID:       "ae28a573-f485-48e1-87d0-98b7b8b66dfa",
			Kind:       "app",
			CreatedAt:  time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC),
			RawMessage: json.RawMessage(`{"state": "STAGING_STOPPED", "parent_app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "parent_app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 10, "previous_state": "STARTED", "memory_in_mb_per_instance": 1000}`),
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
			app1EventBuildpackSet,
			app1EventStagingStart,
			app1EventStart,
			app1EventStagingStop,
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
		Expect(usageEvents).To(HaveLen(3))

		Expect(usageEvents[0]).To(Equal(eventio.UsageEvent{
			EventGUID:     "ae28a571-f485-48e1-87d0-98b7b8b66dfa",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T00:01:00+00:00",
			ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
			ResourceName:  "APP1",
			ResourceType:  "app",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:      eventstore.StagingPlanGUID,
			PlanName:      "staging",
			ServiceGUID:   eventstore.ComputeServiceGUID,
			ServiceName:   "app",
			NumberOfNodes: 1,
			MemoryInMB:    1000,
			StorageInMB:   0,
		}))
		Expect(usageEvents[1]).To(Equal(eventio.UsageEvent{
			EventGUID:     "ae28a572-f485-48e1-87d0-98b7b8b66dfa",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "c85e98f0-6d1b-4f45-9368-ea58263165a0",
			ResourceName:  "APP1",
			ResourceType:  "app",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:      eventstore.ComputePlanGUID,
			PlanName:      "app",
			ServiceGUID:   eventstore.ComputeServiceGUID,
			ServiceName:   "app",
			NumberOfNodes: 10,
			MemoryInMB:    1000,
			StorageInMB:   0,
		}))

		Expect(usageEvents[2]).To(Equal(eventio.UsageEvent{
			EventGUID:     "c497eb13-f48a-4859-be53-5569f302b516",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "f3f98365-6a95-4bbd-ab8f-527a7957a41f",
			ResourceName:  "DB1",
			ResourceType:  "service",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d",
			PlanGUID:      "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:      "Free",
			ServiceGUID:   "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:   "postgres",
			NumberOfNodes: 0,
			MemoryInMB:    0,
			StorageInMB:   0,
		}))

	})

	/*-----------------------------------------------------------------------------------*
	       00:00       01:00       02:00   03:00   04:00                                   .
	         |           |           |       |       |                                    .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [================db1====================]   .   .   .   .   .   .   .   .   .   .
	 .   .   |   .   .   |   .   .   |   .   |   .   |   .   .   .   .   .   .   .   .   .
	       start      update      scale    update   stop                                      .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	<=======================================PLAN1=======================================>.
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	-------------------------------------------------------------------------------------*/
	It("Should use the memory and storage values from compose scaling events if available", func() {
		cfg.AddVATRate(eventio.VATRate{
			Code:      "Zero",
			Rate:      0,
			ValidFrom: "epoch",
		})
		plan := eventio.PricingPlan{
			PlanGUID:      "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom:     "2001-01-01",
			Name:          "PLAN1",
			NumberOfNodes: 1,
			MemoryInMB:    1024,
			StorageInMB:   2048,
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compose",
					Formula:      "ceil($time_in_seconds/3600) * $memory_in_mb * $storage_in_mb * $number_of_nodes",
					CurrencyCode: "GBP",
					VATCode:      "Zero",
				},
			},
		}
		cfg.AddPlan(plan)

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		service1EventStart := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000001",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventUpdate1 := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000002",
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "UPDATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1-renamed", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventScale := testenv.Row{
			"event_id":    "audit-id-000000000003",
			"created_at":  "2001-01-01T02:00Z",
			"raw_message": json.RawMessage(` {"id": "audit-id-000000000003", "ip": "", "data": {"units": "2", "memory": "2 GB", "cluster": "gds-eu-west1-c00", "storage": "4 GB", "deployment": "prod-aaaaaaaa-0000-0000-0000-000000000001"}, "event": "deployment.scale.members", "_links": {"alerts": {"href": "", "templated": false}, "backups": {"href": "", "templated": false}, "cluster": {"href": "", "templated": false}, "scalings": {"href": "", "templated": false}, "portal_users": {"href": "", "templated": false}, "compose_web_ui": {"href": "", "templated": false}}, "user_id": "", "account_id": "58d3e39c0045bb00135ee6ad", "cluster_id": "5941cf9f859d2c0015000021", "created_at": "2001-01-01T02:00:00.000Z", "user_agent": "", "deployment_id": "59de3e8cc9ecc40010324fc6"}`),
		}
		service1EventUpdate2 := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000004",
			"created_at":  "2001-01-01T03:00Z",
			"raw_message": json.RawMessage(`{"state": "UPDATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1-renamed-again", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000005",
			"created_at":  "2001-01-01T04:00Z",
			"raw_message": json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1-renamed-again", "service_instance_type": "managed_service_instance"}`),
		}

		Expect(db.Insert("service_usage_events", service1EventStart, service1EventUpdate1, service1EventUpdate2, service1EventStop)).To(Succeed())
		Expect(db.Insert("compose_audit_events", service1EventScale)).To(Succeed())

		Expect(db.Schema.Refresh()).To(Succeed())

		events, err := db.Schema.GetUsageEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(events[0]).To(Equal(eventio.UsageEvent{
			EventGUID:    "00000000-0000-0000-0000-000000000001",
			EventStart:   "2001-01-01T00:00:00+00:00",
			EventStop:    "2001-01-01T01:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
		}))

		events[1].EventGUID = "randomly-generated-event-1"
		Expect(events[1]).To(Equal(eventio.UsageEvent{
			EventGUID:    "randomly-generated-event-1",
			EventStart:   "2001-01-01T01:00:00+00:00",
			EventStop:    "2001-01-01T02:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1-renamed",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
		}))

		events[2].EventGUID = "randomly-generated-event-2"
		Expect(events[2]).To(Equal(eventio.UsageEvent{
			EventGUID:    "randomly-generated-event-2",
			EventStart:   "2001-01-01T02:00:00+00:00",
			EventStop:    "2001-01-01T03:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1-renamed",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
			MemoryInMB:   2048,
			StorageInMB:  4096,
		}))

		events[3].EventGUID = "randomly-generated-event-3"
		Expect(events[3]).To(Equal(eventio.UsageEvent{
			EventGUID:    "randomly-generated-event-3",
			EventStart:   "2001-01-01T03:00:00+00:00",
			EventStop:    "2001-01-01T04:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1-renamed-again",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
			MemoryInMB:   2048,
			StorageInMB:  4096,
		}))
	})

	/*-----------------------------------------------------------------------------------*
	       00:00                   01:00               02:00                             .
	         |                       |                   |                               .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	 .   .   [==================db1======================]   .   .   .   .   .   .   .   .
	 .   .   |   .   .               |   .   .   .       |   .   .   .   .   .   .   .   .
	    compose-scale               start              stop                              .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	<=======================================PLAN1=======================================>.
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	-------------------------------------------------------------------------------------*/
	It("should use the compose event as the EventStart  ", func() {
		cfg.AddVATRate(eventio.VATRate{
			Code:      "Zero",
			Rate:      0,
			ValidFrom: "epoch",
		})
		plan := eventio.PricingPlan{
			PlanGUID:      "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom:     "2001-01-01",
			Name:          "PLAN1",
			NumberOfNodes: 1,
			MemoryInMB:    1024,
			StorageInMB:   2048,
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compose",
					Formula:      "ceil($time_in_seconds/3600) * $memory_in_mb * $storage_in_mb * $number_of_nodes",
					CurrencyCode: "GBP",
					VATCode:      "Zero",
				},
			},
		}
		cfg.AddPlan(plan)

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		service1EventScale := testenv.Row{
			"event_id":    "audit-id-000000000003",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(` {"id": "audit-id-000000000003", "ip": "", "data": {"units": "2", "memory": "2 GB", "cluster": "gds-eu-west1-c00", "storage": "4 GB", "deployment": "prod-aaaaaaaa-0000-0000-0000-000000000001"}, "event": "deployment.scale.members", "_links": {"alerts": {"href": "", "templated": false}, "backups": {"href": "", "templated": false}, "cluster": {"href": "", "templated": false}, "scalings": {"href": "", "templated": false}, "portal_users": {"href": "", "templated": false}, "compose_web_ui": {"href": "", "templated": false}}, "user_id": "", "account_id": "58d3e39c0045bb00135ee6ad", "cluster_id": "5941cf9f859d2c0015000021", "created_at": "2001-01-01T02:00:00.000Z", "user_agent": "", "deployment_id": "59de3e8cc9ecc40010324fc6"}`),
		}
		service1EventStart := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000001",
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := testenv.Row{
			"guid":        "00000000-0000-0000-0000-000000000005",
			"created_at":  "2001-01-01T02:00Z",
			"raw_message": json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "compose-db", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "PLAN1", "service_instance_guid": "aaaaaaaa-0000-0000-0000-000000000001", "service_instance_name": "db1", "service_instance_type": "managed_service_instance"}`),
		}

		Expect(db.Insert("service_usage_events", service1EventStart, service1EventStop)).To(Succeed())
		Expect(db.Insert("compose_audit_events", service1EventScale)).To(Succeed())

		Expect(db.Schema.Refresh()).To(Succeed())

		events, err := db.Schema.GetUsageEvents(eventio.EventFilter{
			RangeStart: "2001-01-01",
			RangeStop:  "2001-02-01",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(events).To(HaveLen(2))

		events[0].EventGUID = "unknowable-guid"
		Expect(events[0]).To(Equal(eventio.UsageEvent{
			EventGUID:    "unknowable-guid",
			EventStart:   "2001-01-01T00:00:00+00:00",
			EventStop:    "2001-01-01T01:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
			MemoryInMB:   2048,
			StorageInMB:  4096,
		}))

		Expect(events[1]).To(Equal(eventio.UsageEvent{
			EventGUID:    "00000000-0000-0000-0000-000000000001",
			EventStart:   "2001-01-01T01:00:00+00:00",
			EventStop:    "2001-01-01T02:00:00+00:00",
			ResourceGUID: "aaaaaaaa-0000-0000-0000-000000000001",
			ResourceName: "db1",
			ResourceType: "service",
			OrgGUID:      "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:    "276f4886-ac40-492d-a8cd-b2646637ba76",
			PlanGUID:     "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:     "PLAN1",
			ServiceGUID:  "efadb775-58c4-4e17-8087-6d0f4febc489",
			ServiceName:  "compose-db",
			MemoryInMB:   2048,
			StorageInMB:  4096,
		}))

	})

	/*-----------------------------------------------------------------------------------*
	     2001-01-01                                                      2002-01-01      .
	       00:00           01:00                                           00:00         .
	         |               |                                               |           .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   [======DB1======]   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   .   .   .
	 .   .   |   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   |   :   .   .
	 .   .   |_____________________ request range ___________________________|   .   .   .
	 .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .   .
	*-----------------------------------------------------------------------------------*/
	It("should use the service_name from the historic service/service_plans data if available", func() {
		cfg.AddPlan(eventio.PricingPlan{
			PlanGUID:  "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom: "2001-01-01",
			Name:      "DB_PLAN_1",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 1",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

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

		Expect(db.Insert("services", testenv.Row{
			"guid":                "6c3d1a25-0fbc-45e5-9076-d940390a3bc0",
			"valid_from":          "2000-01-01T00:00:00Z",
			"created_at":          "2000-01-01T00:00:00Z",
			"updated_at":          "2018-06-14T16:32:38Z",
			"label":               "AWESOME_SERVICE_NAME",
			"description":         "the test service service",
			"active":              true,
			"bindable":            true,
			"service_broker_guid": "879d7b06-642d-4bf6-b5e8-1a52451c849a",
		})).To(Succeed())

		Expect(db.Insert("service_plans", testenv.Row{
			"guid":               "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			"valid_from":         "2000-01-01T00:00:00Z",
			"created_at":         "2000-01-01T00:00:00Z",
			"updated_at":         "2018-06-14T16:32:38Z",
			"name":               "AWESOME_SERVICE_PLAN_NAME",
			"description":        "the test service service",
			"service_guid":       "6c3d1a25-0fbc-45e5-9076-d940390a3bc0",
			"service_valid_from": "2000-01-01T00:00:00Z",
			"unique_id":          "c6221308-b7bb-46d2-9d79-a357f5a3837b",
			"active":             true,
			"public":             true,
			"free":               false,
			"extra":              "",
		})).To(Succeed())

		Expect(store.StoreEvents([]eventio.RawEvent{
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
		Expect(usageEvents).To(HaveLen(1))

		Expect(usageEvents[0]).To(Equal(eventio.UsageEvent{
			EventGUID:     "c497eb13-f48a-4859-be53-5569f302b516",
			EventStart:    "2001-01-01T00:00:00+00:00",
			EventStop:     "2001-01-01T01:00:00+00:00",
			ResourceGUID:  "f3f98365-6a95-4bbd-ab8f-527a7957a41f",
			ResourceName:  "DB1",
			ResourceType:  "service",
			OrgGUID:       "51ba75ef-edc0-47ad-a633-a8f6e8770944",
			SpaceGUID:     "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d",
			PlanGUID:      "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			PlanName:      "AWESOME_SERVICE_PLAN_NAME",
			ServiceGUID:   "6c3d1a25-0fbc-45e5-9076-d940390a3bc0",
			ServiceName:   "AWESOME_SERVICE_NAME",
			NumberOfNodes: 0,
			MemoryInMB:    0,
			StorageInMB:   0,
		}))

	})

})
