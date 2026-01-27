package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/firestore"
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
	TimeStamp       time.Time `firestore:"ts"`
	HumidityPercent int       `firestore:"h"`
	PressureMb      float64   `firestore:"p"`

	TempC     float64 `firestore:"t"`
	TempFeelC float64 `firestore:"temp_feel_c"`
	DewpointC float64 `firestore:"dewpoint_c"`

	TempF     float64 `firestore:"temp_f"`
	TempFeelF float64 `firestore:"temp_feel_f"`
	DewpointF float64 `firestore:"dewpoint_f"`
}

type PressureStats struct {
	Delta1h  float64 `firestore:"delta_01h"`
	Delta3h  float64 `firestore:"delta_03h"`
	Delta6h  float64 `firestore:"delta_06h"`
	Delta12h float64 `firestore:"delta_12h"`
	Delta24h float64 `firestore:"delta_24h"`
	Trend    string  `firestore:"trend"`
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
	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")

	if apiKey == "" || projectID == "" {
		log.Fatal("Missing required env vars: GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
	}

	client, err := firestore.NewClientWithDatabase(ctx, projectID, "weather-log")
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	defer client.Close()

	for _, loc := range locations {
		wp, err := fetchWeather(apiKey, loc)
		if err != nil {
			log.Printf("Error fetching weather for %s: %v", loc.ID, err)
			continue
		}

		// 1. Save to Raw Archive
		err = saveRawWeatherData(ctx, client, wp)
		if err != nil {
			log.Printf("Error saving raw weather data for %s: %v", loc.ID, err)
			continue
		}

		// 2. Update Hot Cache (Transaction)
		cacheRef := client.Collection("weather_cache").Doc(loc.ID)
		err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			cache, err := getUpdatedCacheDoc(cacheRef, wp, tx)
			if err != nil {
				log.Printf("Error getting updated cache doc for %s: %v", loc.ID, err)
				return err
			}

			return tx.Set(cacheRef, cache)
		})

		if err != nil {
			log.Printf("Error updating cache for %s: %v", loc.ID, err)
		}

		fmt.Printf("Processed weather for %s\n", loc.ID)
	}
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

	return mapToWeatherPoint(loc.ID, data), nil
}

func CtoF(c float64) float64 {
	return (c * 1.8) + 32
}

func KtoM(k float64) float64 {
	return k * 0.621371
}

func mapToWeatherPoint(locationID string, data WeatherAPIResponse) *WeatherPoint {

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

	log.Printf("Mapped Data [DB Format] for %s:\n"+
		"  Timestamp:    %v\n"+
		"  Temp:         %.1f°C\n"+
		"  Feels Like:   %.1f°C\n"+
		"  Humidity:     %d%%\n"+
		"  UV Index:     %d\n"+
		"  Pressure:     %.1f mb\n"+
		"  Wind:         %d° @ %.1f kph (gust %.1f kph)\n"+
		"  Visibility:   %.1f km\n"+
		"  DewPoint:     %.1f°C\n"+
		"  Precipitation:     %d%%\n",
		locationID, wp.Timestamp.Format(time.RFC3339), wp.TempC, wp.TempFeelC, wp.HumidityPercent, wp.UVIndex, wp.PressureMb, wp.WindDirDeg, wp.WindSpeedKph, wp.WindGustKph, wp.VisibilityKm, wp.DewpointC, wp.PrecipitationPercent)

	return wp
}

func calculatePressureStats(history []PressurePoint) PressureStats {
	stats := PressureStats{Trend: "stable"}

	// Need at least 2 points to calculate any delta
	if len(history) < 2 {
		return stats
	}

	// Helper to find pressure X hours ago (assuming hourly points)
	// Returns 0.0 if not enough history
	getDelta := func(hoursAgo int) float64 {
		// We need 'hoursAgo + 1' items to look back that far
		// e.g. for 1h ago, we need index (len-1) and (len-2), so len >= 2
		if len(history) > hoursAgo {
			current := history[len(history)-1].PressureMb
			past := history[len(history)-1-hoursAgo].PressureMb
			return current - past
		}
		return 0.0
	}

	stats.Delta1h = getDelta(1)
	stats.Delta3h = getDelta(3)
	stats.Delta6h = getDelta(6)
	stats.Delta12h = getDelta(12)
	stats.Delta24h = getDelta(24)

	// Simple trend logic with noise threshold
	if stats.Delta3h > 0.5 {
		stats.Trend = "rising"
	} else if stats.Delta3h < -0.5 {
		stats.Trend = "falling"
	} else {
		stats.Trend = "stable"
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
	if len(cache.History) > 48 {
		cache.History = cache.History[len(cache.History)-24:]
	}

	cache.LastUpdated = wp.Timestamp
	cache.CurrentValue = *wp
	cache.Analysis = calculatePressureStats(cache.History)

	return cache, nil
}

func saveRawWeatherData(ctx context.Context, client *firestore.Client, wp *WeatherPoint) error {
	_, _, err := client.Collection("weather_raw").Add(ctx, wp)
	if err != nil {
		return err
	}
	return nil
}
