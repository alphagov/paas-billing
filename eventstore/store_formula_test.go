package eventstore_test

import (
	"math"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/testenv"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pricing Formulae", func() {
	var insert = func(formula string, out interface{}, ctx SpecContext) error {
		plan := eventio.PricingPlan{
			Name:      "FormulaTestPlan",
			PlanGUID:  uuid.NewV4().String(),
			ValidFrom: "2000-01-01",
			Components: []eventio.PricingPlanComponent{
				{
					Name:         "FormulaTestPlanComponent",
					VATCode:      "Standard",
					CurrencyCode: "GBP",
					Formula:      formula,
				},
			},
		}

		cfg := testenv.BasicConfig
		cfg.AddPlan(plan)
		db, err := testenv.OpenWithContext(cfg, ctx)
		if err != nil {
			return err
		}
		defer db.Close()

		return db.Conn.QueryRow(`
			select
				eval_formula(64, 128, 2, tstzrange(now(), now() + '60 seconds'), formula) as result
			from
				pricing_plan_components
			where
				plan_guid = $1 and valid_from = $2
		`, plan.PlanGUID, plan.ValidFrom).Scan(out)
	}

	It("Should allow basic integer formulae", func(ctx SpecContext) {
		var out int
		err := insert("((2 * 2::integer) + 1 - 1) / 1", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(((2 * 2) + 1 - 1) / 1))
	})

	It("Should allow basic bigint formulae", func(ctx SpecContext) {
		var out int64
		err := insert("12147483647 * (2)::bigint", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(int64(12147483647 * 2)))
	})

	It("Should allow basic numeric formulae", func(ctx SpecContext) {
		var out float64
		err := insert("1.5 * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(float64(1.5 * 2)))
	})

	It("Should allow $time_in_seconds variable", func(ctx SpecContext) {
		var out int
		err := insert("$time_in_seconds * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(60 * 2))
	})

	It("Should not truncate the result of a division of $time_in_seconds", func(ctx SpecContext) {
		var out float64
		err := insert("$time_in_seconds / 3600 * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(60 * 2 / 3600.0))
	})

	It("Should allow $memory_in_mb variable", func(ctx SpecContext) {
		var out int
		err := insert("$memory_in_mb * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(64 * 2))
	})

	It("Should not truncate the result of a division of $memory_in_mb", func(ctx SpecContext) {
		var out float64
		err := insert("$memory_in_mb / 1024 * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(64 / 1024.0 * 2))
	})

	It("Should allow $storage_in_mb variable", func(ctx SpecContext) {
		var out int
		err := insert("$storage_in_mb * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(128 * 2))
	})

	It("Should not truncate the result of a division of $storage_in_mb", func(ctx SpecContext) {
		var out float64
		err := insert("$storage_in_mb / 1024 * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(128 / 1024.0 * 2))
	})

	It("Should allow $number_of_nodes variable", func(ctx SpecContext) {
		var out int
		err := insert("$number_of_nodes * 2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(2 * 2))
	})

	It("Should allow power of operator", func(ctx SpecContext) {
		var out float64
		err := insert("2^2", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(math.Pow(2, 2)))
	})

	It("Should allow ceil function", func(ctx SpecContext) {
		var out float64
		err := insert("ceil(5.0/3.0)", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(BeNumerically("==", 2))
	})

	It("Should allow ceil function with a variable", func(ctx SpecContext) {
		var out float64
		err := insert("ceil($time_in_seconds / 3600) * 10", &out, ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(BeNumerically("==", 10))
	})

	It("Should throw error if ceil is used wrongly", func(ctx SpecContext) {
		var out float64
		err := insert("ceil(5", &out, ctx)
		Expect(err).To(HaveOccurred())
	})

	It("Should not allow `;`", func(ctx SpecContext) {
		var out interface{}
		err := insert("1+1;", &out, ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(MatchRegexp(`illegal token in formula: ;`))
	})

	It("Should not allow `select`", func(ctx SpecContext) {
		var out interface{}
		err := insert("select", &out, ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(MatchRegexp(`illegal token in formula: select`))
	})

	It("Should not allow `$unknown variable`", func(ctx SpecContext) {
		var out interface{}
		err := insert("$unknown", &out, ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(MatchRegexp(`illegal token in formula: \$unknown`))
	})
})
