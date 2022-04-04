package pod

import (
	"rightsizing-api-server/internal/api/common/table"
)

var (
	MetricName = []string{
		"cpu",
		"memory",
	}
)

var (
	IDTableName = []string{
		"prom_series.container:container_cpu_usage:rate",
		"prom_series.container_memory_working_set_bytes",
	}
	MetricTableName = []string{
		":container_cpu_usage:10min",
		":container_memory_working_set_bytes:10min",
	}
)

var ContainerMetricTables = table.SetupTable(MetricName, IDTableName, MetricTableName)

const (
	requestQuotaQuery = `SELECT DISTINCT ON (namespace_id, pod_id, container_id, resource_id) 
val(namespace_id) namespace, 
val(pod_id) pod, 
val(container_id) container, 
val(resource_id) resource, 
value 
FROM prom_metric.kube_pod_container_resource_requests `
	limitQuotaQuery = `SELECT DISTINCT ON (namespace_id, pod_id, container_id, resource_id) 
val(namespace_id) namespace, 
val(pod_id) pod, 
val(container_id) container, 
val(resource_id) resource, 
value
FROM prom_metric.kube_pod_container_resource_limits `
	allQuotaQuery    = `WHERE time >= now() - interval '5m' AND value != 'Nan' AND val(resource_id) IN ('cpu', 'memory') ORDER BY namespace_id, pod_id, container_id, resource_id, time DESC`
	targetQuotaQuery = `WHERE time >= now() - interval '5m' AND val(namespace_id) = ? AND val(pod_id) = ? AND value != 'NaN' AND val(resource_id) IN ('cpu', 'memory') ORDER BY namespace_id, pod_id, container_id, resource_id, time DESC`
)
