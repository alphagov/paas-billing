package api_test

import (
	"time"

	"github.com/alphagov/paas-billing/db"
	. "github.com/alphagov/paas-billing/fixtures"
)

func ago(t time.Duration) time.Time {
	return now.Add(0 - t)
}

func monthsAgo(months int) time.Time {
	return now.AddDate(0, -months, 0)
}

var planFixtures = Plans{
	{
		ID:        10,
		Name:      "ComputePlanA",
		PlanGuid:  db.ComputePlanGuid,
		ValidFrom: monthsAgo(3),
		Components: []PricingPlanComponent{
			{
				ID:        101,
				Name:      "ComputePlanA/1",
				Formula:   "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
				VATRateID: 1,
			},
			{
				ID:        102,
				Name:      "ComputePlanA/2",
				Formula:   "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.3",
				VATRateID: 1,
			},
		},
	},
	{
		ID:        11,
		Name:      "ComputePlanB",
		PlanGuid:  db.ComputePlanGuid,
		ValidFrom: monthsAgo(1),
		Components: []PricingPlanComponent{
			{
				ID:        111,
				Name:      "ComputePlanB/1",
				Formula:   "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
				VATRateID: 1,
			},
		},
	},
	{
		ID:        20,
		Name:      "ServicePlanA",
		PlanGuid:  "00000000-0000-0000-0000-100000000000",
		ValidFrom: monthsAgo(3),
		Components: []PricingPlanComponent{
			{
				ID:        201,
				Name:      "ServicePlanA/1",
				Formula:   "($time_in_seconds / 60 / 60) * 0.2",
				VATRateID: 1,
			},
			{
				ID:        202,
				Name:      "ServicePlanA/2",
				Formula:   "($time_in_seconds / 60 / 60) * 0.3",
				VATRateID: 1,
			},
		},
	},
	{
		ID:        30,
		Name:      "ServicePlanB",
		PlanGuid:  "00000000-0000-0000-0000-200000000000",
		ValidFrom: monthsAgo(3),
		Components: []PricingPlanComponent{
			{
				ID:        301,
				Name:      "ServicePlanB/1",
				Formula:   "($time_in_seconds / 60 / 60) * 1",
				VATRateID: 1,
			},
		},
	},
	{
		ID:        40,
		Name:      "VATTest",
		PlanGuid:  "00000000-0000-0000-0000-300000000000",
		ValidFrom: monthsAgo(3),
		Components: []PricingPlanComponent{
			{
				ID:        401,
				Name:      "With standard VAT",
				Formula:   "($time_in_seconds / 60 / 60) * 0.2",
				VATRateID: 1,
			},
			{
				ID:        402,
				Name:      "With zero VAT",
				Formula:   "($time_in_seconds / 60 / 60) * 0.3",
				VATRateID: 2,
			},
		},
	},
}

// ORG1
//   SPACE1
//     APP1                    START
//   SPACE2
//     SERVICE1                CREATED
//     APP2                    START,START
// ORG2
//   SPACE3
//     SERVICE2                CREATED,DELETED       With different but overlapping timeframe to SERVICE3, and different plan
//     SERVICE3                CREATED,DELETED
//     APP3                    START,STOP
//     APP4                    START,START,STOP,START
//   SPACE4
//     SERVICE4                CREATED
// ORG3
//   SPACE5
//     SERVICE5                CREATED               For testing different VAT rates

