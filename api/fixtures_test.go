package api_test

import (
	"time"

	"github.com/alphagov/paas-billing/db"
	. "github.com/alphagov/paas-billing/fixtures"
)

func ago(t time.Duration) time.Time {
	return now.Add(0 - t)
}

var planFixtures = Plans{
	{
		ID:        10,
		Name:      "ComputePlanA",
		PlanGuid:  db.ComputePlanGuid,
		ValidFrom: ago(100 * time.Hour),
		Formula:   "($time_in_seconds / 60 / 60) * $memory_in_mb * 1",
		Components: []PricingPlanComponent{
			{
				ID:      101,
				Name:    "ComputePlanA/1",
				Formula: "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.7",
			},
			{
				ID:      102,
				Name:    "ComputePlanA/2",
				Formula: "($time_in_seconds / 60 / 60) * $memory_in_mb * 0.3",
			},
		},
	},
	{
		ID:        11,
		Name:      "ComputePlanB",
		PlanGuid:  db.ComputePlanGuid,
		ValidFrom: ago(1 * time.Hour),
		Formula:   "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
		Components: []PricingPlanComponent{
			{
				ID:      111,
				Name:    "ComputePlanB/1",
				Formula: "($time_in_seconds / 60 / 60) * $memory_in_mb * 2",
			},
		},
	},
	{
		ID:        20,
		Name:      "ServicePlanA",
		PlanGuid:  "00000000-0000-0000-0000-100000000000",
		ValidFrom: ago(100 * time.Hour),
		Formula:   "($time_in_seconds / 60 / 60) * 0.5",
		Components: []PricingPlanComponent{
			{
				ID:      201,
				Name:    "ServicePlanA/1",
				Formula: "($time_in_seconds / 60 / 60) * 0.2",
			},
			{
				ID:      202,
				Name:    "ServicePlanA/2",
				Formula: "($time_in_seconds / 60 / 60) * 0.3",
			},
		},
	},
	{
		ID:        30,
		Name:      "ServicePlanB",
		PlanGuid:  "00000000-0000-0000-0000-200000000000",
		ValidFrom: ago(100 * time.Hour),
		Formula:   "($time_in_seconds / 60 / 60) * 1",
		Components: []PricingPlanComponent{
			{
				ID:      301,
				Name:    "ServicePlanB/1",
				Formula: "($time_in_seconds / 60 / 60) * 1",
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
var orgsFixtures = Orgs{
	"00000001-0000-0000-0000-000000000000": Org{
		"00000001-0001-0000-0000-000000000000": Space{
			AppEvents: []AppEvent{
				{
					AppGuid:               "00000001-0001-0001-0000-000000000000",
					State:                 "STARTED",
					InstanceCount:         1,
					MemoryInMBPerInstance: 64,
					Time: ago(10 * time.Hour),
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
}
