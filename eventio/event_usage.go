package eventio

type UsageEventReader interface {
	GetUsageEventRows(filter EventFilter) (UsageEventRows, error)
	GetUsageEvents(filter EventFilter) ([]UsageEvent, error)
}

type UsageEvent struct {
	EventGUID     string `json:"event_guid"`
	EventStart    string `json:"event_start"`
	EventStop     string `json:"event_stop"`
	ResourceGUID  string `json:"resource_guid"`
	ResourceName  string `json:"resource_name"`
	ResourceType  string `json:"resource_type"`
	OrgGUID       string `json:"org_guid"`
	SpaceGUID     string `json:"space_guid"`
	PlanGUID      string `json:"plan_guid"`
	PlanName      string `json:"plan_name"`
	ServiceGUID   string `json:"service_guid"`
	ServiceName   string `json:"service_name"`
	NumberOfNodes int64  `json:"number_of_nodes"`
	MemoryInMB    int64  `json:"memory_in_mb"`
	StorageInMB   int64  `json:"storage_in_mb"`
}

type UsageEventRows interface {
	Next() bool
	Close() error
	Err() error
	EventJSON() ([]byte, error)
	Event() (*UsageEvent, error)
}
