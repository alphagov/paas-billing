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

	It("should basically work TODO", func() {
		db, err := testenv.Open(eventstore.Config{})
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		// TODO: truncate existing tables first?
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
				"memory_in_mb":       1,
				"number_of_nodes":    1,
				"external_price":     0.01,
				"component_name":     "test",
				"formula_name":       "test", // should match formula name above
				"vat_code":           "Standard",
				"currency_code":      "GBP",
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
				"memory_in_mb":    1,
				"number_of_nodes": 1,
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
				"space_name":         "test-space",
				"resource_type":      "service",
				"resource_name":      "alex-test-1",
				"component_name":     "test",
				"charge_usd_exc_vat": 0.0, // TODO
				"charge_gbp_exc_vat": 0.0, // TODO
				"charge_gbp_inc_vat": 0.0, // TODO
			},
		}))
	})
})
