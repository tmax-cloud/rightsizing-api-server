package table

import "gorm.io/gorm"

type Table struct {
	MetricName      []string
	IDTableName     []string
	MetricTableName []string
}

func SetupTable(metricNames, idTables, metricTables []string) Table {
	return Table{
		MetricName:      metricNames,
		IDTableName:     idTables,
		MetricTableName: metricTables,
	}

}

func (t Table) GetIDTableName(idx int) string {
	if idx >= len(t.IDTableName) {
		return ""
	}
	return t.IDTableName[idx]
}

func (t Table) GetIDTable(idx int) func(tx *gorm.DB) *gorm.DB {
	if idx >= len(t.IDTableName) {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table(t.IDTableName[idx])
	}
}

func (t Table) GetMetricTableName(idx int) string {
	if idx >= len(t.MetricTableName) {
		return ""
	}
	return t.MetricTableName[idx]
}

func (t Table) GetMetricTable(idx int) func(tx *gorm.DB) *gorm.DB {
	if idx >= len(t.MetricTableName) {
		return nil
	}

	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table(t.MetricTableName[idx])
	}
}

func (t Table) GetMetricNames() []string {
	return t.MetricName
}

func (t Table) Len() int {
	return len(t.MetricTableName)
}