var orgsFixtures = Orgs{
	"00000001-0000-0000-0000-000000000000": Org{
		"00000001-0001-0000-0000-000000000000": Space{
			AppEvents: []AppEvent{
				{
					AppGuid:               "00000001-0001-0001-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         1,
					MemoryInMBPerInstance: 64,
					Time: monthsAgo(3),
				},
			},
		},
		"00000001-0002-0000-0000-000000000000": Space{
			AppEvents: []AppEvent{
				{
					AppGuid:               "00000001-0002-0001-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         3,
					MemoryInMBPerInstance: 512,
					Time: ago(2 * time.Hour),
				},
				{
					AppGuid:               "00000001-0002-0001-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         2,
					MemoryInMBPerInstance: 1024,
					Time: ago(1 * time.Hour),
				},
			},
			ServiceEvents: []ServiceEvent{
				{
					ServiceInstanceGuid: "00000001-0002-0002-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
					Time:                ago(1 * time.Hour),
				},
			},
		},
	},
	"00000002-0000-0000-0000-000000000000": Org{
		"00000002-0001-0000-0000-000000000000": Space{
			AppEvents: []AppEvent{
				{
					AppGuid:               "00000002-0001-0001-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         1,
					MemoryInMBPerInstance: 64,
					Time: ago(4 * time.Hour),
				},
				{
					AppGuid:               "00000002-0001-0002-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         2,
					MemoryInMBPerInstance: 256,
					Time: ago(4 * time.Hour),
				},
				{
					AppGuid: "00000002-0001-0002-0000-000000000000",
					State:   "STOPPED",
					Time:    ago(3 * time.Hour),
				},
				{
					AppGuid:               "00000002-0001-0002-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         3,
					MemoryInMBPerInstance: 128,
					Time: ago(2 * time.Hour),
				},
				{
					AppGuid: "00000002-0001-0001-0000-000000000000",
					State:   "STOPPED",
					Time:    ago(1 * time.Hour),
				},
			},
			ServiceEvents: []ServiceEvent{
				{
					ServiceInstanceGuid: "00000002-0001-0003-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
					Time:                ago(4 * time.Hour),
				},
				{
					ServiceInstanceGuid: "00000002-0001-0004-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-200000000000",
					Time:                ago(3 * time.Hour),
				},
				{
					ServiceInstanceGuid: "00000002-0001-0003-0000-000000000000",
					State:               "DELETED",
					Time:                ago(2 * time.Hour),
				},
				{
					ServiceInstanceGuid: "00000002-0001-0004-0000-000000000000",
					State:               "DELETED",
					Time:                ago(1 * time.Hour),
				},
			},
		},
		"00000002-0002-0000-0000-000000000000": Space{
			ServiceEvents: []ServiceEvent{
				{
					ServiceInstanceGuid: "00000002-0002-0001-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
					Time:                ago(1 * time.Hour),
				},
			},
		},
		"00000002-0003-0000-0000-000000000000": Space{
			ServiceEvents: []ServiceEvent{
				{
					ServiceInstanceGuid: "00000002-0003-0001-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-100000000000",
					Time:                ago(5 * time.Hour),
				},
				{
					ServiceInstanceGuid: "00000002-0003-0001-0000-000000000000",
					State:               "UPDATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-200000000000",
					Time:                ago(2 * time.Hour),
				},
				{
					ServiceInstanceGuid: "00000002-0003-0001-0000-000000000000",
					State:               "DELETED",
					Time:                ago(1 * time.Hour),
				},
			},
			AppEvents: []AppEvent{
				{
					AppGuid:               "00000002-0003-0002-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         1,
					MemoryInMBPerInstance: 2,
					Time: ago(5 * time.Hour),
				},
				{
					AppGuid:               "00000002-0003-0002-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         2,
					MemoryInMBPerInstance: 1,
					Time: ago(2 * time.Hour),
				},
				{
					AppGuid: "00000002-0003-0002-0000-000000000000",
					State:   "STOPPED",
					Time:    ago(1 * time.Hour),
				},
			},
		},
	},
	"00000003-0000-0000-0000-000000000000": Org{
		"00000003-0005-0000-0000-000000000000": Space{
			ServiceEvents: []ServiceEvent{
				{
					ServiceInstanceGuid: "00000003-0005-0001-0000-000000000000",
					State:               "CREATED",
					ServicePlanGuid:     "00000000-0000-0000-0000-300000000000",
					Time:                ago(24 * time.Hour),
				},
			},
		},
	},
}
