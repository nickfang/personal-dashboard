package shared

import "testing"

func TestWeatherDatabaseID(t *testing.T) {
	if WeatherDatabaseID != "weather-log" {
		t.Errorf("WeatherDatabaseID = %q, want %q", WeatherDatabaseID, "weather-log")
	}
}

func TestPollenDatabaseID(t *testing.T) {
	if PollenDatabaseID != "pollen-log" {
		t.Errorf("PollenDatabaseID = %q, want %q", PollenDatabaseID, "pollen-log")
	}
}

func TestWeatherCacheCollection(t *testing.T) {
	if WeatherCacheCollection != "weather_cache" {
		t.Errorf("WeatherCacheCollection = %q, want %q", WeatherCacheCollection, "weather_cache")
	}
}

func TestPollenCacheCollection(t *testing.T) {
	if PollenCacheCollection != "pollen_cache" {
		t.Errorf("PollenCacheCollection = %q, want %q", PollenCacheCollection, "pollen_cache")
	}
}

func TestDatabaseIDsAreDistinct(t *testing.T) {
	if WeatherDatabaseID == PollenDatabaseID {
		t.Error("WeatherDatabaseID and PollenDatabaseID must be distinct")
	}
}

func TestCollectionNamesAreDistinct(t *testing.T) {
	if WeatherCacheCollection == PollenCacheCollection {
		t.Error("WeatherCacheCollection and PollenCacheCollection must be distinct")
	}
}
