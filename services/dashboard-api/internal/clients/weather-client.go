package clients

import (
	"context"
	"log/slog"
	"strings"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
)

type WeatherClient struct {
	conn   *grpc.ClientConn
	client pb.PressureStatsServiceClient
}

func NewWeatherClient(ctx context.Context, address string) (*WeatherClient, error) {
	var opts []grpc.DialOption

	if strings.HasSuffix(address, ":443") {
		// Cloud Run gRPC always uses port 443 with TLS + ID tokens.
		audience := "https://" + strings.TrimSuffix(address, ":443")

		// Create an ID Token source for the target audience
		tokenSource, err := idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			slog.Error("Failed to create token source", "error", err, "audience", audience)
			return nil, err
		}

		// Use system certs + ID token
		opts = append(opts,
			grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
			grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: tokenSource}),
		)
		slog.Info("Using Google ID Token authentication", "address", address, "audience", audience)
	} else {
		// Local development or Docker Compose: no TLS, no auth.
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		slog.Info("Using insecure gRPC credentials", "address", address)
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		slog.Error("Failed to create gRPC client", "error", err, "address", address)
		return nil, err
	}
	client := pb.NewPressureStatsServiceClient(conn)

	return &WeatherClient{conn: conn, client: client}, nil
}

func (c *WeatherClient) Close() error {
	return c.conn.Close()
}

func (c *WeatherClient) GetPressureStats(ctx context.Context) ([]*pb.PressureStat, error) {
	resp, err := c.client.GetAllPressureStats(ctx, &pb.GetAllPressureStatsRequest{})
	if err != nil {
		slog.Error("Failed to get weather stats", "error", err)
		return nil, err
	}
	return resp.Stats, nil
}

func (c *WeatherClient) GetPressureStat(ctx context.Context, locationId string) (*pb.PressureStat, error) {
	resp, err := c.client.GetPressureStats(ctx, &pb.GetPressureStatsRequest{LocationId: locationId})
	if err != nil {
		slog.Error("Failed to get weather stat", "error", err)
		return nil, err
	}
	return resp.Stat, nil
}
