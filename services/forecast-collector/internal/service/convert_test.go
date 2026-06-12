package service

import (
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
)

func validHour() api.ForecastHour {
	var h api.ForecastHour
	h.Interval.StartTime = time.Date(2026, 6, 12, 5, 0, 0, 0, time.UTC)
	h.Temperature.Degrees = 27.5
	h.FeelsLikeTemperature.Degrees = 31.6
	h.DewPoint.Degrees = 24.2
	h.RelativeHumidityPercent = 82
	h.UVIndex = 0
	h.AirPressure.MeanSeaLevelMillibars = 1012.65
	h.Wind.Direction.Degrees = 160
	h.Wind.Speed.Value = 13
	h.Wind.Gust.Value = 21
	h.Precipitation.Probability.Percent = 5
	return h
}

func TestMapToForecastPoint(t *testing.T) {
	p, err := MapToForecastPoint(validHour())
	if err != nil {
		t.Fatalf("MapToForecastPoint() returned error: %v", err)
	}

	if !p.ValidTime.Equal(time.Date(2026, 6, 12, 5, 0, 0, 0, time.UTC)) {
		t.Errorf("ValidTime = %v, want 2026-06-12T05:00:00Z", p.ValidTime)
	}
	if p.PressureMb != 1012.65 {
		t.Errorf("PressureMb = %v, want 1012.65", p.PressureMb)
	}
	if p.TempC != 27.5 {
		t.Errorf("TempC = %v, want 27.5", p.TempC)
	}
	wantF := (27.5 * 1.8) + 32
	if p.TempF != wantF {
		t.Errorf("TempF = %v, want %v", p.TempF, wantF)
	}
	if p.TempFeelC != 31.6 || p.DewpointC != 24.2 {
		t.Errorf("TempFeelC/DewpointC = %v/%v, want 31.6/24.2", p.TempFeelC, p.DewpointC)
	}
	if p.HumidityPercent != 82 || p.PrecipitationPercent != 5 {
		t.Errorf("Humidity/Precipitation = %v/%v, want 82/5", p.HumidityPercent, p.PrecipitationPercent)
	}
	if p.WindDirDeg != 160 || p.WindSpeedKph != 13 || p.WindGustKph != 21 {
		t.Errorf("Wind = %v/%v/%v, want 160/13/21", p.WindDirDeg, p.WindSpeedKph, p.WindGustKph)
	}
}

func TestMapToForecastPoint_ZeroPressureRejected(t *testing.T) {
	h := validHour()
	h.AirPressure.MeanSeaLevelMillibars = 0

	_, err := MapToForecastPoint(h)
	if err == nil {
		t.Fatal("MapToForecastPoint() should reject 0.0 pressure")
	}
}

func TestMapRun_SkipsInvalidHours(t *testing.T) {
	bad := validHour()
	bad.AirPressure.MeanSeaLevelMillibars = 0

	points := MapRun([]api.ForecastHour{validHour(), bad, validHour()})

	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2 (invalid hour skipped)", len(points))
	}
}

func TestMapRun_Empty(t *testing.T) {
	if points := MapRun(nil); len(points) != 0 {
		t.Fatalf("MapRun(nil) = %d points, want 0", len(points))
	}
}
