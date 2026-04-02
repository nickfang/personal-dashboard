package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	// "github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
)

type WeatherPoint struct {
	Location             string    `firestore:"location"`
	Timestamp            time.Time `firestore:"timestamp"`
	HumidityPercent      int       `firestore:"humidity_pct"`
	PrecipitationPercent int       `firestore:"precipitation_pct"`
	UVIndex              int       `firestore:"uv_index"`
	PressureMb           float64   `firestore:"pressure_mb"`
	WindDirDeg           int       `firestore:"wind_dir_deg"`
	TempC                float64   `firestore:"temp_c"`
	TempFeelC            float64   `firestore:"temp_feel_c"`
	DewpointC            float64   `firestore:"dewpoint_c"`
	WindSpeedKph         float64   `firestore:"wind_speed_kph"`
	WindGustKph          float64   `firestore:"wind_gust_kph"`
	VisibilityKm         float64   `firestore:"visibility_km"`
	TempF                float64   `firestore:"temp_f"`
	TempFeelF            float64   `firestore:"temp_feel_f"`
	WindSpeedMph         float64   `firestore:"wind_speed_mph"`
	WindGustMph          float64   `firestore:"wind_gust_mph"`
	VisibilityM          float64   `firestore:"visibility_miles"`
	DewpointF            float64   `firestore:"dewpoint_f"`
}

type WeatherPoints []WeatherPoint

func main() {
	// get data
	projectID := "fang-gcp"
	ctx := context.Background()

	repo, err := repository.NewFirestoreRepository(ctx, projectID)
	if err != nil {
		os.Exit(1)
	}
	defer repo.Close()

	file, err := os.Create("weather_raw.csv")
	if err != nil {
		log.Fatalf("failed to create file: %s", err)
	}
	defer file.Close()

	svc := service.NewWeatherService(repo)
	data, err := svc.GetAllRaw(ctx)
	if err != nil {
		os.Exit(1)
	}

	writer := csv.NewWriter(file)

	headers := []string{
		"Id",
		"Location",
		"Timestamp",
		"HumidityPercent",
		"PrecipitationPercent",
		"UVIndex",
		"PressureMb",
		"WindDirDeg",
		"TempC",
		"TempFeelC",
		"DewpointC",
		"WindSpeedKph",
		"WindGustKph",
		"VisibilityKm",
		"TempF",
		"TempFeelF",
		"WindSpeedMph",
		"WindGustMph",
		"VisibilityM",
		"DewpointF",
	}
	writer.Write(headers)
	for i, d := range data {
		record := []string{
			strconv.Itoa(i),
			d.LocationID,
			d.Timestamp.Format(time.RFC3339),
			strconv.Itoa(d.HumidityPercent),
			strconv.Itoa(d.PrecipitationPercent),
			strconv.Itoa(d.UVIndex),
			strconv.FormatFloat(d.PressureMb, 'f', 2, 64),
			strconv.Itoa(d.WindDirDeg),
			strconv.FormatFloat(d.TempC, 'f', 2, 64),
			strconv.FormatFloat(d.TempFeelC, 'f', 2, 64),
			strconv.FormatFloat(d.DewpointC, 'f', 2, 64),
			strconv.FormatFloat(d.WindSpeedKph, 'f', 2, 64),
			strconv.FormatFloat(d.WindGustKph, 'f', 2, 64),
			strconv.FormatFloat(d.VisibilityKm, 'f', 2, 64),
			strconv.FormatFloat(d.TempF, 'f', 2, 64),
			strconv.FormatFloat(d.TempFeelF, 'f', 2, 64),
			strconv.FormatFloat(d.WindSpeedMph, 'f', 2, 64),
			strconv.FormatFloat(d.WindGustMph, 'f', 2, 64),
			strconv.FormatFloat(d.VisibilityM, 'f', 2, 64),
			strconv.FormatFloat(d.DewpointF, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			log.Fatalf("error writing data to csv: %s", err)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		log.Fatalf("error flushing writer: %s", err)
	}

	fmt.Println("Success!")
}
