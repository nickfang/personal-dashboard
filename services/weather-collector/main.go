package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	MAX_HISTORY_POINTS       = 48
	DELTA_TOLERANCE          = 45 * time.Minute
	DELTA_NOISE_THRESHOLD    = 0.5 // mb
	WEATHER_RAW_COLLECTION   = "weather_raw"
	WEATHER_CACHE_COLLECTION = "weather_cache"
)

type WeatherLocation struct {
	ID   string
	lat  float64
	long float64
}

type WeatherPoint struct {
	Location  string    `firestore:"location"`
	Timestamp time.Time `firestore:"timestamp"`

	HumidityPercent      int     `firestore:"humidity_pct"`
	PrecipitationPercent int     `firestore:"precipitation_pct"`
	UVIndex              int     `firestore:"uv_index"`
	PressureMb           float64 `firestore:"pressure_mb"`
	WindDirDeg           int     `firestore:"wind_dir_deg"`

	TempC        float64 `firestore:"temp_c"`
	TempFeelC    float64 `firestore:"temp_feel_c"`
	DewpointC    float64 `firestore:"dewpoint_c"`
	WindSpeedKph float64 `firestore:"wind_speed_kph"`
	WindGustKph  float64 `firestore:"wind_gust_kph"`
	VisibilityKm float64 `firestore:"visibility_km"`

	TempF        float64 `firestore:"temp_f"`
	TempFeelF    float64 `firestore:"temp_feel_f"`
	WindSpeedMph float64 `firestore:"wind_speed_mph"`
	WindGustMph  float64 `firestore:"wind_gust_mph"`
	VisibilityM  float64 `firestore:"visibility_miles"`
	DewpointF    float64 `firestore:"dewpoint_f"`
}

type PressurePoint struct {
	TimeStamp       time.Time `firestore:"timestamp"`
	HumidityPercent int       `firestore:"humidity_pct"`
	PressureMb      float64   `firestore:"pressure_mb"`

	TempC     float64 `firestore:"temp_c"`
	TempFeelC float64 `firestore:"temp_feel_c"`
	DewpointC float64 `firestore:"dewpoint_c"`

	TempF     float64 `firestore:"temp_f"`
	TempFeelF float64 `firestore:"temp_feel_f"`
	DewpointF float64 `firestore:"dewpoint_f"`
}

type PressureStats struct {
	// Pointers are used for Delta fields to support a true "N/A" (nil) state.
	// This allows the dashboard to distinguish between a 0.0 change and missing data,
	// avoiding "Data Lies" where gaps are incorrectly represented as stable trends.
	Timestamp time.Time `firestore:"timestamp"`
	Delta1h   *float64  `firestore:"delta_01h"`
	Delta3h   *float64  `firestore:"delta_3h"`
	Delta6h   *float64  `firestore:"delta_6h"`
	Delta12h  *float64  `firestore:"delta_12h"`
	Delta24h  *float64  `firestore:"delta_24h"`
	Trend     string    `firestore:"trend"`
}

type CacheDoc struct {
	LastUpdated  time.Time       `firestore:"last_updated"`
	CurrentValue WeatherPoint    `firestore:"current"`
	Analysis     PressureStats   `firestore:"analysis"`
	History      []PressurePoint `firestore:"history"`
}

var locations = []WeatherLocation{
	{
		ID:   "house-nick",
		lat:  30.260543381977474,
		long: -97.66768538740229,
	},
	{
		ID:   "house-nita",
		lat:  30.29420179895202,
		long: -97.6958691874014,
	},
	{
		ID:   "distribution-hall",
		lat:  30.261932944618565,
		long: -97.72816923158192,
	},
}

// WeatherAPIResponse matches the Google Weather API JSON structure
type WeatherAPIResponse struct {
	Temperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"temperature"`
	FeelsLikeTemperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"feelsLikeTemperature"`
	RelativeHumidityPercent int `json:"relativeHumidity"`
	UVIndex                 int `json:"uvIndex"`
	AirPressure             struct {
		MeanSeaLevelMillibars float64 `json:"meanSeaLevelMillibars"`
	} `json:"airPressure"`
	Wind struct {
		Direction struct {
			Degrees int `json:"degrees"`
		} `json:"direction"`
		Speed struct {
			Value float64 `json:"value"`
		} `json:"speed"`
		Gust struct {
			Value float64 `json:"value"`
		} `json:"gust"`
	} `json:"wind"`
	Visibility struct {
		Distance float64 `json:"distance"`
	} `json:"visibility"`
	DewPoint struct {
		Degrees float64 `json:"degrees"`
	} `json:"dewPoint"`
	Precipitation struct {
		Probability struct {
			Percent int    `json:"probability"`
			Type    string `json:"type"`
		} `json:"probability"`
	} `json:"precipitation"`
}

