package resource

import (
	"encoding/base64"
	"math"
	"sync"

	"rightsizing-api-server/internal/models"
	pb "rightsizing-api-server/proto"

	"github.com/pquerna/ffjson/ffjson"
)

const (
	threshold = 100
)

const (
	StatusUnknown   = "unknown"
	StatusHealty    = "healty"
	StatusNotHealty = "inefficient"
)

type TimeseriesData []TimeSeriesDatapoint

type TimeSeriesDatapoint struct {
	Time  int64
	Value float64
}

type ForecastUsage struct {
	Name  string                               `json:"name"`
	Usage map[string][]*pb.TimeSeriesDatapoint `json:"usage"`
}

func EncodeForecastUsage(usage []*ForecastUsage) (string, error) {
	buf, err := ffjson.Marshal(usage)
	b64Encoded := base64.StdEncoding.EncodeToString(buf)
	if err != nil {
		return "", err
	}
	return b64Encoded, nil
}

func DecodeForecastUsage(data string) ([]*ForecastUsage, error) {
	var usage []*ForecastUsage

	b64Decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	if err := ffjson.Unmarshal(b64Decoded, &usage); err != nil {
		return nil, err
	}
	return usage, nil
}

type ResourceUsageInfo struct {
	lock           sync.RWMutex
	ResourceName   string         `json:"name"`
	Usage          TimeseriesData `json:"usage,omitempty" description:"resource usage"`
	Request        float64        `json:"request,omitempty"`
	Limit          float64        `json:"limit,omitempty"`
	CurrentUsage   float64        `json:"current_usage" description:"current usage"`
	OptimizedUsage float64        `json:"optimized_usage,omitempty"`
	Status         *string        `json:"status,omitempty" description:"resource status"`
}

func NewResourceUsage(name string, data []models.TimeSeriesDatapoint) *ResourceUsageInfo {
	datapoints := make(TimeseriesData, len(data))
	for i, point := range data {
		datapoints[i] = TimeSeriesDatapoint{
			Time:  point.Time.Unix(),
			Value: point.Value,
		}
	}

	var currentUsage float64
	if len(data) > 0 {
		currentUsage = data[len(data)-1].Value
	}

	return &ResourceUsageInfo{
		ResourceName: name,
		Usage:        datapoints,
		CurrentUsage: currentUsage,
	}
}

func (info *ResourceUsageInfo) SetOptimizedUsage(value float64) {
	info.lock.Lock()
	defer info.lock.Unlock()

	info.OptimizedUsage = value
}

func (info *ResourceUsageInfo) GetStandardQuota() float64 {
	standard := info.Request
	if info.Request == 0 {
		standard = info.Limit
		if info.Limit == 0 {
			standard = -1
		}
	}
	return standard
}

func (info *ResourceUsageInfo) GetStatus() string {
	info.lock.Lock()
	defer info.lock.Unlock()

	const epsilon = 0.2

	if len(info.Usage) < threshold {
		return StatusUnknown
	}

	standard := info.GetStandardQuota()
	if standard == -1 {
		return StatusUnknown
	}
	eps := math.Abs(info.CurrentUsage-standard) / standard
	if eps < epsilon {
		return StatusHealty
	}
	return StatusNotHealty
}
