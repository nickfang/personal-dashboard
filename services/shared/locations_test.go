package shared

import "testing"

func TestLocationsExist(t *testing.T) {
	if len(Locations) != 3 {
		t.Fatalf("expected 3 locations, got %d", len(Locations))
	}
}

func TestLocationIDs(t *testing.T) {
	expectedIDs := []string{"house-nick", "house-nita", "distribution-hall"}

	for i, expected := range expectedIDs {
		if Locations[i].ID != expected {
			t.Errorf("Locations[%d].ID = %q, want %q", i, Locations[i].ID, expected)
		}
	}
}

func TestLocationCoordinatesAreValid(t *testing.T) {
	for _, loc := range Locations {
		if loc.Lat < -90 || loc.Lat > 90 {
			t.Errorf("location %q has invalid latitude: %f", loc.ID, loc.Lat)
		}
		if loc.Long < -180 || loc.Long > 180 {
			t.Errorf("location %q has invalid longitude: %f", loc.ID, loc.Long)
		}
	}
}

func TestLocationFieldsExported(t *testing.T) {
	// Verify all fields are accessible (exported) from the shared package.
	loc := Locations[0]
	_ = loc.ID
	_ = loc.Lat
	_ = loc.Long
}
