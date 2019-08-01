package eventio

type TotalCostReader interface {
	GetTotalCost() ([]TotalCost, error)
}

type TotalCost struct {
	PlanGUID string  `json:"plan_guid"`
	Cost     float32 `json:"cost"`
}
