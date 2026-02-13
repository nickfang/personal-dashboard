package clients

import (
	"context"
	"log/slog"

	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WeatherClient struct {
	conn   *grpc.ClientConn
	client pb.PressureStatsServiceClient
}

func NewWeatherClient(address string) (*WeatherClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to create gRPC client", "error", err)
		return nil, err
	}
	client := pb.NewPressureStatsServiceClient(conn)

	return &WeatherClient{conn: conn, client: client}, nil
}

func (c *WeatherClient) Close() error {
	return c.conn.Close()
}

func (c *WeatherClient) GetWeatherStats(ctx context.Context) ([]*pb.PressureStat, error) {
	resp, err := c.client.GetAllPressureStats(ctx, &pb.GetAllPressureStatsRequest{})
	if err != nil {
		slog.Error("Failed to get weather stats", "error", err)
		return nil, err
	}
	return resp.Stats, nil
}

func (c *WeatherClient) GetWeatherStat(ctx context.Context, locationId string) (*pb.PressureStat, error) {
	resp, err := c.client.GetPressureStats(ctx, &pb.GetPressureStatsRequest{LocationId: locationId})
	if err != nil {
		slog.Error("Failed to get weather stat", "error", err)
		return nil, err
	}
	return resp.Stat, nil
}