func main() {
	// Setup Structured Logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if os.Getenv("DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")

	if apiKey == "" || projectID == "" {
		slog.Error("Missing required env vars", "vars", "GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
		os.Exit(1)
	}

	client, err := firestore.NewClientWithDatabase(ctx, projectID, "weather-log")
	if err != nil {
		slog.Error("Failed to create firestore client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	for _, loc := range locations {
		wp, err := fetchWeatherWithRetry(apiKey, loc)
		if err != nil {
			// Structured logging for GCP Error Reporting
			slog.Error("Failed to fetch/validate weather after retries",
				"location", loc.ID,
				"error", err,
			)
			continue
		}

		// 1. Save to Raw Archive
		err = saveRawWeatherData(ctx, client, wp)
		if err != nil {
			slog.Error("Error saving raw weather data", "location", loc.ID, "error", err)
			continue
		}

		// 2. Update Hot Cache (Transaction)
		cacheRef := client.Collection(WEATHER_CACHE_COLLECTION).Doc(loc.ID)
		err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			cache, err := getUpdatedCacheDoc(cacheRef, wp, tx)
			if err != nil {
				return err
			}

			return tx.Set(cacheRef, cache)
		})

		if err != nil {
			slog.Error("Error updating cache", "location", loc.ID, "error", err)
		}

		slog.Info("Processed weather", "location", loc.ID)
	}
}

