package handlers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	weatherPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- formatPressureText tests ---

func TestFormatPressureText(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	localFormatted := fixedTime.Local().Format("2006.01.02 15:04:05")

	pressureStats := []*weatherPb.PressureStat{
		{
			LocationId:  "house-nick",
			Trend:       "rising",
			Delta_1H:    0.50,
			Delta_3H:    1.25,
			Delta_6H:    -0.75,
			Delta_12H:   2.00,
			Delta_24H:   3.10,
			LastUpdated: timestamppb.New(fixedTime),
		},
	}

	result := formatPressureText(pressureStats)

	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	// Location header
	if !strings.Contains(text, "house-nick\n") {
		t.Errorf("expected location header, got:\n%s", text)
	}

	// Timestamp formatted as local time
	if !strings.Contains(text, fmt.Sprintf("Pressure: %s", localFormatted)) {
		t.Errorf("expected pressure timestamp in local time, got:\n%s", text)
	}

	// Trend
	if !strings.Contains(text, "  rising\n") {
		t.Errorf("expected trend, got:\n%s", text)
	}

	// All 5 deltas formatted to 2 decimal places
	if !strings.Contains(text, "Deltas: 0.50(1h), 1.25(3h), -0.75(6h) 2.00(12h) 3.10(24h)") {
		t.Errorf("expected all 5 formatted deltas, got:\n%s", text)
	}
}

func TestFormatPressureText_MultipleLocations(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pressureStats := []*weatherPb.PressureStat{
		{LocationId: "house-nick", Trend: "rising", Delta_1H: 0.5, Delta_3H: 1.0, Delta_6H: 1.5, LastUpdated: fixedTime},
		{LocationId: "house-mom", Trend: "falling", Delta_1H: -0.3, Delta_3H: -0.8, Delta_6H: -1.2, LastUpdated: fixedTime},
	}

	result := formatPressureText(pressureStats)

	if _, ok := result["house-nick"]; !ok {
		t.Error("expected 'house-nick' key in result map")
	}
	if _, ok := result["house-mom"]; !ok {
		t.Error("expected 'house-mom' key in result map")
	}
	if !strings.Contains(result["house-nick"], "rising") {
		t.Errorf("expected 'rising' trend for house-nick, got:\n%s", result["house-nick"])
	}
	if !strings.Contains(result["house-mom"], "falling") {
		t.Errorf("expected 'falling' trend for house-mom, got:\n%s", result["house-mom"])
	}
}

func TestFormatPressureText_ZeroDeltas(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pressureStats := []*weatherPb.PressureStat{
		{LocationId: "house-nick", Trend: "steady", LastUpdated: fixedTime},
	}

	result := formatPressureText(pressureStats)
	text := result["house-nick"]

	// Zero-value deltas should still appear
	if !strings.Contains(text, "Deltas: 0.00(1h), 0.00(3h), 0.00(6h) 0.00(12h) 0.00(24h)") {
		t.Errorf("expected zero deltas to be included, got:\n%s", text)
	}
}

// --- formatPollenText tests ---

func TestFormatPollenText(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	localFormatted := fixedTime.Local().Format("2006.01.02 15:04:05")

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: timestamppb.New(fixedTime),
			Types: []*pollenPb.PollenType{
				{Code: "TREE", Index: 4, Category: "High", InSeason: true},
				{Code: "GRASS", Index: 0, Category: "None", InSeason: false},
				{Code: "WEED", Index: 2, Category: "Low", InSeason: false},
			},
			Plants: []*pollenPb.PollenPlant{
				{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				{Code: "OAK", DisplayName: "Oak", Index: 0, Category: "None", InSeason: false},
				{Code: "BIRCH", DisplayName: "Birch", Index: 2, Category: "Low", InSeason: false},
			},
		},
	}

	result := formatPollenText(pollenReports)

	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	// Timestamp formatted as local time
	if !strings.Contains(text, fmt.Sprintf("Plants: %s", localFormatted)) {
		t.Errorf("expected plants timestamp in local time, got:\n%s", text)
	}

	// Plants: InSeason true → "In Season"
	if !strings.Contains(text, "Juniper (High): 4 - In Season") {
		t.Errorf("expected Juniper as In Season, got:\n%s", text)
	}

	// Plants: InSeason false → "Out of Season"
	if !strings.Contains(text, "Birch (Low): 2 - Out of Season") {
		t.Errorf("expected Birch as Out of Season, got:\n%s", text)
	}

	// Plants: Index == 0 are excluded
	if strings.Contains(text, "Oak") {
		t.Errorf("expected Oak (Index=0) to be excluded, got:\n%s", text)
	}

	// Types: Index > 0 are included
	if !strings.Contains(text, "TREE:") {
		t.Errorf("expected TREE type in output, got:\n%s", text)
	}
	if !strings.Contains(text, "WEED:") {
		t.Errorf("expected WEED type in output, got:\n%s", text)
	}

	// Types: Index == 0 are excluded
	if strings.Contains(text, "GRASS") {
		t.Errorf("expected GRASS type (Index=0) to be excluded, got:\n%s", text)
	}
}

