package schema_test

import (
	"encoding/json"

	"github.com/alphagov/paas-billing/schema"
	"github.com/alphagov/paas-billing/testenv"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Init", func() {

	var (
		cfg schema.Config
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

	It("should normalize *_usage_events tables into a consistant format with durations", func() {
		cfg.AddPlan(schema.PricingPlan{
			PlanGUID:  schema.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []schema.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(schema.PricingPlan{
			PlanGUID:  "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom: "2001-01-01",
			Name:      "DB_PLAN_1",
			Components: []schema.PricingPlanComponent{
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
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "STOPPED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		service1EventStart := testenv.Row{
			"guid":        "c497eb13-f48a-4859-be53-5569f302b516",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := testenv.Row{
			"guid":        "6d52b4f4-9e33-4504-8fca-fd9e33af11a6",
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("app_usage_events", app1EventStart, app1EventStop)).To(Succeed())
		Expect(db.Insert("service_usage_events", service1EventStart, service1EventStop)).To(Succeed())
		Expect(db.Schema.Refresh()).To(Succeed())

		Expect(
			db.Query(`select * from events`),
		).To(MatchJSON(testenv.Rows{
			{
				"duration":        "[\"2001-01-01 00:00:00+00\",\"2001-01-01 01:00:00+00\")",
				"event_guid":      "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
				"memory_in_mb":    1024,
				"number_of_nodes": 1,
				"org_guid":        "51ba75ef-edc0-47ad-a633-a8f6e8770944",
				"plan_guid":       "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"plan_name":       "app",
				"resource_guid":   "c85e98f0-6d1b-4f45-9368-ea58263165a0",
				"resource_name":   "APP1",
				"resource_type":   "app",
				"space_guid":      "276f4886-ac40-492d-a8cd-b2646637ba76",
				"storage_in_mb":   0,
			},
			{
				"duration":        "[\"2001-01-01 00:00:00+00\",\"2001-01-01 01:00:00+00\")",
				"event_guid":      "c497eb13-f48a-4859-be53-5569f302b516",
				"memory_in_mb":    nil,
				"number_of_nodes": nil,
				"org_guid":        "51ba75ef-edc0-47ad-a633-a8f6e8770944",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":       "Free",
				"resource_guid":   "f3f98365-6a95-4bbd-ab8f-527a7957a41f",
				"resource_name":   "ja-rails-postgres",
				"resource_type":   "postgres",
				"space_guid":      "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d",
				"storage_in_mb":   nil,
			},
		}))
	})

	It("only outputs a single resource row because the others have zero duration", func() {
		cfg.AddPlan(schema.PricingPlan{
			PlanGUID:  schema.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []schema.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"guid":        "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
				"created_at":  "2001-01-01T00:00Z",
				"raw_message": json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
			},
		)).To(Succeed())

		Expect(db.Schema.Refresh()).To(Succeed())

		Expect(db.Get(`SELECT COUNT(*) FROM billable_event_components`)).To(BeNumerically("==", 1))
	})

	It("should ensure plan has unique plan_guid + valid_from", func() {
		cfg.AddPlan(schema.PricingPlan{
			PlanGUID:  schema.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []schema.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(schema.PricingPlan{
			PlanGUID:  schema.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []schema.PricingPlanComponent{
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
			db, err := testenv.Open(schema.Config{
				PricingPlans: []schema.PricingPlan{
					{
						PlanGUID:  uuid.NewV4().String(),
						ValidFrom: timestamp,
						Name:      "bad-plan",
						Components: []schema.PricingPlanComponent{
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
			db, err := testenv.Open(schema.Config{
				VATRates: []schema.VATRate{
					{
						ValidFrom: timestamp,
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

	DescribeTable("reject currency_rates with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(schema.Config{
				CurrencyRates: []schema.CurrencyRate{
					{
						ValidFrom: timestamp,
						Code:      "USD",
						Rate:      0.8,
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

	DescribeTable("allow whitelisted currency codes",
		func(code string) {
			db, err := testenv.Open(schema.Config{
				CurrencyRates: []schema.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
						Code:      code,
						Rate:      0.8,
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
			db, err := testenv.Open(schema.Config{
				CurrencyRates: []schema.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
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
			db, err := testenv.Open(schema.Config{
				VATRates: []schema.VATRate{
					{
						ValidFrom: "2001-01-01",
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
			db, err := testenv.Open(schema.Config{
				VATRates: []schema.VATRate{
					{
						ValidFrom: "2001-01-01",
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

})