func fetchWeatherWithRetry(apiKey string, loc WeatherLocation) (*WeatherPoint, error) {
	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i <= len(backoffs); i++ {
		wp, err := fetchWeather(apiKey, loc)
		if err == nil {
			return wp, nil
		}

		lastErr = err
		if i < len(backoffs) {
			time.Sleep(backoffs[i])
		}
	}
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

func fetchWeather(apiKey string, loc WeatherLocation) (*WeatherPoint, error) {
	baseUrl := "https://weather.googleapis.com/v1/currentConditions:lookup"
	queryParams := url.Values{
		"key":                {apiKey},
		"location.latitude":  {fmt.Sprintf("%f", loc.lat)},
		"location.longitude": {fmt.Sprintf("%f", loc.long)},
		// "unitsSystem":        {"imperial"},
	}
	url := baseUrl + "?" + queryParams.Encode()
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	var data WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return mapToWeatherPoint(loc.ID, data)
}

func CtoF(c float64) float64 {
	return (c * 1.8) + 32
}

func KtoM(k float64) float64 {
	return k * 0.621371
}

func mapToWeatherPoint(locationID string, data WeatherAPIResponse) (*WeatherPoint, error) {
	// Strict Data Validation:
	// If pressure is 0.0, we assume the API response is incomplete or corrupted.
	// Saving a 0.0 pressure reading ruins statistical analysis (deltas).
	if data.AirPressure.MeanSeaLevelMillibars == 0 {
		return nil, fmt.Errorf("invalid pressure data (0.0) received for %s. Full payload: %+v", locationID, data)
	}

	wp := &WeatherPoint{
		Location:             locationID,
		Timestamp:            time.Now(),
		TempC:                data.Temperature.Degrees,
		TempF:                CtoF(data.Temperature.Degrees),
		TempFeelC:            data.FeelsLikeTemperature.Degrees,
		TempFeelF:            CtoF(data.FeelsLikeTemperature.Degrees),
		HumidityPercent:      data.RelativeHumidityPercent,
		UVIndex:              data.UVIndex,
		PressureMb:           data.AirPressure.MeanSeaLevelMillibars,
		WindDirDeg:           data.Wind.Direction.Degrees,
		WindSpeedKph:         data.Wind.Speed.Value,
		WindSpeedMph:         KtoM(data.Wind.Speed.Value),
		WindGustKph:          data.Wind.Gust.Value,
		WindGustMph:          KtoM(data.Wind.Gust.Value),
		VisibilityKm:         data.Visibility.Distance,
		VisibilityM:          KtoM(data.Visibility.Distance),
		DewpointC:            data.DewPoint.Degrees,
		DewpointF:            CtoF(data.DewPoint.Degrees),
		PrecipitationPercent: data.Precipitation.Probability.Percent,
	}

	// Structured Debug Log - Contains all mapping info for troubleshooting
	slog.Debug("Mapped Weather Data [DB Format]",
		"location", locationID,
		"timestamp", wp.Timestamp,
		"temp_c", wp.TempC,
		"feels_like_c", wp.TempFeelC,
		"humidity_pct", wp.HumidityPercent,
		"uv_index", wp.UVIndex,
		"pressure_mb", wp.PressureMb,
		"wind_dir_deg", wp.WindDirDeg,
		"wind_speed_kph", wp.WindSpeedKph,
		"wind_gust_kph", wp.WindGustKph,
		"visibility_km", wp.VisibilityKm,
		"dewpoint_c", wp.DewpointC,
		"precipitation_pct", wp.PrecipitationPercent,
	)

	return wp, nil
}

func calculatePressureStats(history []PressurePoint) PressureStats {
	stats := PressureStats{Trend: "unknown"}

	if len(history) < 2 {
		return stats
	}

	current := history[len(history)-1]

	// Log audit info once per location
	type deltaAudit struct {
		Target string   `json:"target"`
		Found  string   `json:"found,omitempty"`
		Delta  *float64 `json:"delta,omitempty"`
	}
	audit := make(map[string]deltaAudit)

	// getDelta uses a Time-Window Search instead of array indices.
	// This decouples logic from the sampling rate and makes it resilient to
	// missing data points or job scheduling jitter.
	getDelta := func(hoursAgo int) *float64 {
		targetTime := current.TimeStamp.Add(time.Duration(-hoursAgo) * time.Hour)
		// 45 minute tolerance allows us to find the closest point even if
		// some cycles were missed or delayed.
		tolerance := DELTA_TOLERANCE

		var bestMatch *PressurePoint
		minDiff := tolerance + (1 * time.Second)

		for i := len(history) - 2; i >= 0; i-- {
			p := &history[i]

			diff := p.TimeStamp.Sub(targetTime)
			if diff < 0 {
				diff = -diff
			}

			if diff <= tolerance {
				if diff < minDiff {
					minDiff = diff
					bestMatch = p
				}
			}

			// Optimization: History is sorted ascending; if we are way before
			// the target window, we can safely stop searching.
			if targetTime.Sub(p.TimeStamp) > tolerance {
				break
			}
		}

		key := fmt.Sprintf("%dh", hoursAgo)
		entry := deltaAudit{Target: targetTime.Format(time.RFC3339)}

		if bestMatch != nil {
			entry.Found = bestMatch.TimeStamp.Format(time.RFC3339)
			res := current.PressureMb - bestMatch.PressureMb
			entry.Delta = &res
			audit[key] = entry
			return &res
		}

		audit[key] = entry
		return nil
	}

	stats.Delta1h = getDelta(1)
	stats.Delta3h = getDelta(3)
	stats.Delta6h = getDelta(6)
	stats.Delta12h = getDelta(12)
	stats.Delta24h = getDelta(24)

	// Log the timestamp and value audit info at INFO level
	slog.Info("Pressure Analysis Diagnostics",
		"location", current.TimeStamp,
		"current_time", current.TimeStamp.Format(time.RFC3339),
		"analysis", audit,
	)

	// Simple trend logic with noise threshold
	// NOTE: We rely exclusively on the 3-Hour Delta (Delta3h) to define the "Trend" string.
	// This adheres to the World Meteorological Organization (WMO) definition of "Barometric Tendency".
	if stats.Delta3h != nil {
		d3 := *stats.Delta3h
		if d3 > DELTA_NOISE_THRESHOLD {
			stats.Trend = "rising"
		} else if d3 < -DELTA_NOISE_THRESHOLD {
			stats.Trend = "falling"
		} else {
			stats.Trend = "stable"
		}
	}

	return stats
}

func getUpdatedCacheDoc(cacheRef *firestore.DocumentRef, wp *WeatherPoint, tx *firestore.Transaction) (CacheDoc, error) {
	doc, err := tx.Get(cacheRef)
	var cache CacheDoc
	if err == nil {
		if err := doc.DataTo(&cache); err != nil {
			return cache, err
		}
	} else {
		cache = CacheDoc{History: []PressurePoint{}}
	}
	newPoint := PressurePoint{
		TimeStamp:       wp.Timestamp,
		TempC:           wp.TempC,
		TempF:           wp.TempF,
		HumidityPercent: wp.HumidityPercent,
		PressureMb:      wp.PressureMb,
		TempFeelC:       wp.TempFeelC,
		TempFeelF:       wp.TempFeelF,
		DewpointC:       wp.DewpointC,
		DewpointF:       wp.DewpointF,
	}
	cache.History = append(cache.History, newPoint)
	if len(cache.History) > MAX_HISTORY_POINTS {
		cache.History = cache.History[len(cache.History)-48:]
	}

	cache.LastUpdated = wp.Timestamp
	cache.CurrentValue = *wp
	cache.Analysis = calculatePressureStats(cache.History)

	return cache, nil
}

func saveRawWeatherData(ctx context.Context, client *firestore.Client, wp *WeatherPoint) error {
	_, _, err := client.Collection(WEATHER_RAW_COLLECTION).Add(ctx, wp)
	if err != nil {
		return err
	}
	return nil
}
