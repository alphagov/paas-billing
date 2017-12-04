package fixtures

import (
	"encoding/json"
	"strconv"
	"time"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	"github.com/alphagov/paas-usage-events-collector/db"
	uuid "github.com/satori/go.uuid"
)

type Orgs map[string]Org

type Org map[string]Space

type Space struct {
	AppEvents     []AppEvent
	ServiceEvents []ServiceEvent
}

type AppEvent struct {
	AppGuid               string
	State                 string
	InstanceCount         int
	MemoryInMBPerInstance int
	Time                  time.Time
}

type ServiceEvent struct {
	ServiceInstanceGuid string
	State               string
	ServicePlanGuid     string
	Time                time.Time
}

func (orgs Orgs) Insert(sqlClient *db.PostgresClient, now time.Time) error {
	usageEvents, err := orgsToUsageEvents(orgs, now)
	if err != nil {
		return err
	}

	appUsageEventList := cf.UsageEventList{
		Resources: usageEvents.AppEvents,
	}
	err = sqlClient.InsertUsageEventList(&appUsageEventList, db.AppUsageTableName)
	if err != nil {
		return err
	}

	serviceUsageEventList := cf.UsageEventList{
		Resources: usageEvents.ServiceEvents,
	}
	return sqlClient.InsertUsageEventList(&serviceUsageEventList, db.ServiceUsageTableName)
}

type usageEvents struct {
	AppEvents     []cf.UsageEvent
	ServiceEvents []cf.UsageEvent
}

func orgsToUsageEvents(orgs Orgs, now time.Time) (*usageEvents, error) {
	usageEvents := usageEvents{
		AppEvents:     []cf.UsageEvent{},
		ServiceEvents: []cf.UsageEvent{},
	}

	for orgGuid, org := range orgs {
		for spaceGuid, space := range org {
			for _, appEvent := range space.AppEvents {
				usageEvent := cf.UsageEvent{
					MetaData: cf.MetaData{
						GUID:      uuid.NewV4().String(),
						CreatedAt: appEvent.Time,
					},
					EntityRaw: json.RawMessage(`{
						"state": "` + appEvent.State + `",
						"app_guid": "` + appEvent.AppGuid + `",
						"app_name": "` + appEvent.AppGuid + `",
						"org_guid": "` + orgGuid + `",
						"space_guid": "` + spaceGuid + `",
						"instance_count": ` + strconv.Itoa(appEvent.InstanceCount) + `,
						"memory_in_mb_per_instance": ` + strconv.Itoa(appEvent.MemoryInMBPerInstance) + `
					}`),
				}
				usageEvents.AppEvents = append(usageEvents.AppEvents, usageEvent)
			}

			for _, serviceEvent := range space.ServiceEvents {
				usageEvent := cf.UsageEvent{
					MetaData: cf.MetaData{
						GUID:      uuid.NewV4().String(),
						CreatedAt: serviceEvent.Time,
					},
					EntityRaw: json.RawMessage(`{
						"state": "` + serviceEvent.State + `",
						"org_guid": "` + orgGuid + `",
						"space_guid": "` + spaceGuid + `",
						"service_plan_guid": "` + serviceEvent.ServicePlanGuid + `",
						"service_instance_guid": "` + serviceEvent.ServiceInstanceGuid + `",
						"service_instance_name": "` + serviceEvent.ServiceInstanceGuid + `",
						"service_instance_type": "managed_service_instance"
					}`),
				}
				usageEvents.ServiceEvents = append(usageEvents.ServiceEvents, usageEvent)
			}
		}
	}
	return &usageEvents, nil
}
