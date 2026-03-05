package shared

import "time"

// Intentional build break to test deploy failure notifications (issue #33)
var BuildBreaker = undefinedVariable

const (
	// Firestore databases used by each service.
	WeatherDatabaseID = "weather-log"
	PollenDatabaseID  = "pollen-log"

	// Cache collection names shared between each collector/provider pair.
	WeatherCacheCollection = "weather_cache"
	WeatherRawCollection   = "weather_raw"
	PollenCacheCollection  = "pollen_cache"
	PollenRawCollection    = "pollen_raw"

	// RPC timeouts
	RPCClientTimeout = 2 * time.Second
)
