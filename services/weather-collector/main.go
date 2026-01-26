package main

import (
	// "context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	// "cloud.google.com/go/firestore"
)

type WeatherLocation struct {
	ID   string
	lat  float64
	long float64
}

type WeatherPoint struct {
	Location     string    `firestore:"location"`
	Timestamp    time.Time `firestore:"timestamp"`
	TempC        float64   `firestore:"temp_c"`
	TempFeelC    float64   `firestore:"temp_feel_c"`
	Humidity     int       `firestore:"humidity"`
	UVIndex      int       `firestore:"uv_index"`
	PressureMb   float64   `firestore:"pressure_mb"`
	WindDirDeg   int       `firestore:"wind_dir_deg"`
	WindSpeedKph float64   `firestore:"wind_speed_kph"`
	WindGustKph  float64   `firestore:"wind_gust_kph"`
	VisibilityKm float64   `firestore:"visibility_km"`
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
	RelativeHumidity int `json:"relativeHumidity"`
	UVIndex          int `json:"uvIndex"`
	AirPressure      struct {
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
}

func main() {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		log.Fatal("Missing required env var: GOOGLE_MAPS_API_KEY")
	}

	for _, loc := range locations {
		_, err := fetchWeather(apiKey, loc)
		if err != nil {
			log.Printf("Error fetching weather for %s: %v", loc.ID, err)
			continue
		}

		// For now, just print the data
		// fmt.Printf("Fetched weather for %s: %+v\n", loc.ID, wp)
	}
}

func fetchWeather(apiKey string, loc WeatherLocation) (*WeatherPoint, error) {
	url := fmt.Sprintf("https://weather.googleapis.com/v1/currentConditions:lookup?key=%s&location.latitude=%f&location.longitude=%f", apiKey, loc.lat, loc.long)

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

func mapToWeatherPoint(locationID string, data WeatherAPIResponse) *WeatherPoint {
	wp := &WeatherPoint{
		Location:     locationID,
		Timestamp:    time.Now(),
		TempC:        data.Temperature.Degrees,
		TempFeelC:    data.FeelsLikeTemperature.Degrees,
		Humidity:     data.RelativeHumidity,
		UVIndex:      data.UVIndex,
		PressureMb:   data.AirPressure.MeanSeaLevelMillibars,
		WindDirDeg:   data.Wind.Direction.Degrees,
		WindSpeedKph: data.Wind.Speed.Value,
		WindGustKph:  data.Wind.Gust.Value,
		VisibilityKm: data.Visibility.Distance,
	}

	log.Printf("Mapped Data [DB Format] for %s:\n"+
		"  Timestamp:    %v\n"+
		"  Temp:         %.1f°C\n"+
		"  Feels Like:   %.1f°C\n"+
		"  Humidity:     %d%%\n"+
		"  UV Index:     %d\n"+
		"  Pressure:     %.1f mb\n"+
		"  Wind:         %d° @ %.1f kph (gust %.1f kph)\n"+
		"  Visibility:   %.1f km",
		locationID, wp.Timestamp.Format(time.RFC3339), wp.TempC, wp.TempFeelC, wp.Humidity, wp.UVIndex, wp.PressureMb, wp.WindDirDeg, wp.WindSpeedKph, wp.WindGustKph, wp.VisibilityKm)

	return wp
}
