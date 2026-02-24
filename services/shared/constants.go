package shared

const (
	// DatabaseID is the Firestore database used by all services.
	WeatherDatabaseID = "weather-log"
	PollenDatabaseID  = "pollen-log"

	// Cache collection names shared between each collector/provider pair.
	WeatherCacheCollection = "weather_cache"
	WeatherRawCollection   = "weather_raw"
	PollenCacheCollection  = "pollen_cache"
	PollenRawCollection    = "pollen_raw"
)
