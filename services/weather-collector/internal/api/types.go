package api

// WeatherAPIResponse matches the Google Weather API JSON structure.
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
