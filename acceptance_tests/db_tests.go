package acceptance_tests

import (
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BillingSQLFunctions", func() {
	It("should be idempotent", func() {
		db, err := testenv.Open(testenv.BasicConfig)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		Expect(db.Schema.Init()).To(Succeed())
		Expect(db.Schema.Init()).To(Succeed())
	})

	It("should basically work", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "GBP",
				"to_ccy":     "GBP",
				"valid_from": "0800-01-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1,
   			})).To(Succeed())

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "USD",
				"to_ccy":     "GBP",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       0.719743892,
   			})).To(Succeed())

		Expect(db.Insert("vat_rates_new",
			testenv.Row{
				"vat_code":   "Standard",
				"valid_from": "2011-01-04T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"vat_rate":   0.2,
   			})).To(Succeed())

		Expect(db.Insert("billing_formulae",
			testenv.Row{
				"formula_name":    "test",
				"generic_formula": "(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * external_price",
				"formula_source":  "imagination",
  			})).To(Succeed())

		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":          "Cheap",
				"valid_from":         "2000-01-01T00:00Z",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      1,
				"memory_in_mb":       1024,
				"number_of_nodes":    10,
				"external_price":     0.1,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())

		Expect(db.Insert("resources",
			testenv.Row{
				"valid_from":      "2021-07-01T00:00:00Z",
				"valid_to":        "2021-08-01T00:00:00Z",
				"resource_guid":   "09582243-ee5a-4d0d-840b-5fde3dd453a8",
				"resource_name":   "alex-test-1",
				"resource_type":   "service",
				"org_guid":        "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"org_name":        "test-org",
				"space_guid":      "8c8afc3b-deb3-4dd0-be91-c2276a56c12f",
				"space_name":      "test-space",
				"plan_name":       "Cheap",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"cf_event_guid":   "2312590b-14c9-47e6-bd34-a04305739c55",
				"last_updated":    "2021-08-03T13:04:00Z",
			})).To(Succeed())

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-31T23:59:59Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.354892557191411,
				"charge_gbp_inc_vat": 6.425871068629693,
				"charge_usd_exc_vat": 7.439997222222222,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.1821560224,
				"charge_gbp_inc_vat": 6.21858722688,
				"charge_usd_exc_vat": 7.2,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.35489455648,
				"charge_gbp_inc_vat": 6.425873467776,
				"charge_usd_exc_vat": 7.44,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.35489455648,
				"charge_gbp_inc_vat": 6.425873467776,
				"charge_usd_exc_vat": 7.44,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.35489455648,
				"charge_gbp_inc_vat": 6.425873467776,
				"charge_usd_exc_vat": 7.44,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-08-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.35489455648,
				"charge_gbp_inc_vat": 6.425873467776,
				"charge_usd_exc_vat": 7.44,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-07-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.41833947712,
				"charge_gbp_inc_vat": 2.902007372544,
				"charge_usd_exc_vat": 3.36,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-08-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.93655507936,
				"charge_gbp_inc_vat": 3.523866095232,
				"charge_usd_exc_vat": 4.08,
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-11T00:00:00Z', '2021-07-28T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.93655507936,
				"charge_gbp_inc_vat": 3.523866095232,
				"charge_usd_exc_vat": 4.08,
			},
		}))
	})

	It("should handle a mid-month plan change", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "GBP",
				"to_ccy":     "GBP",
				"valid_from": "0800-01-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1,
   			})).To(Succeed())

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "USD",
				"to_ccy":     "GBP",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       0.719743892,
   			})).To(Succeed())

		Expect(db.Insert("vat_rates_new",
			testenv.Row{
				"vat_code":   "Standard",
				"valid_from": "2011-01-04T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"vat_rate":   0.2,
   			})).To(Succeed())

		Expect(db.Insert("billing_formulae",
			testenv.Row{
				"formula_name":    "test",
				"generic_formula": "(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * external_price",
				"formula_source":  "imagination",
  			})).To(Succeed())

		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":          "Cheap",
				"valid_from":         "2000-01-01T00:00Z",
				"valid_to":           "2021-07-15T00:00:00Z",
				"storage_in_mb":      1,
				"memory_in_mb":       1024,
				"number_of_nodes":    10,
				"external_price":     0.1,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())
		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":          "Cheap",
				"valid_from":         "2021-07-15T00:00:00",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      1,
				"memory_in_mb":       1024,
				"number_of_nodes":    10,
				"external_price":     0.11,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())

		Expect(db.Insert("resources",
			testenv.Row{
				"valid_from":      "2021-07-01T00:00:00Z",
				"valid_to":        "2021-08-01T00:00:00Z",
				"resource_guid":   "09582243-ee5a-4d0d-840b-5fde3dd453a8",
				"resource_name":   "alex-test-1",
				"resource_type":   "service",
				"org_guid":        "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"org_name":        "test-org",
				"space_guid":      "8c8afc3b-deb3-4dd0-be91-c2276a56c12f",
				"space_name":      "test-space",
				"plan_name":       "Cheap",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"cf_event_guid":   "2312590b-14c9-47e6-bd34-a04305739c55",
				"last_updated":    "2021-08-03T13:04:00Z",
			})).To(Succeed())

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.648550064416,
				"charge_gbp_inc_vat": 6.7782600772992,
				"charge_usd_exc_vat": 7.848, // 14×0.1×24×60×60÷36000 + 17×0.11×24×60×60÷36000 = 7.848
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304,
				"charge_gbp_inc_vat": 2.694721131648,
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 3.040198199808,
				"charge_gbp_inc_vat": 3.6482378397696,
				"charge_usd_exc_vat": 4.224, // 16×0.11×24×60×60÷36000 = 4.224
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304,
				"charge_gbp_inc_vat": 2.694721131648,
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 3.040198199808,
				"charge_gbp_inc_vat": 3.6482378397696,
				"charge_usd_exc_vat": 4.224, // 16×0.11×24×60×60÷36000 = 4.224
			},
		}))
	})

	It("should handle a mid-month VAT rate change", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "GBP",
				"to_ccy":     "GBP",
				"valid_from": "0800-01-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1,
   			})).To(Succeed())

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "USD",
				"to_ccy":     "GBP",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       0.719743892,
   			})).To(Succeed())

		Expect(db.Insert("vat_rates_new",
			testenv.Row{
				"vat_code":   "Standard",
				"valid_from": "2011-01-04T00:00:00Z",
				"valid_to":   "2021-07-15T00:00:00Z",
				"vat_rate":   0.2,
   			})).To(Succeed())
		Expect(db.Insert("vat_rates_new",
			testenv.Row{
				"vat_code":   "Standard",
				"valid_from": "2021-07-15T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"vat_rate":   0.32,
   			})).To(Succeed())

		Expect(db.Insert("billing_formulae",
			testenv.Row{
				"formula_name":    "test",
				"generic_formula": "(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * external_price",
				"formula_source":  "imagination",
  			})).To(Succeed())

		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":          "Cheap",
				"valid_from":         "2000-01-01T00:00Z",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      1,
				"memory_in_mb":       1024,
				"number_of_nodes":    10,
				"external_price":     0.1,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())

		Expect(db.Insert("resources",
			testenv.Row{
				"valid_from":      "2021-07-01T00:00:00Z",
				"valid_to":        "2021-08-01T00:00:00Z",
				"resource_guid":   "09582243-ee5a-4d0d-840b-5fde3dd453a8",
				"resource_name":   "alex-test-1",
				"resource_type":   "service",
				"org_guid":        "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"org_name":        "test-org",
				"space_guid":      "8c8afc3b-deb3-4dd0-be91-c2276a56c12f",
				"space_name":      "test-space",
				"plan_name":       "Cheap",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"cf_event_guid":   "2312590b-14c9-47e6-bd34-a04305739c55",
				"last_updated":    "2021-08-03T13:04:00Z",
			})).To(Succeed())

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.35489455648, // 31×0.1×24×60×60÷36000×0.719743892 = 5.354894556
				"charge_gbp_inc_vat": 6.7782600772992, // 14×0.1×24×60×60÷36000×1.2×0.719743892 + 17×0.1×24×60×60÷36000×1.32×0.719743892 = 6.778260077
				"charge_usd_exc_vat": 7.44, // 31×0.1×24×60×60÷36000 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304,
				"charge_gbp_inc_vat": 2.694721131648,
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.76381654528,
				"charge_gbp_inc_vat": 3.6482378397696,
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304,
				"charge_gbp_inc_vat": 2.694721131648,
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.76381654528, // 16×0.1×24×60×60×0.719743892÷36000 = 2.76381654528
				"charge_gbp_inc_vat": 3.6482378397696,
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))
	})

	It("should handle a mid-month currency exchange rate change", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "GBP",
				"to_ccy":     "GBP",
				"valid_from": "0800-01-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1,
   			})).To(Succeed())

		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "USD",
				"to_ccy":     "GBP",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "2021-07-15T00:00:00Z",
				"rate":       0.719743892,
   			})).To(Succeed())
		Expect(db.Insert("currency_exchange_rates",
			testenv.Row{
				"from_ccy":   "USD",
				"to_ccy":     "GBP",
				"valid_from": "2021-07-15T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       0.8,
   			})).To(Succeed())

		Expect(db.Insert("vat_rates_new",
			testenv.Row{
				"vat_code":   "Standard",
				"valid_from": "2011-01-04T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"vat_rate":   0.2,
   			})).To(Succeed())

		Expect(db.Insert("billing_formulae",
			testenv.Row{
				"formula_name":    "test",
				"generic_formula": "(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * external_price",
				"formula_source":  "imagination",
  			})).To(Succeed())

		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":          "Cheap",
				"valid_from":         "2000-01-01T00:00Z",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      1,
				"memory_in_mb":       1024,
				"number_of_nodes":    10,
				"external_price":     0.1,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())

		Expect(db.Insert("resources",
			testenv.Row{
				"valid_from":      "2021-07-01T00:00:00Z",
				"valid_to":        "2021-08-01T00:00:00Z",
				"resource_guid":   "09582243-ee5a-4d0d-840b-5fde3dd453a8",
				"resource_name":   "alex-test-1",
				"resource_type":   "service",
				"org_guid":        "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"org_name":        "test-org",
				"space_guid":      "8c8afc3b-deb3-4dd0-be91-c2276a56c12f",
				"space_name":      "test-space",
				"plan_name":       "Cheap",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"cf_event_guid":   "2312590b-14c9-47e6-bd34-a04305739c55",
				"last_updated":    "2021-08-03T13:04:00Z",
			})).To(Succeed())

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 5.68233947712, // 14×0.1×24×60×60÷36000×0.719743892 + 17×0.1×24×60×60÷36000×0.8 = 5.68233947712
				"charge_gbp_inc_vat": 6.818807372544, // 14×0.1×24×60×60÷36000×1.2×0.719743892 + 17×0.1×24×60×60÷36000×1.2×0.8 = 6.818807373
				"charge_usd_exc_vat": 7.44, // 31×0.1×24×60×60÷36000 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60×0.719743892÷36000 = 2.24560094304
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60×1.2×0.719743892÷36000 = 2.694721131648
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 3.072, // 16×0.1×24×60×60×0.8÷36000 = 3.072
				"charge_gbp_inc_vat": 3.6864, // 16×0.1×24×60×60×0.8×1.2÷36000 = 3.6864
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60×0.719743892÷36000 = 2.24560094304
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60×1.2×0.719743892÷36000 = 2.694721131648
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`),
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "Cheap",
				"plan_guid":          "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_gbp_exc_vat": 3.072, // 16×0.1×24×60×60×0.8÷36000 = 3.072
				"charge_gbp_inc_vat": 3.6864, // 16×0.1×24×60×60×0.8×1.2÷36000 = 3.6864
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))
	})
	It("Correctly updates resources based on app usage events", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
    // TODO: uncomment the below
    //defer db.Close()

    Expect(db.Insert("app_usage_events",
			testenv.Row{
        "id":                "1",
        "guid":              "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
				"created_at":        "2021-07-01T00:00:00Z",
				"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

    Expect(db.Insert("app_usage_events",
			testenv.Row{
        "id":                "2",
        "guid":              "b84b96e3-ea99-46d0-9520-76c2f40efff7",
				"created_at":        "2021-07-15T00:00:00Z",
				"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 2, \"previous_state\": \"STARTED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

    Expect(db.Insert("app_usage_events",
			testenv.Row{
        "id":                "3",
        "guid":              "14066ea1-38af-4d0e-af70-ba6cb6b44866",
				"created_at":        "2021-08-01T00:00:00Z",
				"raw_message":       "{\"state\": \"STOPPED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"PENDING\", \"buildpack_guid\": null, \"buildpack_name\": null, \"instance_count\": 2, \"previous_state\": \"STARTED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 2, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

    Expect(db.Query(`select * from update_resources('1970-01-01T00:00:00Z')`)).To(MatchJSON(testenv.Rows{
        {
        "num_rows_added": 2,
        },
      }))
		Expect(
			db.Query(`select valid_from,valid_to,resource_guid,resource_name,resource_type,org_guid,org_name,space_guid,space_name,plan_name,plan_guid,storage_in_mb,memory_in_mb,number_of_nodes,cf_event_guid from resources`),
		).To(MatchJSON(testenv.Rows{
			{
        "valid_from":      "2021-07-01T00:00:00+00:00",
        "valid_to":        "2021-07-15T00:00:00+00:00",
        "resource_guid":   "12a71e81-8cbf-4d46-bfa5-a5d446735f73",
        "resource_name":   "unit-test-APP-c83c773e9daf5af3",
        "resource_type":   "app",
        "org_guid":        "428d5022-3ea5-46e9-8220-fc1e80b58de5",
        "org_name":        "428d5022-3ea5-46e9-8220-fc1e80b58de5",
        "space_guid":      "c9bbfb98-9429-4c58-a57f-4304ef7f30a2",
        "space_name":      "c9bbfb98-9429-4c58-a57f-4304ef7f30a2",
        "plan_name":       "app",
        "plan_guid":       "f4d4b95a-f55e-4593-8d54-3364c25798c4",
        "storage_in_mb":   0,
        "memory_in_mb":    30,
        "number_of_nodes": 1,
        "cf_event_guid":   "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
			},
      {
        "valid_from":      "2021-07-15T00:00:00+00:00",
        "valid_to":        "2021-08-01T00:00:00+00:00",
        "resource_guid":   "12a71e81-8cbf-4d46-bfa5-a5d446735f73",
        "resource_name":   "unit-test-APP-c83c773e9daf5af3",
        "resource_type":   "app",
        "org_guid":        "428d5022-3ea5-46e9-8220-fc1e80b58de5",
        "org_name":        "428d5022-3ea5-46e9-8220-fc1e80b58de5",
        "space_guid":      "c9bbfb98-9429-4c58-a57f-4304ef7f30a2",
        "space_name":      "c9bbfb98-9429-4c58-a57f-4304ef7f30a2",
        "plan_name":       "app",
        "plan_guid":       "f4d4b95a-f55e-4593-8d54-3364c25798c4",
        "storage_in_mb":   0,
        "memory_in_mb":    30,
        "number_of_nodes": 2,
        "cf_event_guid":   "b84b96e3-ea99-46d0-9520-76c2f40efff7",
      },
		}))
	})
})
