package handlers

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	pressurePb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
)

func formatPressureText(pressureStats []*pressurePb.PressureStat) map[string]string {
	pressureByLocation := make(map[string]string)
	sortedPressureStats := slices.Clone(pressureStats)
	slices.SortFunc(sortedPressureStats, func(a, b *pressurePb.PressureStat) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, pressureStat := range sortedPressureStats {
		location := pressureStat.LocationId
		pressureByLocation[location] = ""
		pressureByLocation[location] += fmt.Sprintf("Pressure: %s\n", pressureStat.LastUpdated.AsTime().Local().Format("2006.01.02 15:04:05"))
		pressureByLocation[location] += fmt.Sprintf("  %s\n", pressureStat.Trend)
		pressureByLocation[location] += fmt.Sprintf("  Deltas: %.2f(1h), %.2f(3h), %.2f(6h) %.2f(12h) %.2f(24h)\n", pressureStat.Delta_1H, pressureStat.Delta_3H, pressureStat.Delta_6H, pressureStat.Delta_12H, pressureStat.Delta_24H)
	}
	return pressureByLocation
}

func formatPollenText(pollenReports []*pollenPb.PollenReport) map[string]string {
	pollenByLocation := make(map[string]string)
	sortedPollenReports := slices.Clone(pollenReports)
	slices.SortFunc(sortedPollenReports, func(a, b *pollenPb.PollenReport) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, pollenReport := range sortedPollenReports {
		if len(pollenReport.Plants) == 0 {
			pollenByLocation[pollenReport.LocationId] = "No pollen data available"
			continue
		}
		location := pollenReport.LocationId
		pollenByLocation[location] = ""
		pollenByLocation[location] += fmt.Sprintf("Pollen: %s\n", pollenReport.CollectedAt.AsTime().Local().Format("2006.01.02 15:04:05"))
		sortedPlants := slices.Clone(pollenReport.Plants)
		slices.SortFunc(sortedPlants, func(a, b *pollenPb.PollenPlant) int {
			return int(b.Index) - int(a.Index)
		})
		first := true
		currentIndex := sortedPlants[0].Index + 1
		for _, plant := range sortedPlants {
			if plant.Index < 1 {
				break
			}
			if currentIndex != plant.Index {
				if first {
					first = false
				} else {
					pollenByLocation[location] += fmt.Sprintln()
				}
				pollenByLocation[location] += fmt.Sprintf("  %-10s", plant.Category)
				currentIndex = plant.Index
			}
			inSeason := "Out of Season"
			if plant.InSeason {
				inSeason = "In Season"
			}
			pollenByLocation[location] += fmt.Sprintf(" %s (%s)", plant.DisplayName, inSeason)
		}
		pollenByLocation[location] += fmt.Sprintln()
		// Don't show types.  It's not useful.
	}
	return pollenByLocation
}

func formatDashboardText(pressureStats []*pressurePb.PressureStat, pollenReports []*pollenPb.PollenReport) (string, error) {
	byLocation := make(map[string]string)
	pressureByLocation := formatPressureText(pressureStats)

	for _, pressureStat := range pressureStats {
		byLocation[pressureStat.LocationId] = pressureByLocation[pressureStat.LocationId]
	}
	pollenByLocation := formatPollenText(pollenReports)
	for _, pollenReport := range pollenReports {
		byLocation[pollenReport.LocationId] += pollenByLocation[pollenReport.LocationId]
	}

	data := ""
	for _, key := range slices.Sorted(maps.Keys(byLocation)) {
		info := byLocation[key]
		data += fmt.Sprintf("---------------- %s ----------------\n", key)
		data += info
	}
	return data, nil
}
