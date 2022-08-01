package eventstore_test

import (
	"encoding/json"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TotalCostEvents", func() {
	var (
		cfg eventstore.Config
	)
	BeforeEach(func() {
		cfg = testenv.BasicConfig
	})
	It("should return the cost for each plan_guid", func() {
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
			PlanGUID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
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
		app1EventStart := testenv.Row{
			"guid":        "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		app1EventStop := testenv.Row{
			"guid":        "8d9036c5-8367-497d-bb56-94bfcac6621a",
			"created_at":  "2001-02-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "STOPPED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		service1EventStart := testenv.Row{
			"guid":        "c497eb13-f48a-4859-be53-5569f302b516",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := testenv.Row{
			"guid":        "6d52b4f4-9e33-4504-8fca-fd9e33af11a6",
			"created_at":  "2001-03-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}
		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		Expect(db.Insert("services",
			testenv.Row{
				"label":               "postgres",
				"guid":                "efadb775-58c4-4e17-8087-6d0f4febc489",
				"valid_from":          "2000-01-01T00:00Z",
				"created_at":          "2000-01-01T00:00Z",
				"updated_at":          "2000-01-01T00:00Z",
				"description":         "",
				"service_broker_guid": "efadb775-58c4-4e17-8087-6d0f4febc481",
				"active":              true,
				"bindable":            true,
			})).To(Succeed())

		Expect(db.Insert("service_plans",
			testenv.Row{
				"unique_id":          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				"name":               "Free",
				"guid":               "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"valid_from":         "2000-01-01T00:00Z",
				"created_at":         "2000-01-01T00:00Z",
				"updated_at":         "2000-01-01T00:00Z",
				"description":        "",
				"service_guid":       "efadb775-58c4-4e17-8087-6d0f4febc489",
				"service_valid_from": "2000-01-01T00:00Z",
				"active":             true,
				"public":             true,
				"free":               true,
				"extra":              "",
			})).To(Succeed())

		Expect(db.Insert("app_usage_events", app1EventStart, app1EventStop)).To(Succeed())
		Expect(db.Insert("service_usage_events", service1EventStart, service1EventStop)).To(Succeed())
		Expect(db.Schema.Refresh()).To(Succeed())
		store := db.Schema
		outputEvents, err := store.GetTotalCost()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(outputEvents)).To(Equal(2))
		Expect(outputEvents[0]).To(Equal(eventio.TotalCost{
			PlanGUID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Cost:     1417,
		}))
		Expect(outputEvents[1]).To(Equal(eventio.TotalCost{
			PlanGUID: "f4d4b95a-f55e-4593-8d54-3364c25798c4",
			Cost:     7.45,
		}))
	})
})
