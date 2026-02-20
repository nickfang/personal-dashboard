package shared

import "testing"

func TestDatabaseID(t *testing.T) {
	if DatabaseID != "weather-log" {
		t.Errorf("DatabaseID = %q, want %q", DatabaseID, "weather-log")
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

func TestCollectionNamesAreDistinct(t *testing.T) {
	if WeatherCacheCollection == PollenCacheCollection {
		t.Error("WeatherCacheCollection and PollenCacheCollection must be distinct")
	}
}
