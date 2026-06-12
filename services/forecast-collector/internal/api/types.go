package api

import "time"

// ForecastHour matches one entry of forecastHours in the Google Weather API
// forecast/hours:lookup JSON response (verified against a live sample).
type ForecastHour struct {
	Interval struct {
		StartTime time.Time `json:"startTime"`
		EndTime   time.Time `json:"endTime"`
	} `json:"interval"`
	Temperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"temperature"`
	FeelsLikeTemperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"feelsLikeTemperature"`
	DewPoint struct {
		Degrees float64 `json:"degrees"`
	} `json:"dewPoint"`
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
	// NOTE: hourly forecasts use probability.percent, unlike
	// currentConditions:lookup which uses probability.probability.
	Precipitation struct {
		Probability struct {
			Percent int    `json:"percent"`
			Type    string `json:"type"`
		} `json:"probability"`
	} `json:"precipitation"`
}

// forecastHoursResponse is one page of the paginated forecast response.
type forecastHoursResponse struct {
	ForecastHours []ForecastHour `json:"forecastHours"`
	TimeZone      struct {
		ID string `json:"id"`
	} `json:"timeZone"`
	NextPageToken string `json:"nextPageToken"`
}
