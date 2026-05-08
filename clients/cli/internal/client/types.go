package client

// Response is the top-level JSON shape returned by GET /v1/dashboard.
// Each inner map is keyed by location ID. A location may be absent from
// any of the three maps if that data type has no reading for it.
type Response struct {
	Weather  map[string]Weather  `json:"weather"`
	Pressure map[string]Pressure `json:"pressure"`
	Pollen   map[string]Pollen   `json:"pollen"`
}

// Merge folds the entries from src into r using last-write-wins semantics:
// for each key present in src, the entry is written into r only if it is
// strictly newer than the entry already there (compared by per-payload
// timestamp), or if r had no entry for that key. Keys absent from src are
// left untouched.
//
// The timestamps come straight from the upstream payload (LastUpdated for
// Weather/Pressure, CollectedAt for Pollen) and arrive as canonical RFC 3339
// strings via protojson, so lexicographic > is equivalent to time.After.
//
// Why LWW: the TUI may have an in-flight full Fetch and an in-flight
// FetchLocation arrive out of order — a stale full response that returns
// after a fresh single-location response should not clobber the focused
// location's newer data. Equal timestamps (the steady state today, since
// upstream readings update hourly) skip the write entirely.
func (r *Response) Merge(src *Response) {
	if r == nil || src == nil {
		return
	}
	if r.Weather == nil {
		r.Weather = make(map[string]Weather, len(src.Weather))
	}
	for k, v := range src.Weather {
		if existing, ok := r.Weather[k]; !ok || v.LastUpdated > existing.LastUpdated {
			r.Weather[k] = v
		}
	}
	if r.Pressure == nil {
		r.Pressure = make(map[string]Pressure, len(src.Pressure))
	}
	for k, v := range src.Pressure {
		if existing, ok := r.Pressure[k]; !ok || v.LastUpdated > existing.LastUpdated {
			r.Pressure[k] = v
		}
	}
	if r.Pollen == nil {
		r.Pollen = make(map[string]Pollen, len(src.Pollen))
	}
	for k, v := range src.Pollen {
		if existing, ok := r.Pollen[k]; !ok || v.CollectedAt > existing.CollectedAt {
			r.Pollen[k] = v
		}
	}
}

// Weather matches the protojson output of the dashboard-api weather payload.
type Weather struct {
	LocationID           string  `json:"locationId"`
	LastUpdated          string  `json:"lastUpdated"`
	TempC                float64 `json:"tempC"`
	TempF                float64 `json:"tempF"`
	TempFeelC            float64 `json:"tempFeelC"`
	TempFeelF            float64 `json:"tempFeelF"`
	HumidityPercent      int     `json:"humidityPercent"`
	PrecipitationPercent int     `json:"precipitationPercent"`
	PressureMb           float64 `json:"pressureMb"`
}

// Pressure matches the protojson output of the dashboard-api pressure payload.
type Pressure struct {
	LocationID  string  `json:"locationId"`
	LastUpdated string  `json:"lastUpdated"`
	Delta1h     float64 `json:"delta1h"`
	Delta3h     float64 `json:"delta3h"`
	Delta6h     float64 `json:"delta6h"`
	Delta12h    float64 `json:"delta12h"`
	Delta24h    float64 `json:"delta24h"`
	Trend       string  `json:"trend"`
}

// PollenType is one of the high-level pollen categories (TREE, GRASS, WEED).
type PollenType struct {
	Code     string `json:"code"`
	Index    int    `json:"index"`
	Category string `json:"category"`
	InSeason bool   `json:"inSeason"`
}

// PollenPlant is a specific plant reading within a pollen payload.
type PollenPlant struct {
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
	Index       int    `json:"index"`
	Category    string `json:"category"`
	InSeason    bool   `json:"inSeason"`
}

// Pollen matches the protojson output of the dashboard-api pollen payload.
type Pollen struct {
	LocationID      string        `json:"locationId"`
	CollectedAt     string        `json:"collectedAt"`
	OverallIndex    int           `json:"overallIndex"`
	OverallCategory string        `json:"overallCategory"`
	DominantType    string        `json:"dominantType"`
	Types           []PollenType  `json:"types"`
	Plants          []PollenPlant `json:"plants"`
}
