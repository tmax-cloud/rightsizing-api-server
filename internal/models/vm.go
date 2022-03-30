package models

type VmID struct {
	ID   int64  `gorm:"column:series_id" json:"series_id"`
	Name string `gorm:"column:domain"    json:"vm"`
}

type Vm struct {
	VmID
	Usage []TimeSeriesDatapoint `gorm:"foreignKey:ID" json:"usage"`
	Limit []float64             `gorm:"foreignKey:ID" json:"usage"`
}
