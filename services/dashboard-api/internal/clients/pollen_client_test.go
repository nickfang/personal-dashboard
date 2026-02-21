package clients

import (
	"context"
	"net"
	"testing"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// mockPollenServer implements the gRPC interface for testing
type mockPollenServer struct {
	pb.UnimplementedPollenServiceServer
}

func (m *mockPollenServer) GetAllPollenReports(ctx context.Context, req *pb.GetAllPollenReportsRequest) (*pb.GetAllPollenReportsResponse, error) {
	return &pb.GetAllPollenReportsResponse{
		Reports: []*pb.PollenReport{
			{
				LocationId:      "house-nick",
				OverallIndex:    4,
				OverallCategory: "High",
				DominantType:    "TREE",
				Types: []*pb.PollenType{
					{Code: "TREE", Index: 4, Category: "High", InSeason: true},
				},
				Plants: []*pb.PollenPlant{
					{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				},
			},
			{
				LocationId:      "house-nita",
				OverallIndex:    1,
				OverallCategory: "Very Low",
				DominantType:    "GRASS",
			},
		},
	}, nil
}

func (m *mockPollenServer) GetPollenReport(ctx context.Context, req *pb.GetPollenReportRequest) (*pb.GetPollenReportResponse, error) {
	return &pb.GetPollenReportResponse{
		Report: &pb.PollenReport{
			LocationId:      req.LocationId,
			OverallIndex:    4,
			OverallCategory: "High",
			DominantType:    "TREE",
			Types: []*pb.PollenType{
				{Code: "TREE", Index: 4, Category: "High", InSeason: true},
				{Code: "GRASS", Index: 1, Category: "Very Low", InSeason: false},
			},
			Plants: []*pb.PollenPlant{
				{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
			},
		},
	}, nil
}

// setupPollenTestClient creates an in-memory gRPC server and returns a connected PollenClient
func setupPollenTestClient(t *testing.T) *PollenClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterPollenServiceServer(s, &mockPollenServer{})
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

	return &PollenClient{
		conn:   conn,
		client: pb.NewPollenServiceClient(conn),
	}
}

func TestPollenClient_GetPollenReport(t *testing.T) {
	client := setupPollenTestClient(t)

	report, err := client.GetPollenReport(context.Background(), "house-nick")
	if err != nil {
		t.Fatalf("GetPollenReport failed: %v", err)
	}

	if report.LocationId != "house-nick" {
		t.Errorf("Expected location house-nick, got %s", report.LocationId)
	}
	if report.OverallIndex != 4 {
		t.Errorf("Expected OverallIndex 4, got %d", report.OverallIndex)
	}
	if report.DominantType != "TREE" {
		t.Errorf("Expected DominantType TREE, got %s", report.DominantType)
	}
	if len(report.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(report.Types))
	}
	if len(report.Plants) != 1 {
		t.Errorf("Expected 1 plant, got %d", len(report.Plants))
	}
}

func TestPollenClient_GetPollenReports(t *testing.T) {
	client := setupPollenTestClient(t)

	reports, err := client.GetPollenReports(context.Background())
	if err != nil {
		t.Fatalf("GetPollenReports failed: %v", err)
	}

	if len(reports) != 2 {
		t.Fatalf("Expected 2 reports, got %d", len(reports))
	}

	if reports[0].LocationId != "house-nick" {
		t.Errorf("Expected first location house-nick, got %s", reports[0].LocationId)
	}
	if reports[0].OverallIndex != 4 {
		t.Errorf("Expected first OverallIndex 4, got %d", reports[0].OverallIndex)
	}
	if reports[1].LocationId != "house-nita" {
		t.Errorf("Expected second location house-nita, got %s", reports[1].LocationId)
	}
}
