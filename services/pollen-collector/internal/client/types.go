package client

// PollenAPIResponse represents the Google Pollen API response.
type PollenAPIResponse struct {
	DailyInfo []DailyInfo `json:"dailyInfo"`
}

type DailyInfo struct {
	Date           APIDate          `json:"date"`
	PollenTypeInfo []PollenTypeInfo `json:"pollenTypeInfo"`
	PlantInfo      []PlantInfo      `json:"plantInfo"`
}

type APIDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type PollenTypeInfo struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	InSeason    bool      `json:"inSeason"`
	IndexInfo   IndexInfo `json:"indexInfo"`
}

type PlantInfo struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	InSeason    bool      `json:"inSeason"`
	IndexInfo   IndexInfo `json:"indexInfo"`
}

type IndexInfo struct {
	Value    int    `json:"value"`
	Category string `json:"category"`
}
