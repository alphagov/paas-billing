package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"time"
)

// MetaData contains the record metadata like id and creation date
type MetaData struct {
	GUID      string    `json:"guid"`
	CreatedAt time.Time `json:"created_at"`
}

// UsageEventList contains usage event records
type UsageEventList struct {
	Resources []UsageEvent `json:"resources"`
}

// UsageEvent represent a usage event record from the API
type UsageEvent struct {
	MetaData  MetaData
	EntityRaw json.RawMessage `json:"entity"`
}

func (u UsageEvent) String() string {
	return fmt.Sprintf("%s %s\n%s\n", u.MetaData.CreatedAt, u.MetaData.GUID, string(u.EntityRaw))
}
