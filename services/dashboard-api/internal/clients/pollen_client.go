package clients

import (
	"context"
	"log/slog"
	"strings"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
)

type PollenClient struct {
	conn   *grpc.ClientConn
	client pb.PollenServiceClient
}

func NewPollenClient(ctx context.Context, address string) (*PollenClient, error) {
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
		slog.Error("Failed to create gRPC pollen client", "error", err, "address", address)
		return nil, err
	}
	client := pb.NewPollenServiceClient(conn)

	return &PollenClient{conn: conn, client: client}, nil
}

func (c *PollenClient) Close() error {
	return c.conn.Close()
}

func (c *PollenClient) GetPollenReports(ctx context.Context) ([]*pb.PollenReport, error) {
	resp, err := c.client.GetAllPollenReports(ctx, &pb.GetAllPollenReportsRequest{})
	if err != nil {
		slog.Error("Failed to get pollen reports", "error", err)
		return nil, err
	}
	return resp.Reports, nil
}

func (c *PollenClient) GetPollenReport(ctx context.Context, locationID string) (*pb.PollenReport, error) {
	resp, err := c.client.GetPollenReport(ctx, &pb.GetPollenReportRequest{LocationId: locationID})
	if err != nil {
		slog.Error("Failed to get pollen report", "error", err)
		return nil, err
	}
	return resp.Report, nil
}
