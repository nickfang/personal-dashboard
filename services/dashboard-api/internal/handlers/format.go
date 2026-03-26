package handlers

import (
	"fmt"
	"slices"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	pressurePb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
)

func formatPressureText(pressureStats []*pressurePb.PressureStat) map[string]string {
	pressureByLocation := make(map[string]string)
	for _, pressureStat := range pressureStats {
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
	for _, pollenReport := range pollenReports {
		if len(pollenReport.Plants) == 0 {
			pollenByLocation[pollenReport.LocationId] = "No pollen data available"
			continue
		}
		location := pollenReport.LocationId
		pollenByLocation[location] = ""
		pollenByLocation[location] += fmt.Sprintf("Pollen: %s\n", pollenReport.CollectedAt.AsTime().Local().Format("2006.01.02 15:04:05"))
		slices.SortFunc(pollenReport.Plants, func(a, b *pollenPb.PollenPlant) int {
			// uncomment to sort by In Season first
			// if a.InSeason != b.InSeason {
			// 	if a.InSeason {
			// 		return -1
			// 	}
			// 	return 1
			// }
			return int(b.Index) - int(a.Index)
		})
		first := true
		currentIndex := pollenReport.Plants[0].Index + 1
		for _, plant := range pollenReport.Plants {
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
		// Type doesn't seem to be useful.  uncomment if someone asks for it.
		// pollenByLocation[location] += fmt.Sprintln("  Type:")
		// for _, pollenType := range pollenReport.Types {
		// 	if pollenType.Index > 0 {
		// 		inSeason := "In Season"
		// 		if pollenType.InSeason {
		// 			inSeason = "Out of Season"
		// 		}
		// 		pollenByLocation[location] += fmt.Sprintf("  %s: %s - %s\n", pollenType.Code, pollenType.Category, inSeason)
		// 	}
		// }
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
	for location, info := range byLocation {
		data += fmt.Sprintf("---------------- %s ----------------\n", location)
		data += info
	}
	return data, nil
}
