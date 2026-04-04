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

	if !strings.Contains(text, "Deltas: 0.00(1h), 0.00(3h), 0.00(6h) 0.00(12h) 0.00(24h)") {
		t.Errorf("expected zero deltas to be included, got:\n%s", text)
	}
}

// --- formatWeatherText tests ---

func TestFormatWeatherText(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	localFormatted := fixedTime.Local().Format("2006.01.02 15:04:05")

	weathers := []*weatherPb.Weather{
		{
			LocationId:           "house-nick",
			TempC:                22.50,
			TempF:                72.50,
			TempFeelC:            21.00,
			TempFeelF:            69.80,
			HumidityPercent:      65.00,
			PressureMb:           1013.25,
			PrecipitationPercent: 10.00,
			LastUpdated:          timestamppb.New(fixedTime),
		},
	}

	result := formatWeatherText(weathers)

	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	// Timestamp formatted as local time
	if !strings.Contains(text, fmt.Sprintf("Weather: %s", localFormatted)) {
		t.Errorf("expected weather timestamp in local time, got:\n%s", text)
	}

	// Temperature
	if !strings.Contains(text, "Temp: 72.50F") {
		t.Errorf("expected temperature, got:\n%s", text)
	}

	// Feels Like
	if !strings.Contains(text, "Feels Like: 69.80F") {
		t.Errorf("expected feels like, got:\n%s", text)
	}

	// Humidity
	if !strings.Contains(text, "Humidity: 65%") {
		t.Errorf("expected humidity, got:\n%s", text)
	}

	// Pressure
	if !strings.Contains(text, "Pressure: 1013.25mb") {
		t.Errorf("expected pressure, got:\n%s", text)
	}

	// Precipitation
	if !strings.Contains(text, "Precipitation: 10%") {
		t.Errorf("expected precipitation, got:\n%s", text)
	}
}

func TestFormatWeatherText_MultipleLocations(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	weathers := []*weatherPb.Weather{
		{LocationId: "house-nick", TempC: 22.5, TempF: 72.5, LastUpdated: fixedTime},
		{LocationId: "house-mom", TempC: 18.0, TempF: 64.4, LastUpdated: fixedTime},
	}

	result := formatWeatherText(weathers)

	if _, ok := result["house-nick"]; !ok {
		t.Error("expected 'house-nick' key in result map")
	}
	if _, ok := result["house-mom"]; !ok {
		t.Error("expected 'house-mom' key in result map")
	}
	if !strings.Contains(result["house-nick"], "72.50F") {
		t.Errorf("expected house-nick temp, got:\n%s", result["house-nick"])
	}
	if !strings.Contains(result["house-mom"], "64.40F") {
		t.Errorf("expected house-mom temp, got:\n%s", result["house-mom"])
	}
}

func TestFormatWeatherText_ZeroValues(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	weathers := []*weatherPb.Weather{
		{LocationId: "house-nick", LastUpdated: fixedTime},
	}

	result := formatWeatherText(weathers)
	text := result["house-nick"]

	if !strings.Contains(text, "Temp: 0.00F") {
		t.Errorf("expected zero temps, got:\n%s", text)
	}
	if !strings.Contains(text, "Humidity: 0%") {
		t.Errorf("expected zero humidity, got:\n%s", text)
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
			Plants: []*pollenPb.PollenPlant{
				{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				{DisplayName: "Birch", Index: 2, Category: "Low", InSeason: false},
				{DisplayName: "Oak", Index: 0, Category: "None", InSeason: false},
			},
		},
	}

	result := formatPollenText(pollenReports)

	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	// Header with timestamp
	if !strings.Contains(text, fmt.Sprintf("Pollen: %s", localFormatted)) {
		t.Errorf("expected pollen timestamp in local time, got:\n%s", text)
	}

	// Plants with Index > 0 are included
	if !strings.Contains(text, "Juniper (In Season)") {
		t.Errorf("expected Juniper as In Season, got:\n%s", text)
	}
	if !strings.Contains(text, "Birch (Out of Season)") {
		t.Errorf("expected Birch as Out of Season, got:\n%s", text)
	}

	// Plants with Index < 1 are excluded (break stops iteration)
	if strings.Contains(text, "Oak") {
		t.Errorf("expected Oak (Index=0) to be excluded, got:\n%s", text)
	}
}

