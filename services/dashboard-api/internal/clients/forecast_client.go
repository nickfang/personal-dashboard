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

// ForecastClient wraps the ForecastService RPC, served by weather-provider
// at the same address as PressureStatsService.
type ForecastClient struct {
	conn   *grpc.ClientConn
	client pb.ForecastServiceClient
}

func NewForecastClient(ctx context.Context, address string) (*ForecastClient, error) {
	var opts []grpc.DialOption

	if strings.HasSuffix(address, ":443") {
		// Cloud Run gRPC always uses port 443 with TLS + ID tokens.
		audience := "https://" + strings.TrimSuffix(address, ":443")

		tokenSource, err := idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			slog.Error("Failed to create token source", "error", err, "audience", audience)
			return nil, err
		}

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
		slog.Error("Failed to create gRPC forecast client", "error", err, "address", address)
		return nil, err
	}

	return &ForecastClient{conn: conn, client: pb.NewForecastServiceClient(conn)}, nil
}

func (c *ForecastClient) Close() error {
	return c.conn.Close()
}

func (c *ForecastClient) GetForecast(ctx context.Context, locationId string) (*pb.Forecast, error) {
	resp, err := c.client.GetForecast(ctx, &pb.GetForecastRequest{LocationId: locationId})
	if err != nil {
		slog.Error("Failed to get forecast", "error", err, "location", locationId)
		return nil, err
	}
	return resp.Forecast, nil
}

func (c *ForecastClient) GetAllForecasts(ctx context.Context) ([]*pb.Forecast, error) {
	resp, err := c.client.GetAllForecasts(ctx, &pb.GetAllForecastsRequest{})
	if err != nil {
		slog.Error("Failed to get forecasts", "error", err)
		return nil, err
	}
	return resp.Forecasts, nil
}
