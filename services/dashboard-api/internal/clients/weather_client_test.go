package clients

import (
	"context"
	"net"
	"testing"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// mockWeatherServer implements the gRPC interface for testing
type mockWeatherServer struct {
	pb.UnimplementedPressureStatsServiceServer
}

func (m *mockWeatherServer) GetPressureStats(ctx context.Context, req *pb.GetPressureStatsRequest) (*pb.GetPressureStatsResponse, error) {
	return &pb.GetPressureStatsResponse{
		Stat: &pb.PressureStat{
			LocationId: req.LocationId,
			Trend:      "rising",
		},
	}, nil
}

func (m *mockWeatherServer) GetAllPressureStats(ctx context.Context, req *pb.GetAllPressureStatsRequest) (*pb.GetAllPressureStatsResponse, error) {
	return &pb.GetAllPressureStatsResponse{
		Stats: []*pb.PressureStat{
			{LocationId: "house-nick", Trend: "rising", Delta_1H: 0.5},
			{LocationId: "house-jane", Trend: "falling", Delta_1H: -0.3},
		},
	}, nil
}

// setupTestClient creates an in-memory gRPC server and returns a connected WeatherClient
func setupTestClient(t *testing.T) *WeatherClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterPressureStatsServiceServer(s, &mockWeatherServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	t.Cleanup(s.Stop)

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return &WeatherClient{
		conn:   conn,
		client: pb.NewPressureStatsServiceClient(conn),
	}
}

func TestWeatherClient_GetWeatherStat(t *testing.T) {
	client := setupTestClient(t)

	resp, err := client.GetWeatherStat(context.Background(), "house-nick")
	if err != nil {
		t.Fatalf("GetWeatherStat failed: %v", err)
	}

	if resp.LocationId != "house-nick" {
		t.Errorf("Expected location house-nick, got %s", resp.LocationId)
	}
	if resp.Trend != "rising" {
		t.Errorf("Expected trend rising, got %s", resp.Trend)
	}
}

func TestWeatherClient_GetWeatherStats(t *testing.T) {
	client := setupTestClient(t)

	stats, err := client.GetWeatherStats(context.Background())
	if err != nil {
		t.Fatalf("GetWeatherStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("Expected 2 stats, got %d", len(stats))
	}

	if stats[0].LocationId != "house-nick" {
		t.Errorf("Expected first location house-nick, got %s", stats[0].LocationId)
	}
	if stats[1].LocationId != "house-jane" {
		t.Errorf("Expected second location house-jane, got %s", stats[1].LocationId)
	}
}
