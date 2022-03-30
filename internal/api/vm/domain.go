package vm

import (
	"context"

	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
)

type VMRepository interface {
	GetVm(query query.Query) (*Vm, error)
	Query(ctx context.Context, name, startTime, endTime string) ([]*Vm, error)
}

type VMService interface {
	GetForecastStatusByID(uuid string) (string, error)
	GetForecastResultByID(uuid string) (map[string]*resource.ForecastUsage, error)
	GetForecastStatus(name string) (string, error)
	GetForecastResult(name string) (map[string]*resource.ForecastUsage, error)
	Forecast(query query.Query) (string, error)
	GetVm(query query.Query) (*Vm, error)
}

type Vm struct {
	Name string `json:"name"`
	// Resource usage list
	Usage map[string]*resource.ResourceUsageInfo `json:"usages"`
}

func (v Vm) UniqueName() string {
	return "vm/" + v.Name
}