func TestFormatPollenText_GroupedByIndex(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: fixedTime,
			Plants: []*pollenPb.PollenPlant{
				{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				{DisplayName: "Elm", Index: 4, Category: "High", InSeason: false},
				{DisplayName: "Maple", Index: 2, Category: "Low", InSeason: true},
				{DisplayName: "Oak", Index: 2, Category: "Low", InSeason: false},
			},
		},
	}

	result := formatPollenText(pollenReports)
	text := result["house-nick"]

	// Plants with the same index should be on the same line with a category label
	lines := strings.Split(text, "\n")
	foundHighGroup := false
	foundLowGroup := false
	for _, line := range lines {
		if strings.Contains(line, "High") && strings.Contains(line, "Juniper") && strings.Contains(line, "Elm") {
			foundHighGroup = true
		}
		if strings.Contains(line, "Low") && strings.Contains(line, "Maple") && strings.Contains(line, "Oak") {
			foundLowGroup = true
		}
	}

	if !foundHighGroup {
		t.Errorf("expected Juniper and Elm grouped on same line under High, got:\n%s", text)
	}
	if !foundLowGroup {
		t.Errorf("expected Maple and Oak grouped on same line under Low, got:\n%s", text)
	}

	// Groups are on separate lines
	highLineIdx := strings.Index(text, "High")
	lowLineIdx := strings.Index(text, "Low")
	if highLineIdx == -1 || lowLineIdx == -1 {
		t.Fatalf("expected both High and Low groups, got:\n%s", text)
	}
	if highLineIdx > lowLineIdx {
		t.Errorf("expected High group before Low group, got:\n%s", text)
	}
}

func TestFormatPollenText_SortedByIndexDescending(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: fixedTime,
			Plants: []*pollenPb.PollenPlant{
				{DisplayName: "Oak", Index: 1, Category: "Very Low", InSeason: false},
				{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
				{DisplayName: "Birch", Index: 2, Category: "Low", InSeason: true},
			},
		},
	}

	result := formatPollenText(pollenReports)
	text := result["house-nick"]

	// Higher index should appear first
	juniperIdx := strings.Index(text, "Juniper")
	birchIdx := strings.Index(text, "Birch")
	oakIdx := strings.Index(text, "Oak")

	if juniperIdx == -1 || birchIdx == -1 || oakIdx == -1 {
		t.Fatalf("expected all 3 plants in output, got:\n%s", text)
	}

	if juniperIdx > birchIdx {
		t.Errorf("expected Juniper (index=4) before Birch (index=2)")
	}
	if birchIdx > oakIdx {
		t.Errorf("expected Birch (index=2) before Oak (index=1)")
	}
}

func TestFormatPollenText_EmptyPlants(t *testing.T) {
	fixedTime := timestamppb.New(time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC))

	pollenReports := []*pollenPb.PollenReport{
		{
			LocationId:  "house-nick",
			CollectedAt: fixedTime,
			Plants:      []*pollenPb.PollenPlant{},
		},
	}

	result := formatPollenText(pollenReports)
	text, ok := result["house-nick"]
	if !ok {
		t.Fatal("expected 'house-nick' key in result map")
	}

	if text != "No pollen data available" {
		t.Errorf("expected 'No pollen data available', got:\n%s", text)
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
			Plants: []*pollenPb.PollenPlant{
				{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
			},
		},
	}

	lastWeathers := []*weatherPb.Weather{
		{
			LocationId:  "house-nick",
			TempC:       22.5,
			TempF:       72.5,
			LastUpdated: timestamppb.New(fixedTime),
		},
	}

	result, err := formatDashboardText(pressureStats, pollenReports, lastWeathers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Location separator
	if !strings.Contains(result, "---------------- house-nick ----------------") {
		t.Errorf("expected location separator, got:\n%s", result)
	}

	// Weather, pressure and pollen sections combined
	if !strings.Contains(result, fmt.Sprintf("Weather: %s", localFormatted)) {
		t.Errorf("expected weather section, got:\n%s", result)
	}
	if !strings.Contains(result, fmt.Sprintf("Pressure: %s", localFormatted)) {
		t.Errorf("expected pressure section, got:\n%s", result)
	}
	if !strings.Contains(result, fmt.Sprintf("Pollen: %s", localFormatted)) {
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
		{LocationId: "house-nick", CollectedAt: fixedTime, Plants: []*pollenPb.PollenPlant{{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true}}},
		{LocationId: "house-mom", CollectedAt: fixedTime, Plants: []*pollenPb.PollenPlant{{DisplayName: "Oak", Index: 2, Category: "Low", InSeason: false}}},
	}

	lastWeathers := []*weatherPb.Weather{
		{LocationId: "house-nick", TempC: 22.5, TempF: 72.5, LastUpdated: fixedTime},
		{LocationId: "house-mom", TempC: 18.0, TempF: 64.4, LastUpdated: fixedTime},
	}

	result, err := formatDashboardText(pressureStats, pollenReports, lastWeathers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "---------------- house-nick ----------------") {
		t.Errorf("expected house-nick separator, got:\n%s", result)
	}
	if !strings.Contains(result, "---------------- house-mom ----------------") {
		t.Errorf("expected house-mom separator, got:\n%s", result)
	}
	if !strings.Contains(result, "rising") {
		t.Errorf("expected 'rising' trend for house-nick, got:\n%s", result)
	}
	if !strings.Contains(result, "falling") {
		t.Errorf("expected 'falling' trend for house-mom, got:\n%s", result)
	}
	if !strings.Contains(result, "72.50F") {
		t.Errorf("expected house-nick weather temp, got:\n%s", result)
	}
	if !strings.Contains(result, "64.40F") {
		t.Errorf("expected house-mom weather temp, got:\n%s", result)
	}
}
