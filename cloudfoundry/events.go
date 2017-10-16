package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"time"
)

// AppUsageEventList contains the app usage event records
type AppUsageEventList struct {
	Resources []AppUsageEvent `json:"resources"`
}

// AppUsageEvent represent an app usage event record from the API
type AppUsageEvent struct {
	MetaData struct {
		GUID      string    `json:"guid"`
		CreatedAt time.Time `json:"created_at"`
	}
	EntityRaw json.RawMessage `json:"entity"`
}

func (a AppUsageEvent) String() string {
	return fmt.Sprintf("%s %s\n%s\n", a.MetaData.CreatedAt, a.MetaData.GUID, string(a.EntityRaw))
}

// ServiceUsageEventList contains the service usage event records
type ServiceUsageEventList struct {
	Resources []ServiceUsageEvent `json:"resources"`
}

// ServiceUsageEvent represent a service usage event record from the API
type ServiceUsageEvent struct {
	MetaData struct {
		GUID      string    `json:"guid"`
		CreatedAt time.Time `json:"created_at"`
	}
	EntityRaw json.RawMessage `json:"entity"`
}

func (a ServiceUsageEvent) String() string {
	return fmt.Sprintf("%s %s\n%s\n", a.MetaData.CreatedAt, a.MetaData.GUID, string(a.EntityRaw))
}
