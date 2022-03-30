package grpc

import (
	"context"

	"google.golang.org/grpc"

	"rightsizing-api-server/internal/api/common/resource"
	pb "rightsizing-api-server/proto"
)

type Client struct {
	forecastClient    pb.ForecastClient
	rightsizingClient pb.RightsizingClient
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		forecastClient:    pb.NewForecastClient(conn),
		rightsizingClient: pb.NewRightsizingClient(conn),
	}
}

func (c *Client) Forecast(ctx context.Context, data resource.TimeseriesData) (*pb.ForecastResponse, error) {
	datapoints := make([]*pb.TimeSeriesDatapoint, len(data))
	for i, point := range data {
		datapoints[i] = &pb.TimeSeriesDatapoint{
			Timestamp: point.Time,
			Value:     point.Value,
		}
	}

	request := &pb.ForecastRequest{
		Data: datapoints,
	}
	response, err := c.forecastClient.Forecast(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) Rightsizing(ctx context.Context, data resource.TimeseriesData) (*pb.RightsizingResponse, error) {
	datapoints := make([]float64, len(data))
	for i, point := range data {
		datapoints[i] = point.Value
	}

	request := &pb.RightsizingRequest{
		Data: datapoints,
	}

	response, err := c.rightsizingClient.Rightsizing(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}
