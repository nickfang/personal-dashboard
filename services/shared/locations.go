package shared

// Location represents a monitored geographic point.
type Location struct {
	ID   string
	Lat  float64
	Long float64
}

// Locations is the canonical list used by all collector services.
var Locations = []Location{
	{ID: "house-nick", Lat: 30.260543381977474, Long: -97.66768538740229},
	{ID: "house-nita", Lat: 30.29420179895202, Long: -97.6958691874014},
	{ID: "distribution-hall", Lat: 30.261932944618565, Long: -97.72816923158192},
}
