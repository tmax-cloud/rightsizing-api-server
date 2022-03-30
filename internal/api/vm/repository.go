package vm

import (
	"context"

	"gorm.io/gorm"

	"rightsizing-api-server/internal/api/common/errors"
	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/models"
)

type vmRepository struct {
	db *gorm.DB
}

func NewVMRepository(db *gorm.DB) VMRepository {
	return &vmRepository{
		db: db,
	}
}

func (r *vmRepository) GetVm(query query.Query) (*Vm, error) {
	var (
		name      = query.Name
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	vms, err := r.Query(context.Background(), name, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(vms) == 0 {
		return nil, errors.NotFoundErr("vm", query.Name)
	} else if len(vms) > 1 {
		return nil, errors.NotUniqueErr("vm", query.Name)
	}

	return vms[0], nil
}

func (r *vmRepository) Query(ctx context.Context, name, startTime, endTime string) ([]*Vm, error) {
	var (
		numMetric      = VmMetricTables.Len()
		vmMetricUsages = make([][]models.Vm, numMetric)
		metricNames    = VmMetricTables.GetMetricNames()
		// goroutine and thread safe
		ctxDB = r.db.WithContext(ctx)
		// time formatting for query
	)

	for i := 0; i < numMetric; i++ {
		db := ctxDB.Scopes(VmMetricTables.GetIDTable(i)).
			Preload("Usage", func(db *gorm.DB) *gorm.DB {
				return db.Table(VmMetricTables.GetMetricTableName(i)).
					Where("value != 'Nan'").
					Where("bucket >= ? AND bucket <= ?", startTime, endTime).
					Order("bucket")
			})
		if name != "" {
			db = db.Where("domain=?", name)
		}
		err := db.Find(&vmMetricUsages[i]).Error
		if err != nil {
			return nil, err
		}
	}

	vmMap := make(map[string]*Vm, len(vmMetricUsages))
	for metricIdx := 0; metricIdx < numMetric; metricIdx++ {
		metricName := metricNames[metricIdx]
		for _, vmUsage := range vmMetricUsages[metricIdx] {
			name := vmUsage.Name
			if _, exist := vmMap[name]; !exist {
				vmMap[name] = &Vm{
					Name:  vmUsage.Name,
					Usage: make(map[string]*resource.ResourceUsageInfo),
				}
			}
			vmMap[name].Usage[metricName] = resource.NewResourceUsage(metricName, vmUsage.Usage)
		}
	}

	var vms []*Vm
	for _, vm := range vmMap {
		vms = append(vms, vm)
	}
	return vms, nil
}
