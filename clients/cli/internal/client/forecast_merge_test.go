package client

import "testing"

func forecastAt(issuedAt string, pressure float64) Forecast {
	return Forecast{
		LocationID: "house-nick",
		IssuedAt:   issuedAt,
		Points:     []ForecastPoint{{ValidTime: issuedAt, PressureMb: pressure}},
	}
}

func TestResponseMerge_ForecastNewerWinsAndCarriesAlerts(t *testing.T) {
	dst := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T00:00:00Z", 1010)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "old", Status: "active"}}},
	}
	src := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T06:00:00Z", 1008)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "new", Status: "active"}}},
	}

	dst.Merge(src)

	if dst.Forecast["house-nick"].IssuedAt != "2026-06-12T06:00:00Z" {
		t.Errorf("newer forecast should win, got %q", dst.Forecast["house-nick"].IssuedAt)
	}
	if len(dst.Alerts["house-nick"]) != 1 || dst.Alerts["house-nick"][0].ID != "new" {
		t.Errorf("alerts should travel with the winning forecast, got %v", dst.Alerts["house-nick"])
	}
}

func TestResponseMerge_ForecastOlderLosesAndKeepsAlerts(t *testing.T) {
	dst := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T06:00:00Z", 1008)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "current", Status: "active"}}},
	}
	src := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T00:00:00Z", 1010)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "stale", Status: "active"}}},
	}

	dst.Merge(src)

	if dst.Forecast["house-nick"].IssuedAt != "2026-06-12T06:00:00Z" {
		t.Errorf("older forecast must not clobber, got %q", dst.Forecast["house-nick"].IssuedAt)
	}
	if dst.Alerts["house-nick"][0].ID != "current" {
		t.Errorf("alerts must stay paired with the kept forecast, got %v", dst.Alerts["house-nick"])
	}
}

func TestResponseMerge_NewerForecastWithNoAlertsClearsThem(t *testing.T) {
	dst := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T00:00:00Z", 1010)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "expired", Status: "active"}}},
	}
	src := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T06:00:00Z", 1012)},
		Alerts:   map[string][]Alert{"house-nick": {}},
	}

	dst.Merge(src)

	if len(dst.Alerts["house-nick"]) != 0 {
		t.Errorf("a newer alert-free run should clear stale alerts, got %v", dst.Alerts["house-nick"])
	}
}

func TestResponseMerge_ForecastIntoEmptyResponse(t *testing.T) {
	dst := &Response{}
	src := &Response{
		Forecast: map[string]Forecast{"house-nick": forecastAt("2026-06-12T06:00:00Z", 1008)},
		Alerts:   map[string][]Alert{"house-nick": {{ID: "a1", Status: "active"}}},
	}

	dst.Merge(src)

	if len(dst.Forecast) != 1 || len(dst.Alerts["house-nick"]) != 1 {
		t.Errorf("nil maps should be initialized and filled, got %v / %v", dst.Forecast, dst.Alerts)
	}
}
