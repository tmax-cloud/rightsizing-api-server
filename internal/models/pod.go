package models

type ContainerID struct {
	ID        int64  `gorm:"column:series_id" json:"series_id"`
	Namespace string `gorm:"column:namespace" json:"namespace"`
	Pod       string `gorm:"column:pod"       json:"pod"`
	Name      string `gorm:"column:container" json:"container"`
}

type Container struct {
	// To find series_id of container
	ContainerID
	// time-series data
	Usage []TimeSeriesDatapoint `gorm:"ForeignKey:ID" json:"usage"`
}

type ContainerLimit struct {
	ContainerID
	Value float64 `gorm:"column:value" json:"limit"`
}

type ContainerQuota struct {
	ContainerID
	Resource string  `gorm:"column:resource" json:"resource"`
	Value    float64 `gorm:"column:value" json:"value"`
}
