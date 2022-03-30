package vm

import (
	"rightsizing-api-server/internal/api/common/table"
)

var (
	MetricName = []string{
		"memory",
	}
	IDTableName = []string{
		"prom_series.libvirt_domain_info_memory_usage_bytes",
	}
	MetricTableName = []string{
		":libvirt_domain_info_memory_usage_bytes:10min",
	}
)

var VmMetricTables = table.SetupTable(MetricName, IDTableName, MetricTableName)
