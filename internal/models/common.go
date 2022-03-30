package models

import (
	"time"
)

type data []TimeSeriesDatapoint

type TimeSeriesDatapoint struct {
	ID    int64     `gorm:"column:series_id;references:id" json:"series_id"`
	Time  time.Time `gorm:"column:bucket" json:"time"`
	Value float64   `gorm:"column:value" json:"value"`
}

type Limit struct {
	ID    int64   `gorm:"column:series_id;references:id" json:"series_id"`
	Value float64 `gorm:"column:value" json:"value"`
}
