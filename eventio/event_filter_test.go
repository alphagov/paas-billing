package eventio_test

import (
	. "github.com/alphagov/paas-billing/eventio"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)



var _ = Describe("EventFilter", func() {
	table.DescribeTable(
		"SplitByMonth should return a list of filters split by month",
		func(filter EventFilter, expected []EventFilter) {
			result, err := filter.SplitByMonth()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))

		},
		table.Entry(
			"Empty range should return empty slice",
			EventFilter{RangeStart: "2018-01-01", RangeStop: "2018-01-01"},
			[]EventFilter{},
		),
		table.Entry(
			"Reverse range should return empty slice",
			EventFilter{RangeStart: "9001-01-01", RangeStop: "2018-01-01"},
			[]EventFilter{},
		),
		table.Entry(
			"Less than one month should return the same filter",
			EventFilter{RangeStart: "2018-01-02", RangeStop: "2018-01-05"},
			[]EventFilter{{RangeStart: "2018-01-02T00:00:00Z", RangeStop: "2018-01-05T00:00:00Z"}},
		),
		table.Entry(
			"Range spanning the tail of one month and the whole next month should return both",
			EventFilter{RangeStart: "2018-01-15", RangeStop: "2018-03-01"},
			[]EventFilter{
				{RangeStart: "2018-01-15T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
				{RangeStart: "2018-02-01T00:00:00Z", RangeStop: "2018-03-01T00:00:00Z"},
			},
		),
		table.Entry(
			"Range spanning the whole of one month and the head of the next month should return both",
			EventFilter{RangeStart: "2018-01-01", RangeStop: "2018-02-15"},
			[]EventFilter{
				{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
				{RangeStart: "2018-02-01T00:00:00Z", RangeStop: "2018-02-15T00:00:00Z"},
			},
		),
		table.Entry(
			"Exactly one month should return the same month",
			EventFilter{RangeStart: "2018-01-01", RangeStop: "2018-02-01"},
			[]EventFilter{{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"}},
		),
		table.Entry(
			"Exactly one month across two months should return the tail of the first and the head of the second ",
			EventFilter{RangeStart: "2018-01-05", RangeStop: "2018-02-05"},
			[]EventFilter{
				{RangeStart: "2018-01-05T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
				{RangeStart: "2018-02-01T00:00:00Z", RangeStop: "2018-02-05T00:00:00Z"},
			},
		),
		table.Entry(
			"Two month range should return two months",
			EventFilter{RangeStart: "2017-12-01", RangeStop: "2018-02-01"},
			[]EventFilter{
				{RangeStart: "2017-12-01T00:00:00Z", RangeStop: "2018-01-01T00:00:00Z"},
				{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
			},
		),
		table.Entry(
			"Should maintain org guids",
			EventFilter{
				RangeStart: "2017-01-15",
				RangeStop:  "2017-03-15",
				OrgGUIDs:   []string{"some-guid", "some-other-guid"},
			},
			[]EventFilter{
				{
					RangeStart: "2017-01-15T00:00:00Z",
					RangeStop:  "2017-02-01T00:00:00Z",
					OrgGUIDs:   []string{"some-guid", "some-other-guid"},
				},
				{
					RangeStart: "2017-02-01T00:00:00Z",
					RangeStop:  "2017-03-01T00:00:00Z",
					OrgGUIDs:   []string{"some-guid", "some-other-guid"},
				},
				{
					RangeStart: "2017-03-01T00:00:00Z",
					RangeStop:  "2017-03-15T00:00:00Z",
					OrgGUIDs:   []string{"some-guid", "some-other-guid"},
				},
			},
		),
		table.Entry(
			"Multi-year range should return all months",
			EventFilter{RangeStart: "2016-11-12", RangeStop: "2018-01-05"},
			[]EventFilter{
				{RangeStart: "2016-11-12T00:00:00Z", RangeStop: "2016-12-01T00:00:00Z"},
				{RangeStart: "2016-12-01T00:00:00Z", RangeStop: "2017-01-01T00:00:00Z"},
				{RangeStart: "2017-01-01T00:00:00Z", RangeStop: "2017-02-01T00:00:00Z"},
				{RangeStart: "2017-02-01T00:00:00Z", RangeStop: "2017-03-01T00:00:00Z"},
				{RangeStart: "2017-03-01T00:00:00Z", RangeStop: "2017-04-01T00:00:00Z"},
				{RangeStart: "2017-04-01T00:00:00Z", RangeStop: "2017-05-01T00:00:00Z"},
				{RangeStart: "2017-05-01T00:00:00Z", RangeStop: "2017-06-01T00:00:00Z"},
				{RangeStart: "2017-06-01T00:00:00Z", RangeStop: "2017-07-01T00:00:00Z"},
				{RangeStart: "2017-07-01T00:00:00Z", RangeStop: "2017-08-01T00:00:00Z"},
				{RangeStart: "2017-08-01T00:00:00Z", RangeStop: "2017-09-01T00:00:00Z"},
				{RangeStart: "2017-09-01T00:00:00Z", RangeStop: "2017-10-01T00:00:00Z"},
				{RangeStart: "2017-10-01T00:00:00Z", RangeStop: "2017-11-01T00:00:00Z"},
				{RangeStart: "2017-11-01T00:00:00Z", RangeStop: "2017-12-01T00:00:00Z"},
				{RangeStart: "2017-12-01T00:00:00Z", RangeStop: "2018-01-01T00:00:00Z"},
				{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-01-05T00:00:00Z"},
			},
		),
	)

	table.DescribeTable(
		"EventFilter SplitByMonth should cope with multiple date formats",
		func(filter EventFilter, expected []EventFilter) {
			result, err := filter.SplitByMonth()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		table.Entry(
			"Chop off any non-date bits",
			EventFilter{RangeStart: "2018-01-01T10:10", RangeStop: "2018-02-01T10:11"},
			[]EventFilter{
				{RangeStart: "2018-01-01T10:10:00Z", RangeStop: "2018-02-01T00:00:00Z"},
				{RangeStart: "2018-02-01T00:00:00Z", RangeStop: "2018-02-01T10:11:00Z"},
			},
		),
	)

	table.DescribeTable(
		"TruncateMonth truncate to one month both start and end",
		func(filter EventFilter, expected EventFilter) {
			result, err := filter.TruncateMonth()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))

		},
		table.Entry(
			"1st-1st",
			EventFilter{RangeStart: "2018-01-01", RangeStop: "2018-02-01"},
			EventFilter{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
		),
		table.Entry(
			"1st-Xth",
			EventFilter{RangeStart: "2018-01-01", RangeStop: "2018-02-15"},
			EventFilter{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
		),
		table.Entry(
			"Xth-1st",
			EventFilter{RangeStart: "2018-01-15", RangeStop: "2018-02-01"},
			EventFilter{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
		),
		table.Entry(
			"Xth-Xth",
			EventFilter{RangeStart: "2018-01-15", RangeStop: "2018-02-15"},
			EventFilter{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z"},
		),
		table.Entry(
			"Perserves orgs",
			EventFilter{RangeStart: "2018-01-15", RangeStop: "2018-02-15", OrgGUIDs: []string{"org-guid"}},
			EventFilter{RangeStart: "2018-01-01T00:00:00Z", RangeStop: "2018-02-01T00:00:00Z", OrgGUIDs: []string{"org-guid"}},
		),
	)
})
