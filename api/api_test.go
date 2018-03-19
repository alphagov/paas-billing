package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"net/http"
	"net/http/httptest"
	"net/url"

	cf "github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/db"
	"github.com/alphagov/paas-billing/db/dbhelper"
	"github.com/alphagov/paas-billing/fixtures"
	"github.com/alphagov/paas-billing/server"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ThreeMonthsInHours = 1488
	OneMonthInHours    = 720
)

var (
	now = time.Date(2017, time.October, 1, 0, 0, 0, 0, time.UTC)
)

var _ = Describe("API", func() {

	type ResourceReport struct {
		Name        string `json:"name"`
		PriceIncVAT int64  `json:"price_inc_vat"`
		PriceExVAT  int64  `json:"price_ex_vat"`
	}
	type SpaceReport struct {
		SpaceGuid   string           `json:"space_guid"`
		PriceIncVAT int64            `json:"price_inc_vat"`
		PriceExVAT  int64            `json:"price_ex_vat"`
		Resources   []ResourceReport `json:"resources"`
	}
	type OrgReport struct {
		OrgGuid     string        `json:"org_guid"`
		PriceIncVAT int64         `json:"price_inc_vat"`
		PriceExVAT  int64         `json:"price_ex_vat"`
		Spaces      []SpaceReport `json:"spaces"`
	}

	var (
		X10ComputePlan = fixtures.Plan{
			ID:        1,
			Name:      "x10-compute-plan",
			ValidFrom: monthsAgo(3),
			PlanGuid:  db.ComputePlanGuid,
			Components: []fixtures.PricingPlanComponent{
				{
					ID:        11,
					Name:      "x10-compute-plan/1",
					Formula:   "$time_in_seconds * 4",
					VATRateID: 1,
					Currency:  "GBP",
				},
				{
					ID:        12,
					Name:      "x10-compute-plan/2",
					Formula:   "$time_in_seconds * 6",
					VATRateID: 1,
					Currency:  "GBP",
				},
			},
		}
		X4ComputePlan = fixtures.Plan{
			ID:        2,
			Name:      "x4-compute-plan",
			ValidFrom: monthsAgo(1),
			PlanGuid:  db.ComputePlanGuid,
			Components: []fixtures.PricingPlanComponent{
				{
					ID:        21,
					Name:      "x4-compute-plan/1",
					Formula:   "$time_in_seconds * 1",
					VATRateID: 1,
					Currency:  "GBP",
				},
				{
					ID:        22,
					Name:      "x4-compute-plan/2",
					Formula:   "$time_in_seconds * 3",
					VATRateID: 1,
					Currency:  "GBP",
				},
			},
		}
		X2ServicePlan = fixtures.Plan{
			ID:          3,
			Name:        "x2-service-plan",
			ValidFrom:   time.Unix(0, 0),
			PlanGuid:    uuid.NewV4().String(),
			MemoryInMb:  10240,
			StorageInMb: 102400,
			Components: []fixtures.PricingPlanComponent{
				{
					ID:        31,
					Name:      "x2-service-plan/1",
					Formula:   "$time_in_seconds * 0.5",
					VATRateID: 1,
					Currency:  "GBP",
				},
				{
					ID:        32,
					Name:      "x2-service-plan/2",
					Formula:   "$time_in_seconds * 1.5",
					VATRateID: 1,
					Currency:  "GBP",
				},
			},
		}
		pricingPlans = fixtures.Plans{
			X4ComputePlan,
			X10ComputePlan,
			X2ServicePlan,
		}
	)

	var (
		sqlClient *db.PostgresClient
		connstr   string
	)

	BeforeEach(func() {
		var err error
		connstr, err = dbhelper.CreateDB()
		Expect(err).ToNot(HaveOccurred())
		sqlClient, err = db.NewPostgresClient(connstr)
		Expect(err).ToNot(HaveOccurred())
		err = sqlClient.InitSchema()
		Expect(err).ToNot(HaveOccurred())

		err = pricingPlans.Insert(sqlClient)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := sqlClient.Close()
		Expect(err).ToNot(HaveOccurred())
		err = dbhelper.DropDB(connstr)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Resource Usage Queries", func() {

		type UsageEntry struct {
			Guid            string    `json:"guid"`
			OrgGuid         string    `json:"org_guid"`
			SpaceGuid       string    `json:"space_guid"`
			Name            string    `json:"name"`
			PricingPlanId   int       `json:"pricing_plan_id"`
			PricingPlanName string    `json:"pricing_plan_name"`
			MemoryInMb      uint      `json:"memory_in_mb"`
			StorageInMb     uint      `json:"storage_in_mb"`
			From            time.Time `json:"from"`
			To              time.Time `json:"to"`
			PriceIncVAT     int64     `json:"price_inc_vat"`
			PriceExVAT      int64     `json:"price_ex_vat"`
		}

		cases := []struct {
			Name           string
			RequestQuery   url.Values
			AppEvents      []cf.UsageEvent
			ServiceEvents  []cf.UsageEvent
			ExpectedOutput []UsageEntry
		}{
			{
				Name: "should return 1 compute usage row for a pair of STARTED / STOPPED app events (1x instance)",
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app",
						OrgGuid:         "org_guid",
						SpaceGuid:       "space_guid",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name",
						MemoryInMb:      512,
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return 2 compute usage row for a pair of STARTED / STOPPED app events (1x instance) that spans a pricing_plan boundry",
				RequestQuery: url.Values{
					"from": []string{monthsAgo(3).Format(time.RFC3339)},
					"to":   []string{now.Format(time.RFC3339)},
				},
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: monthsAgo(3)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app",
						OrgGuid:         "org_guid",
						SpaceGuid:       "space_guid",
						PricingPlanName: X10ComputePlan.Name,
						PricingPlanId:   X10ComputePlan.ID,
						Name:            "app_name",
						MemoryInMb:      512,
						From:            monthsAgo(3),
						To:              monthsAgo(1),
						PriceExVAT:      ThreeMonthsInHours * (60 * 60) * 10,
						PriceIncVAT:     int64(ThreeMonthsInHours * (60 * 60) * 10 * 1.2),
					},
					{
						Guid:            "app",
						OrgGuid:         "org_guid",
						SpaceGuid:       "space_guid",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name",
						MemoryInMb:      512,
						From:            monthsAgo(1),
						To:              now,
						PriceExVAT:      OneMonthInHours * (60 * 60) * 4,
						PriceIncVAT:     int64(OneMonthInHours * (60 * 60) * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return 2 compute usage row for a pair of STARTED / STOPPED app events (2x instance)",
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 2,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app",
							"app_name": "app_name",
							"org_guid": "org_guid",
							"space_guid": "space_guid",
							"instance_count": 2,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app",
						OrgGuid:         "org_guid",
						SpaceGuid:       "space_guid",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name",
						MemoryInMb:      512,
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					},
					{
						Guid:            "app",
						OrgGuid:         "org_guid",
						SpaceGuid:       "space_guid",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name",
						MemoryInMb:      512,
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return 3 resource usage rows for two pairs of STARTED/STOPPED app usage events (app1 * 1inst) + (app2 * 2inst)",
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-90 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-90 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app2",
							"app_name": "app_name2",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"instance_count": 2,
							"memory_in_mb_per_instance": 64,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app2",
							"app_name": "app_name2",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"instance_count": 2,
							"memory_in_mb_per_instance": 64,
							"previous_state": "STARTED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name1",
						MemoryInMb:      512,
						From:            now.Add(-90 * time.Minute),
						To:              now.Add(-60 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					}, {
						Guid:            "app2",
						OrgGuid:         "org_guid2",
						SpaceGuid:       "space_guid2",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name2",
						MemoryInMb:      64,
						From:            now.Add(-90 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      60 * 60 * 4,
						PriceIncVAT:     int64(60 * 60 * 4 * 1.2),
					}, {
						Guid:            "app2",
						OrgGuid:         "org_guid2",
						SpaceGuid:       "space_guid2",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name2",
						MemoryInMb:      64,
						From:            now.Add(-90 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      60 * 60 * 4,
						PriceIncVAT:     int64(60 * 60 * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return 3 resource usage rows when an app is updated (scale from 1x to 2x instances)",
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-90 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 2,
							"memory_in_mb_per_instance": 1024,
							"previous_state": "STARTED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid":"app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 2,
							"memory_in_mb_per_instance": 1024,
							"previous_state": "STARTED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name1",
						MemoryInMb:      512,
						From:            now.Add(-90 * time.Minute),
						To:              now.Add(-60 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					}, {
						Guid:            "app1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name1",
						MemoryInMb:      1024,
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					}, {
						Guid:            "app1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name1",
						MemoryInMb:      1024,
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      30 * 60 * 4,
						PriceIncVAT:     int64(30 * 60 * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return a resource usage row (up to the selected range) for an app that has not been stopped",
				RequestQuery: url.Values{
					"to": []string{now.Format(time.RFC3339)},
				},
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-10 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanName: X4ComputePlan.Name,
						PricingPlanId:   X4ComputePlan.ID,
						Name:            "app_name1",
						MemoryInMb:      512,
						From:            now.Add(-10 * time.Minute),
						To:              now,
						PriceExVAT:      10 * 60 * 4,
						PriceIncVAT:     int64(10 * 60 * 4 * 1.2),
					},
				},
			},
			{
				Name: "should return a single resource usage row (up to current time) for a service that has not been stopped",
				RequestQuery: url.Values{
					"to": []string{now.Format(time.RFC3339)},
				},
				ServiceEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-1 * time.Hour)},
						EntityRaw: json.RawMessage(`{
							"state": "CREATED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "service_instance1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						Name:            "db-service-1",
						PricingPlanId:   X2ServicePlan.ID,
						PricingPlanName: X2ServicePlan.Name,
						MemoryInMb:      X2ServicePlan.MemoryInMb,
						StorageInMb:     X2ServicePlan.StorageInMb,
						From:            now.Add(-1 * time.Hour),
						To:              now,
						PriceExVAT:      7200,
						PriceIncVAT:     int64(7200 * 1.2),
					},
				},
			},
			{
				Name: "should return two resource rows for a service that was UPDATED",
				ServiceEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "CREATED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-50 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "UPDATED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-40 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "DELETED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_plan_name": "service_plan_name1",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "service_instance1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanId:   X2ServicePlan.ID,
						PricingPlanName: X2ServicePlan.Name,
						MemoryInMb:      X2ServicePlan.MemoryInMb,
						StorageInMb:     X2ServicePlan.StorageInMb,
						Name:            "db-service-1",
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-50 * time.Minute),
						PriceExVAT:      1200,
						PriceIncVAT:     int64(1200 * 1.2),
					},
					{
						Guid:            "service_instance1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanId:   X2ServicePlan.ID,
						PricingPlanName: X2ServicePlan.Name,
						MemoryInMb:      X2ServicePlan.MemoryInMb,
						StorageInMb:     X2ServicePlan.StorageInMb,
						Name:            "db-service-1",
						From:            now.Add(-50 * time.Minute),
						To:              now.Add(-40 * time.Minute),
						PriceExVAT:      1200,
						PriceIncVAT:     int64(1200 * 1.2),
					},
				},
			}, {
				Name: "should return a single resource usage item for a pair of CREATED/STOPPED service events",
				ServiceEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "CREATED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_plan_name": "service_plan_name1",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "DELETED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_plan_name": "service_plan_name1",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "service_instance1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanId:   X2ServicePlan.ID,
						PricingPlanName: X2ServicePlan.Name,
						MemoryInMb:      X2ServicePlan.MemoryInMb,
						StorageInMb:     X2ServicePlan.StorageInMb,
						Name:            "db-service-1",
						From:            now.Add(-60 * time.Minute),
						To:              now.Add(-30 * time.Minute),
						PriceExVAT:      60 * 30 * 2,
						PriceIncVAT:     int64(60 * 30 * 2 * 1.2),
					},
				},
			}, {
				Name: "should only return resource usage for the given range",
				RequestQuery: url.Values{
					"from": []string{now.Add(-60 * time.Minute).Format(time.RFC3339)},
					"to":   []string{now.Add(-30 * time.Minute).Format(time.RFC3339)},
				},
				AppEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-100 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-100 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STARTED",
							"app_guid": "app2",
							"app_name": "app_name2",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STOPPED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-61 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app1",
							"app_name": "app_name1",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-31 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "STOPPED",
							"app_guid": "app2",
							"app_name": "app_name2",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"instance_count": 1,
							"memory_in_mb_per_instance": 512,
							"previous_state": "STARTED"
						}`),
					},
				},
				ServiceEvents: []cf.UsageEvent{
					{
						MetaData: cf.MetaData{CreatedAt: now.Add(-41 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "CREATED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_plan_name": "service_plan_name1",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-31 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "DELETED",
							"org_guid": "org_guid1",
							"space_guid": "space_guid1",
							"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
							"service_plan_name": "service_plan_name1",
							"service_instance_guid": "service_instance1",
							"service_instance_name": "db-service-1",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-131 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "CREATED",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"service_plan_guid": "service_plan_guid2",
							"service_plan_name": "service_plan_name2",
							"service_instance_guid": "service_instance2",
							"service_instance_name": "db-service-2",
							"service_instance_type": "managed_service_instance"
						}`),
					}, {
						MetaData: cf.MetaData{CreatedAt: now.Add(-101 * time.Minute)},
						EntityRaw: json.RawMessage(`{
							"state": "DELETED",
							"org_guid": "org_guid2",
							"space_guid": "space_guid2",
							"service_plan_guid": "service_plan_guid2",
							"service_plan_name": "service_plan_name2",
							"service_instance_guid": "service_instance2",
							"service_instance_name": "db-service-2",
							"service_instance_type": "managed_service_instance"
						}`),
					},
				},
				ExpectedOutput: []UsageEntry{
					{
						Guid:            "app2",
						OrgGuid:         "org_guid2",
						SpaceGuid:       "space_guid2",
						PricingPlanId:   X4ComputePlan.ID,
						PricingPlanName: X4ComputePlan.Name,
						Name:            "app_name2",
						MemoryInMb:      512,
						From:            now.Add(-60 * time.Minute), // start of selected range
						To:              now.Add(-31 * time.Minute),
						PriceExVAT:      29 * 60 * 4,
						PriceIncVAT:     int64(29 * 60 * 4 * 1.2)},
					{
						Guid:            "service_instance1",
						OrgGuid:         "org_guid1",
						SpaceGuid:       "space_guid1",
						PricingPlanId:   X2ServicePlan.ID,
						PricingPlanName: X2ServicePlan.Name,
						MemoryInMb:      X2ServicePlan.MemoryInMb,
						StorageInMb:     X2ServicePlan.StorageInMb,
						Name:            "db-service-1",
						From:            now.Add(-41 * time.Minute),
						To:              now.Add(-31 * time.Minute),
						PriceExVAT:      10 * 60 * 2,
						PriceIncVAT:     int64(10 * 60 * 2 * 1.2),
					},
				},
			},
		}

		for testCaseNumber, testCase := range cases {

			tc := testCase
			tcIndex := testCaseNumber

			It(tc.Name, func() {

				if tc.AppEvents != nil {
					for i := range tc.AppEvents {
						// generate unique GUID for each event
						tc.AppEvents[i].MetaData.GUID = fmt.Sprintf("a-usage-%d-%d", tcIndex, i)
					}
					err := sqlClient.InsertUsageEventList(&cf.UsageEventList{
						Resources: tc.AppEvents,
					}, db.AppUsageTableName)
					Expect(err).ToNot(HaveOccurred())
				}

				if tc.ServiceEvents != nil {
					for i := range tc.ServiceEvents {
						// generate unique GUID for each event
						tc.ServiceEvents[i].MetaData.GUID = fmt.Sprintf("s-usage-%d-%d", tcIndex, i)
					}
					err := sqlClient.InsertUsageEventList(&cf.UsageEventList{
						Resources: tc.ServiceEvents,
					}, db.ServiceUsageTableName)
					Expect(err).ToNot(HaveOccurred())
				}

				err := sqlClient.UpdateViews()
				Expect(err).ToNot(HaveOccurred())

				u, err := url.Parse("/events")
				Expect(err).ToNot(HaveOccurred())
				if tc.RequestQuery != nil {
					u.RawQuery = tc.RequestQuery.Encode()
				}

				req, err := http.NewRequest("GET", u.String(), nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSONCharsetUTF8)
				req.Header.Set(echo.HeaderAuthorization, FakeBearerToken)

				rec := httptest.NewRecorder()

				e := server.New(sqlClient, AuthenticatedNonAdmin, nil)
				e.ServeHTTP(rec, req)

				res := rec.Result()
				body, _ := ioutil.ReadAll(res.Body)
				Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))

				actualOutput := []UsageEntry{}
				err = json.Unmarshal(body, &actualOutput)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to unmarshal json: %s\nbody: %s", err, string(body)))

				Expect(actualOutput).To(Equal(tc.ExpectedOutput))
			})
		}
	})

	Context("Simulated report", func() {
		var (
			path = "/forecast/report"
		)

		It("should produce a report", func() {
			reqBody := bytes.NewBufferString(`{
				"events": [
					{
						"name": "o1s1-app1",
						"space_guid": "o1s1",
						"plan_guid": "` + X4ComputePlan.PlanGuid + `",
						"memory_in_mb": 1
					},
					{
						"name": "o1s1-db1",
						"space_guid": "o1s1",
						"plan_guid": "` + X2ServicePlan.PlanGuid + `"
					}
				]
			}`)

			u, err := url.Parse(path + "?from=" + now.Format(time.RFC3339) + "&to=" + now.Add(24*time.Hour).Format(time.RFC3339))
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("POST", u.String(), reqBody)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSONCharsetUTF8)

			rec := httptest.NewRecorder()

			e := server.New(sqlClient, NonAuthenticated, nil)
			e.ServeHTTP(rec, req)

			res := rec.Result()
			body, _ := ioutil.ReadAll(res.Body)
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))

			var actualOutput OrgReport
			err = json.Unmarshal(body, &actualOutput)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to unmarshal json: %s\nbody: %s", err, string(body)))

			expectedOutput := OrgReport{
				OrgGuid:     "simulated-org",
				PriceExVAT:  51840000,
				PriceIncVAT: int64(51840000 * 1.2),
				Spaces: []SpaceReport{
					{
						SpaceGuid:   "o1s1",
						PriceExVAT:  51840000,
						PriceIncVAT: int64(51840000 * 1.2),
						Resources: []ResourceReport{
							{
								Name:        "o1s1-app1",
								PriceExVAT:  34560000,
								PriceIncVAT: int64(34560000 * 1.2),
							},
							{
								Name:        "o1s1-db1",
								PriceExVAT:  17280000,
								PriceIncVAT: int64(17280000 * 1.2),
							},
						},
					},
				},
			}
			Expect(actualOutput).To(Equal(expectedOutput))
		})
	})

	Context("Org report", func() {

		var (
			org_guid = "o1"
			path     = "/report/" + org_guid
		)

		It("should produce a report", func() {
			appEvents := []cf.UsageEvent{
				{
					MetaData: cf.MetaData{CreatedAt: now.Add(-60 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STARTED",
						"app_guid": "o1s1-app1",
						"app_name": "o1s1-app1",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STOPPED"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-30 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STOPPED",
						"app_guid": "o1s1-app1",
						"app_name": "o1s1-app1-renamed",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STARTED"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-58 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STARTED",
						"app_guid": "o1s1-app2",
						"app_name": "o1s1-app1",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STARTED"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-28 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STOPPED",
						"app_guid": "o1s1-app2",
						"app_name": "o1s1-app1-renamed",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STARTED"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-49 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STARTED",
						"app_guid": "o2s1-app1",
						"app_name": "o2s1-app1",
						"org_guid": "o2",
						"space_guid": "o1s2",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STOPPED"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-12 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "STOPPED",
						"app_guid": "o2s1-app1",
						"app_name": "o2s1-app1-renamed",
						"org_guid": "o2",
						"space_guid": "o1s2",
						"instance_count": 2,
						"memory_in_mb_per_instance": 512,
						"previous_state": "STARTED"
					}`),
				},
			}
			serviceEvents := []cf.UsageEvent{
				{
					MetaData: cf.MetaData{CreatedAt: now.Add(-41 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "CREATED",
						"service_instance_guid": "o2s1-db1",
						"service_instance_name": "o2s1-db1",
						"org_guid": "o2",
						"space_guid": "o2s1",
						"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
						"service_instance_type": "managed_service_instance"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-31 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "DELETED",
						"service_instance_guid": "o2s1-db1",
						"service_instance_name": "o2s1-db1-renamed",
						"org_guid": "o2",
						"space_guid": "o2s1",
						"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
						"service_instance_type": "managed_service_instance"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-131 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "CREATED",
						"service_instance_guid": "o1s1-db1",
						"service_instance_name": "o1s1-db1",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
						"service_instance_type": "managed_service_instance"
					}`),
				}, {
					MetaData: cf.MetaData{CreatedAt: now.Add(-101 * time.Minute)},
					EntityRaw: json.RawMessage(`{
						"state": "DELETED",
						"service_instance_guid": "o1s1-db1",
						"service_instance_name": "o1s1-db1-renamed",
						"org_guid": "o1",
						"space_guid": "o1s1",
						"service_plan_guid": "` + X2ServicePlan.PlanGuid + `",
						"service_instance_type": "managed_service_instance"
					}`),
				},
			}

			for i := range appEvents {
				// generate unique GUID for each event
				appEvents[i].MetaData.GUID = fmt.Sprintf("a-reporttest-%d", i)
			}
			err := sqlClient.InsertUsageEventList(&cf.UsageEventList{
				Resources: appEvents,
			}, db.AppUsageTableName)
			Expect(err).ToNot(HaveOccurred())

			for i := range serviceEvents {
				// generate unique GUID for each event
				serviceEvents[i].MetaData.GUID = fmt.Sprintf("s-reporttest-%d", i)
			}
			err = sqlClient.InsertUsageEventList(&cf.UsageEventList{
				Resources: serviceEvents,
			}, db.ServiceUsageTableName)
			Expect(err).ToNot(HaveOccurred())

			err = sqlClient.UpdateViews()
			Expect(err).ToNot(HaveOccurred())

			u, err := url.Parse(path + "?from=2001-01-01T00:00:00Z&to=" + now.Add(72*time.Hour).Format(time.RFC3339))
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("GET", u.String(), nil)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSONCharsetUTF8)
			req.Header.Set(echo.HeaderAuthorization, FakeBearerToken)

			rec := httptest.NewRecorder()

			e := server.New(sqlClient, AuthenticatedNonAdmin, nil)
			e.ServeHTTP(rec, req)

			res := rec.Result()
			body, _ := ioutil.ReadAll(res.Body)
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))

			var actualOutput OrgReport
			err = json.Unmarshal(body, &actualOutput)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to unmarshal json: %s\nbody: %s", err, string(body)))

			expectedOutput := OrgReport{
				OrgGuid:     "o1",
				PriceExVAT:  3240000,
				PriceIncVAT: int64(3240000 * 1.2),
				Spaces: []SpaceReport{
					{
						SpaceGuid:   "o1s1",
						PriceExVAT:  3240000,
						PriceIncVAT: int64(3240000 * 1.2),
						Resources: []ResourceReport{
							{
								Name:        "o1s1-app1",
								PriceExVAT:  2880000,
								PriceIncVAT: int64(2880000 * 1.2),
							},
							{
								Name:        "o1s1-db1",
								PriceExVAT:  360000,
								PriceIncVAT: int64(360000 * 1.2),
							},
						},
					},
				},
			}
			Expect(actualOutput).To(Equal(expectedOutput))
		})
	})
})
