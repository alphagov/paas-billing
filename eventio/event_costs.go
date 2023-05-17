package eventio

type TotalCostReader interface {
	GetTotalCost() ([]TotalCost, error)
}

type TotalCost struct {
	PlanGUID string  `json:"plan_guid"`
	PlanName string  `json:"-"`
	Kind     string  `json:"-"`
	Cost     float32 `json:"cost"`
}
