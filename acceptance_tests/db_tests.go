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

	It("basic service", func() {
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-31T23:59:59Z')`), // 31 days minus 1 second duration overlap with resource
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
				// Formula from above for this plan: number_of_nodes * time_in_seconds * memory_in_mb / 1024 * 0.01 / 3600 * external_price
				"charge_gbp_exc_vat": 5.354892557191411, // (31*24*60*60−1)*10*1024/1024*0.01/3600*0.1*0.719743892 = 5.354892557
				"charge_gbp_inc_vat": 6.425871068629693, // (31*24*60*60−1)*10*1024/1024*0.01/3600*0.1*0.719743892*1.2 = 6.425871069
				"charge_usd_exc_vat": 7.439997222222222, // (31*24*60*60−1)*10*1024/1024*0.01/3600*0.1 = 7.439997222
				// Calculation comments below this point exclude 10*1024/1024*0.1 terms that cancel.
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-08-01T00:00:00Z')`), // 30 days duration with resource
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
				"charge_gbp_exc_vat": 5.1821560224, // 30*24*60*60*0.01/3600*0.719743892 = 5.182156022
				"charge_gbp_inc_vat": 6.21858722688, // 30*24*60*60*0.01/3600*0.719743892*1.2 = 6.218587227
				"charge_usd_exc_vat": 7.2, // 30*24*60*60*0.01/3600 = 7.2
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`), // 31 days duration with resource
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
				"charge_gbp_exc_vat": 5.35489455648, // 31*24*60*60*0.01/3600*0.719743892 = 5.354894556
				"charge_gbp_inc_vat": 6.425873467776, // 31*24*60*60*0.01/3600*0.719743892*1.2 = 6.425873468
				"charge_usd_exc_vat": 7.44, // 31*24*60*60*0.01/3600 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-08-01T00:00:00Z')`), // 31 days duration with resource
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
				"charge_gbp_exc_vat": 5.35489455648, // 31*24*60*60*0.01/3600*0.719743892 = 5.354894556
				"charge_gbp_inc_vat": 6.425873467776, // 31*24*60*60*0.01/3600*0.719743892*1.2 = 6.425873468
				"charge_usd_exc_vat": 7.44, // 31*24*60*60*0.01/3600 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-15T00:00:00Z')`), // 31 days duration overlap with resource
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
				"charge_gbp_exc_vat": 5.35489455648, // 31*24*60*60*0.01/3600*0.719743892 = 5.354894556
				"charge_gbp_inc_vat": 6.425873467776, // 31*24*60*60*0.01/3600*0.719743892*1.2 = 6.425873468
				"charge_usd_exc_vat": 7.44, // 31*24*60*60*0.01/3600 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-08-15T00:00:00Z')`), // 31 days duration overlap with resource
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
				"charge_gbp_exc_vat": 5.35489455648, // 31*24*60*60*0.01/3600*0.719743892 = 5.354894556
				"charge_gbp_inc_vat": 6.425873467776, // 31*24*60*60*0.01/3600*0.719743892*1.2 = 6.425873468
				"charge_usd_exc_vat": 7.44, // 31*24*60*60*0.01/3600 = 7.44
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-06-15T00:00:00Z', '2021-07-15T00:00:00Z')`), // 14 days duration overlap with resource
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
				"charge_gbp_exc_vat": 2.41833947712, // 14*24*60*60*0.01/3600*0.719743892 = 2.418339477
				"charge_gbp_inc_vat": 2.902007372544, // 14*24*60*60*0.01/3600*0.719743892*1.2 = 2.902007373
				"charge_usd_exc_vat": 3.36, // 14*24*60*60*0.01/3600 = 3.36
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-08-15T00:00:00Z')`), // 17 days duration overlap with resource
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
				"charge_gbp_exc_vat": 2.93655507936, // 17*24*60*60*0.01/3600*0.719743892 = 2.936555079
				"charge_gbp_inc_vat": 3.523866095232, // 17*24*60*60*0.01/3600*0.719743892*1.2 = 3.523866095
				"charge_usd_exc_vat": 4.08, // 17*24*60*60*0.01/3600 = 4.08
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-11T00:00:00Z', '2021-07-28T00:00:00Z')`), // 17 days duration overlap with resource
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
				"charge_gbp_exc_vat": 2.93655507936, // 17*24*60*60*0.01/3600*0.719743892 = 2.936555079
				"charge_gbp_inc_vat": 3.523866095232, // 17*24*60*60*0.01/3600*0.719743892*1.2 = 3.523866095
				"charge_usd_exc_vat": 4.08, // 17*24*60*60*0.01/3600 = 4.08
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`), // 14 days duration overlap with first plan, 17 days overlap with second plan
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
				"charge_gbp_exc_vat": 5.648550064416, // (14×0.1 + 17×0.11)×24×60×60÷36000*0.719743892 = 5.648550064
				"charge_gbp_inc_vat": 6.7782600772992, // (14×0.1 + 17×0.11)×24×60×60÷36000*0.719743892*1.2 = 6.778260077
				"charge_usd_exc_vat": 7.848, // (14×0.1 + 17×0.11)×24×60×60÷36000 = 7.848
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`), // 13 days duration overlap with first plan
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
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60÷36000*0.719743892 = 2.245600943
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60÷36000*0.719743892*1.2 = 2.694721132
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`), // 16 days duration overlap with second plan
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
				"charge_gbp_exc_vat": 3.040198199808, // 16×0.11×24×60×60÷36000*0.719743892 = 3.0401982
				"charge_gbp_inc_vat": 3.6482378397696, // 16×0.11×24×60×60÷36000*0.719743892*1.2 = 3.64823784
				"charge_usd_exc_vat": 4.224, // 16×0.11×24×60×60÷36000 = 4.224
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`), // 13 days duration overlap with first plan
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
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60÷36000*0.719743892 = 2.245600943
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60÷36000*0.719743892*1.2 = 2.694721132
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`), // 16 days duration overlap with second plan
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
				"charge_gbp_exc_vat": 3.040198199808, // 16×0.11×24×60×60÷36000*0.719743892 = 3.0401982
				"charge_gbp_inc_vat": 3.6482378397696, // 16×0.11×24×60×60÷36000*0.719743892*1.2 = 3.64823784
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`), // 14 days duration overlap with first VAT rate, 17 days overlap with second VAT rate
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`), // 13 days duration overlap with first VAT rate
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
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60÷36000×0.719743892 = 2.245600943
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60÷36000×0.719743892×1.2 = 2.694721132
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`), // 16 days duration overlap with second VAT rate
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
				"charge_gbp_exc_vat": 2.76381654528, // 16×0.1×24×60×60÷36000×0.719743892 = 2.763816545
				"charge_gbp_inc_vat": 3.6482378397696, // 16×0.1×24×60×60÷36000×0.719743892×1.32 = 3.64823784
				"charge_usd_exc_vat": 3.84, // 16×0.1×24×60×60÷36000 = 3.84
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`), // 13 days duration overlap with first VAT rate
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
				"charge_gbp_exc_vat": 2.24560094304, // 13×0.1×24×60×60÷36000×0.719743892 = 2.245600943
				"charge_gbp_inc_vat": 2.694721131648, // 13×0.1×24×60×60÷36000×0.719743892×1.2 = 2.694721132
				"charge_usd_exc_vat": 3.12, // 13×0.1×24×60×60÷36000 = 3.12
			},
		}))

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`), // 16 days duration overlap with second VAT rate
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
				"charge_gbp_exc_vat": 2.76381654528, // 16×0.1×24×60×60×0.719743892÷36000 = 2.763816545
				"charge_gbp_inc_vat": 3.6482378397696, // 16×0.1×24×60×60÷36000×0.719743892×1.32 = 3.64823784
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
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":       "Cheap",
				"valid_from":      "2000-01-01T00:00Z",
				"valid_to":        "9999-12-31T23:59:59Z",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"external_price":  0.1,
				"component_name":  "test",
				"formula_name":    "test", // should match formula name above
				"vat_code":        "Standard",
				"currency_code":   "USD", // Has to be USD or we break the USD result field
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-08-01T00:00:00Z')`), // 14 days overlap with first currency exchange rate, 17 days overlap with second currency exchange rate
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-14T00:00:00Z')`), // 13 days overlap with first currency exchange rate
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-16T00:00:00Z', '2021-08-01T00:00:00Z')`), // 16 days overlap with second currency exchange rate
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-02T00:00:00Z', '2021-07-15T00:00:00Z')`), // 13 days overlap with first currency exchange rate
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
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-15T00:00:00Z', '2021-07-31T00:00:00Z')`), // 16 days overlap with second currency exchange rate
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

	It("basic app", func() {
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
				"formula_name":    "app",
				"generic_formula": "(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (external_price / 3600))",
				"formula_source":  "based on paas-cf/config/billing/config-parts/platform_pricing_plans.json.erb",
			})).To(Succeed())

		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"plan_name":          "app",
				"valid_from":         "2019-03-01T00:00Z",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      0,
				"memory_in_mb":       0,
				"number_of_nodes":    0,
				"external_price":     0.01,
				"component_name":     "instance",
				"formula_name":       "app", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())
		Expect(db.Insert("charges",
			testenv.Row{
				"plan_guid":          "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"plan_name":          "app",
				"valid_from":         "2019-03-01T00:00Z",
				"valid_to":           "9999-12-31T23:59:59Z",
				"storage_in_mb":      0,
				"memory_in_mb":       0,
				"number_of_nodes":    0,
				"external_price":     0.01*0.4,
				"component_name":     "platform",
				"formula_name":       "app", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "USD", // Has to be USD or we break the USD result field
			})).To(Succeed())

		Expect(db.Insert("resources",
			testenv.Row{
				"valid_from":      "2021-07-01T00:00:00Z",
				"valid_to":        "2021-08-01T00:00:00Z",
				"resource_guid":   "09582243-ee5a-4d0d-840b-5fde3dd453a8",
				"resource_name":   "alex-test-1",
				"resource_type":   "app",
				"org_guid":        "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"org_name":        "test-org",
				"space_guid":      "8c8afc3b-deb3-4dd0-be91-c2276a56c12f",
				"space_name":      "test-space",
				"plan_name":       "app",
				"plan_guid":       "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"storage_in_mb":   1,
				"memory_in_mb":    1024,
				"number_of_nodes": 10,
				"cf_event_guid":   "2312590b-14c9-47e6-bd34-a04305739c55",
				"last_updated":    "2021-08-03T13:04:00Z",
			})).To(Succeed())

		Expect(
			db.Query(`select * from get_tenant_bill('test-org', '2021-07-01T00:00:00Z', '2021-07-31T23:59:59Z')`), // 31 days minus 1 second duration overlap with resource
		).To(MatchJSON(testenv.Rows{
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "app",
				"plan_guid":          "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"space_name":         "test-space",
				"resource_type":      "app",
				"resource_name":      "alex-test-1",
				"component_name":     "platform",
				"charge_gbp_exc_vat": 21.419570228765643, // (31*24*60*60−1)*10*1024/1024*0.01*0.4/3600*0.719743892 = 21.419570229
				"charge_gbp_inc_vat": 25.70348427451877, // (31*24*60*60−1)*10*1024/1024*0.01*0.4/3600*0.719743892*1.2 = 25.703484275
				"charge_usd_exc_vat": 29.759988888888888, // (31*24*60*60−1)*10*1024/1024*0.01*0.4/3600 = 29.759988889
			},
			{
				"org_name":           "test-org",
				"org_guid":           "c87bd66d-11db-49f7-9b1c-c10a75c71537",
				"plan_name":          "app",
				"plan_guid":          "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"space_name":         "test-space",
				"resource_type":      "app",
				"resource_name":      "alex-test-1",
				"component_name":     "instance",
				"charge_gbp_exc_vat": 53.54892557191411, // (31*24*60*60−1)*10*1024/1024*0.01/3600*0.719743892 = 53.548925572
				"charge_gbp_inc_vat": 64.25871068629693, // (31*24*60*60−1)*10*1024/1024*0.01/3600*0.719743892*1.2 = 64.258710686
				"charge_usd_exc_vat": 74.39997222222222, // (31*24*60*60−1)*10*1024/1024*0.01/3600 = 74.399972222
			},
		}))
	})

	// TODO need to include TASKS, etc here?
	It("Correctly updates resources based on app usage events", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		// TODO: uncomment the below
		//defer db.Close()

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"id":          "1",
				"guid":        "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
				"created_at":  "2021-07-01T00:00:00Z",
				"raw_message": "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"id":          "2",
				"guid":        "b84b96e3-ea99-46d0-9520-76c2f40efff7",
				"created_at":  "2021-07-15T00:00:00Z",
				"raw_message": "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 2, \"previous_state\": \"STARTED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"id":          "3",
				"guid":        "14066ea1-38af-4d0e-af70-ba6cb6b44866",
				"created_at":  "2021-08-01T00:00:00Z",
				"raw_message": "{\"state\": \"STOPPED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"unit-test-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"PENDING\", \"buildpack_guid\": null, \"buildpack_name\": null, \"instance_count\": 2, \"previous_state\": \"STARTED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 2, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
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

	It("Ignores app usage events triggered by tests", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		// TODO: uncomment the below
		//defer db.Close()

		Expect(db.Insert("app_usage_events",
			testenv.Row{
			"id":                "1",
			"guid":              "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
			"created_at":        "2021-07-01T00:00:00Z",
			"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"SMOKE-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
		})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"id":                "2",
				"guid":              "59be2b7f-3554-4f54-8dcb-100b0b248321",
				"created_at":        "2021-07-02T00:00:00Z",
				"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"ACC-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"id":                "3",
				"guid":              "be98110f-8397-48fd-a6aa-77e180ea8a3e",
				"created_at":        "2021-07-03T00:00:00Z",
				"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"CATS-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
			})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
			"id":                "4",
			"guid":              "b2395dc0-4b88-4b3a-a782-21badce9bcb5",
			"created_at":        "2021-07-04T00:00:00Z",
			"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"PERF-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
		})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
			"id":                "5",
			"guid":              "0b695683-c896-443a-8a8b-754cef460163",
			"created_at":        "2021-07-05T00:00:00Z",
			"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"BACC-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
		})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
			"id":                "6",
			"guid":              "3b080789-3a43-4768-859f-fe082177fc62",
			"created_at":        "2021-07-06T00:00:00Z",
			"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"AIVENBACC-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
		})).To(Succeed())

		Expect(db.Insert("app_usage_events",
			testenv.Row{
			"id":                "7",
			"guid":              "8e83f645-1a19-48e4-abb7-29ae71941db7",
			"created_at":        "2021-07-07T00:00:00Z",
			"raw_message":       "{\"state\": \"STARTED\", \"app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"org_guid\": \"428d5022-3ea5-46e9-8220-fc1e80b58de5\", \"task_guid\": null, \"task_name\": null, \"space_guid\": \"c9bbfb98-9429-4c58-a57f-4304ef7f30a2\", \"space_name\": \"ASATS-1-SPACE-1c5968cee02f3899\", \"process_type\": \"web\", \"package_state\": \"STAGED\", \"buildpack_guid\": \"60b5ec15-3db4-4554-8cb0-4be2bcb64526\", \"buildpack_name\": \"binary_buildpack\", \"instance_count\": 1, \"previous_state\": \"STOPPED\", \"parent_app_guid\": \"12a71e81-8cbf-4d46-bfa5-a5d446735f73\", \"parent_app_name\": \"unit-test-APP-c83c773e9daf5af3\", \"previous_package_state\": \"UNKNOWN\", \"previous_instance_count\": 1, \"memory_in_mb_per_instance\": 30, \"previous_memory_in_mb_per_instance\": 30}",
		})).To(Succeed())

		Expect(db.Query(`select * from update_resources('1970-01-01T00:00:00Z')`)).To(MatchJSON(testenv.Rows{
			{
			"num_rows_added": 0,
			},
		}))
	})
	It("Correctly updates resources based on service usage events", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		// TODO: uncomment the below
		//defer db.Close()

		Expect(db.Insert("service_usage_events",
			testenv.Row{
			"id":                "1",
			"guid":              "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
			"created_at":        "2021-07-01T00:00:00Z",
			"raw_message":       "{\"state\": \"CREATED\", \"org_guid\": \"53d03a41-16a7-40a3-b1a9-83298e65f8f5\", \"space_guid\": \"1cfdea3c-af34-4950-8383-5492e7c86c36\", \"space_name\": \"test-SPACE-dc424ba9f99abbf5\", \"service_guid\": \"7ab41178-f5a2-4444-b2f6-71abf43708b2\", \"service_label\": \"postgres\", \"service_plan_guid\": \"5eff64fa-8b41-4e6b-9321-b81624c678ce\", \"service_plan_name\": \"tiny-unencrypted-12\", \"service_broker_guid\": \"5cb906f6-ac20-4393-9210-e783fb4c8f39\", \"service_broker_name\": \"rds-broker\", \"service_instance_guid\": \"c64e2c78-db40-44ae-987c-f2171fe5e42d\", \"service_instance_name\": \"BACC-4-test-db-74b2529697d9ed58\", \"service_instance_type\": \"managed_service_instance\"}",
		})).To(Succeed())

		Expect(db.Insert("service_usage_events",
			testenv.Row{
				"id":                "2",
				"guid":              "b84b96e3-ea99-46d0-9520-76c2f40efff7",
				"created_at":        "2021-07-15T00:00:00Z",
				"raw_message":       "{\"state\": \"DELETED\", \"org_guid\": \"53d03a41-16a7-40a3-b1a9-83298e65f8f5\", \"space_guid\": \"1cfdea3c-af34-4950-8383-5492e7c86c36\", \"space_name\": \"test-4-SPACE-dc424ba9f99abbf5\", \"service_guid\": \"7ab41178-f5a2-4444-b2f6-71abf43708b2\", \"service_label\": \"postgres\", \"service_plan_guid\": \"5eff64fa-8b41-4e6b-9321-b81624c678ce\", \"service_plan_name\": \"tiny-unencrypted-12\", \"service_broker_guid\": \"5cb906f6-ac20-4393-9210-e783fb4c8f39\", \"service_broker_name\": \"rds-broker\", \"service_instance_guid\": \"c64e2c78-db40-44ae-987c-f2171fe5e42d\", \"service_instance_name\": \"BACC-4-test-db-74b2529697d9ed58\", \"service_instance_type\": \"managed_service_instance\"}",
			})).To(Succeed())

		Expect(db.Query(`select * from update_resources('1970-01-01T00:00:00Z')`)).To(MatchJSON(testenv.Rows{
			{
			"num_rows_added": 1,
			},
		}))
		Expect(
			db.Query(`select valid_from,valid_to,resource_guid,resource_name,resource_type,org_guid,org_name,space_guid,space_name,plan_name,plan_guid,storage_in_mb,memory_in_mb,number_of_nodes,cf_event_guid from resources`),
		).To(MatchJSON(testenv.Rows{
			{
				"cf_event_guid": "b6253aa7-ce44-4a2a-a9c2-f26a8c3b2c91",
				"memory_in_mb": nil,
				"number_of_nodes": nil,
				"org_guid": "53d03a41-16a7-40a3-b1a9-83298e65f8f5",
				"org_name": "53d03a41-16a7-40a3-b1a9-83298e65f8f5", // This cannot be consistently looked up for deleted test orgs
				"plan_guid": "d5091c33-2f9d-4b15-82dc-4ad69717fc03",
				"plan_name": "tiny-unencrypted-12",
				"resource_guid": "c64e2c78-db40-44ae-987c-f2171fe5e42d",
				"resource_name": "BACC-4-test-db-74b2529697d9ed58",
				"resource_type": "service",
				"space_guid": "1cfdea3c-af34-4950-8383-5492e7c86c36",
				"space_name": "1cfdea3c-af34-4950-8383-5492e7c86c36",
				"storage_in_mb": nil,
				"valid_from": "2021-07-01T00:00:00+00:00",
				"valid_to": "2021-07-15T00:00:00+00:00",
			},
		}))
	})
})
