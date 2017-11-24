package fixtures

import (
	"encoding/json"
	"fmt"
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
	err = sqlClient.InsertUsageEventList(&serviceUsageEventList, db.ServiceUsageTableName)
	if err != nil {
		return err
	}

	return nil
}

type usageEvents struct {
	AppEvents     []cf.UsageEvent
	ServiceEvents []cf.UsageEvent
}

type appUsageEventEntity struct {
	State                 string `json:"state"`
	AppGuid               string `json:"app_guid"`
	AppName               string `json:"app_name"`
	OrgGuid               string `json:"org_guid"`
	SpaceGuid             string `json:"space_guid"`
	InstanceCount         int    `json:"instance_count"`
	MemoryInMBPerInstance int    `json:"memory_in_mb_per_instance"`
	PreviousState         string `json:"previous_state"`
}

type serviceUsageEventEntity struct {
	State           string `json:"state"`
	Guid            string `json:"guid"`
	ServicePlanGuid string `json:"service_plan_guid"`
}

func orgsToUsageEvents(orgs Orgs, now time.Time) (*usageEvents, error) {
	usageEvents := usageEvents{
		AppEvents:     []cf.UsageEvent{},
		ServiceEvents: []cf.UsageEvent{},
	}

	for orgGuid, org := range orgs {
		for spaceGuid, space := range org {
			appsPreviousState := map[string]string{}
			for _, appEvent := range space.AppEvents {
				appPreviousState, ok := appsPreviousState[appEvent.AppGuid]
				if !ok {
					appPreviousState = "STOPPED"
				}
				usageEventEntity := appUsageEventEntity{
					State:                 appEvent.State,
					AppGuid:               appEvent.AppGuid,
					AppName:               fmt.Sprintf("app-%s", appEvent.AppGuid),
					OrgGuid:               orgGuid,
					SpaceGuid:             spaceGuid,
					InstanceCount:         appEvent.InstanceCount,
					MemoryInMBPerInstance: appEvent.MemoryInMBPerInstance,
					PreviousState:         appPreviousState,
				}
				usageEventEntityJSON, err := json.Marshal(usageEventEntity)
				if err != nil {
					return nil, err
				}
				usageEvent := cf.UsageEvent{
					MetaData: cf.MetaData{
						GUID:      uuid.NewV4().String(),
						CreatedAt: appEvent.Time,
					},
					EntityRaw: json.RawMessage(usageEventEntityJSON),
				}
				usageEvents.AppEvents = append(usageEvents.AppEvents, usageEvent)
			}

			for _, serviceEvent := range space.ServiceEvents {
				usageEventEntity := serviceUsageEventEntity{
					State:           serviceEvent.State,
					Guid:            serviceEvent.ServiceInstanceGuid,
					ServicePlanGuid: serviceEvent.ServicePlanGuid,
				}
				usageEventEntityJSON, err := json.Marshal(usageEventEntity)
				if err != nil {
					return nil, err
				}
				usageEvent := cf.UsageEvent{
					MetaData: cf.MetaData{
						GUID:      uuid.NewV4().String(),
						CreatedAt: serviceEvent.Time,
					},
					EntityRaw: json.RawMessage(usageEventEntityJSON),
				}
				usageEvents.ServiceEvents = append(usageEvents.ServiceEvents, usageEvent)
			}
		}
	}
	return &usageEvents, nil
}
