package shared

const (
	// Firestore databases used by each service.
	WeatherDatabaseID = "weather-log"
	PollenDatabaseID  = "pollen-log"

	// Cache collection names shared between each collector/provider pair.
	WeatherCacheCollection = "weather_cache"
	WeatherRawCollection   = "weather_raw"
	PollenCacheCollection  = "pollen_cache"
	PollenRawCollection    = "pollen_raw"
)
