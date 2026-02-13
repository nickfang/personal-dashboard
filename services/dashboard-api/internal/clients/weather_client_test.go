package clients

import (
	"context"
	"net"
	"testing"

	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
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

func TestWeatherClient_GetWeather(t *testing.T) {
	// 1. Setup in-memory gRPC server
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterPressureStatsServiceServer(s, &mockWeatherServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	defer s.Stop()

	// 2. Create the client using the in-memory connection
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// 3. Initialize your WeatherClient (This is what you will implement)
	// Note: In your implementation, NewWeatherClient should accept the address.
	// For this test, we wrap the mock connection.
	client := &WeatherClient{
		client: pb.NewPressureStatsServiceClient(conn),
	}

	// 4. Test the call
	resp, err := client.GetWeatherStat(context.Background(), "house-nick")
	if err != nil {
		t.Fatalf("GetWeather failed: %v", err)
	}

	if resp.LocationId != "house-nick" {
		t.Errorf("Expected location house-nick, got %s", resp.LocationId)
	}
	if resp.Trend != "rising" {
		t.Errorf("Expected trend rising, got %s", resp.Trend)
	}
}
