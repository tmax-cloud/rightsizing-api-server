package resource

type ClusterInfo struct {
	AverageUsage        float64 `json:"average_usage"`
	Count               int     `json:"count"`
	OverAllocatedCount  int     `json:"over_allocated_count"`
	UnderAllocatedCount int     `json:"under_allocated_count"`
	OptimizedCount      int     `json:"optimized_count"`
}

type CachedClusterInfo struct {
	Namespace string
	Name      string
	Info      map[string]*ResourceUsageInfo
}