func TestFormatPollenText_PlantSortOrder(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: fixedTime,
			Plants: []*pollenPb.PollenPlant{
				{DisplayName: "Oak", Index: 1, Category: "Low", InSeason: false},
				{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				{DisplayName: "Birch", Index: 3, Category: "Medium", InSeason: true},
				{DisplayName: "Ragweed", Index: 2, Category: "Low", InSeason: false},
			},
		},
	}

	result := formatPollenText(pollenReports)
	text := result["house-nick"]

	// In-season plants (Juniper=4, Birch=3) should come before out-of-season (Ragweed=2, Oak=1)
	// Within each group, higher index first
	juniperIdx := strings.Index(text, "Juniper")
	birchIdx := strings.Index(text, "Birch")
	ragweedIdx := strings.Index(text, "Ragweed")
	oakIdx := strings.Index(text, "Oak")

	if juniperIdx == -1 || birchIdx == -1 || ragweedIdx == -1 || oakIdx == -1 {
		t.Fatalf("expected all 4 plants in output, got:\n%s", text)
	}

	if juniperIdx > birchIdx {
		t.Errorf("expected Juniper (index=4, in-season) before Birch (index=3, in-season)")
	}
	if birchIdx > ragweedIdx {
		t.Errorf("expected Birch (in-season) before Ragweed (out-of-season)")
	}
	if ragweedIdx > oakIdx {
		t.Errorf("expected Ragweed (index=2) before Oak (index=1)")
	}
}

func TestFormatPollenText_EmptyPlantsAndTypes(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: fixedTime,
			Plants:      []*pollenPb.PollenPlant{},
			Types:       []*pollenPb.PollenType{},
		},
	}

	result := formatPollenText(pollenReports)
	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	// Should still have the Plants header and Type section
	if !strings.Contains(text, "Plants:") {
		t.Errorf("expected Plants header even with empty data, got:\n%s", text)
	}
	if !strings.Contains(text, "Type:") {
		t.Errorf("expected Type header even with empty data, got:\n%s", text)
	}
}

// --- formatDashboardText tests ---

func TestFormatDashboardText(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	localFormatted := fixedTime.Local().Format("2006.01.02 15:04:05")

	pressureStats := []*weatherPb.PressureStat{
		{
			LocationId:  "house-nick",
			Trend:       "rising",
			Delta_1H:    0.50,
			Delta_3H:    1.25,
			Delta_6H:    -0.75,
			Delta_12H:   2.00,
			Delta_24H:   3.10,
			LastUpdated: timestamppb.New(fixedTime),
		},
	}

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: timestamppb.New(fixedTime),
			Types: []*pollenPb.PollenType{
				{Code: "TREE", Index: 4, Category: "High", InSeason: true},
			},
			Plants: []*pollenPb.PollenPlant{
				{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
			},
		},
	}

	result, err := formatDashboardText(pressureStats, pollenReports)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pressure and pollen sections combined for the location
	if !strings.Contains(result, "house-nick\n") {
		t.Errorf("expected location header, got:\n%s", result)
	}
	if !strings.Contains(result, fmt.Sprintf("Pressure: %s", localFormatted)) {
		t.Errorf("expected pressure section, got:\n%s", result)
	}
	if !strings.Contains(result, fmt.Sprintf("Plants: %s", localFormatted)) {
		t.Errorf("expected pollen section, got:\n%s", result)
	}
}

func TestFormatDashboardText_MultipleLocations(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pressureStats := []*weatherPb.PressureStat{
		{LocationId: "house-nick", Trend: "rising", Delta_1H: 0.5, Delta_3H: 1.0, Delta_6H: 1.5, LastUpdated: fixedTime},
		{LocationId: "house-mom", Trend: "falling", Delta_1H: -0.3, Delta_3H: -0.8, Delta_6H: -1.2, LastUpdated: fixedTime},
	}

	pollenReports := []*pollenPb.PollenReport{
		{LocationId: "house-nick", CollectedAt: fixedTime, Types: []*pollenPb.PollenType{{Code: "TREE", Index: 3, Category: "Medium", InSeason: true}}},
		{LocationId: "house-mom", CollectedAt: fixedTime, Types: []*pollenPb.PollenType{{Code: "GRASS", Index: 1, Category: "Low", InSeason: false}}},
	}

	result, err := formatDashboardText(pressureStats, pollenReports)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "house-nick") {
		t.Errorf("expected house-nick location, got:\n%s", result)
	}
	if !strings.Contains(result, "house-mom") {
		t.Errorf("expected house-mom location, got:\n%s", result)
	}
	if !strings.Contains(result, "rising") {
		t.Errorf("expected 'rising' trend for house-nick, got:\n%s", result)
	}
	if !strings.Contains(result, "falling") {
		t.Errorf("expected 'falling' trend for house-mom, got:\n%s", result)
	}
}
