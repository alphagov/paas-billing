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
				"from_ccy":   "GBP",
				"to_ccy":     "USD",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1.3893831,
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
				"from_ccy":   "GBP",
				"to_ccy":     "USD",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1.3893831,
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				"from_ccy":   "GBP",
				"to_ccy":     "USD",
				"valid_from": "2021-07-01T00:00:00Z",
				"valid_to":   "9999-12-31T23:59:59Z",
				"rate":       1.3893831,
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
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
				// seconds * price / 36000
				"charge_gbp_exc_vat": 2.76381654528,
				"charge_gbp_inc_vat": 3.6482378397696,
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))
	})
})
