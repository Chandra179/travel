package flight

import (
	"math"
	"sort"
	"travel/pkg/logger"
)

const (
	priceWeight    = 0.45
	durationWeight = 0.35
	stopsWeight    = 0.20
)

func (s *Service) applySorting(flights []Flight, sortOpt SortOptions) []Flight {
	if len(flights) <= 1 {
		return flights
	}

	// Work on a copy if you want to be safe, though sorting in place is often acceptable in Go services
	// returning a new slice is safer for concurrency.
	sorted := make([]Flight, len(flights))
	copy(sorted, flights)

	switch sortOpt.By {
	case "price":
		s.sortByPrice(sorted, sortOpt.Order)
	case "duration":
		s.sortByDuration(sorted, sortOpt.Order)
	case "departure_time":
		s.sortByDepartureTime(sorted, sortOpt.Order)
	case "arrival_time":
		s.sortByArrivalTime(sorted, sortOpt.Order)
	case "best_value":
		s.sortByBestValue(sorted, sortOpt.Order)
	default:
		s.logger.Warn("invalid_sort_criteria", logger.Field{Key: "sort_by", Value: sortOpt.By})
	}

	return sorted
}

// Using Sort Stable to prevent UI jumping when values are equal
func (s *Service) sortByPrice(flights []Flight, order string) {
	sort.SliceStable(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Price.Amount > flights[j].Price.Amount
		}
		return flights[i].Price.Amount < flights[j].Price.Amount
	})
}

func (s *Service) sortByDuration(flights []Flight, order string) {
	sort.SliceStable(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Duration.TotalMinutes > flights[j].Duration.TotalMinutes
		}
		return flights[i].Duration.TotalMinutes < flights[j].Duration.TotalMinutes
	})
}

func (s *Service) sortByDepartureTime(flights []Flight, order string) {
	sort.SliceStable(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Departure.Timestamp > flights[j].Departure.Timestamp
		}
		return flights[i].Departure.Timestamp < flights[j].Departure.Timestamp
	})
}

func (s *Service) sortByArrivalTime(flights []Flight, order string) {
	sort.SliceStable(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Arrival.Timestamp > flights[j].Arrival.Timestamp
		}
		return flights[i].Arrival.Timestamp < flights[j].Arrival.Timestamp
	})
}

func (s *Service) sortByBestValue(flights []Flight, order string) {
	if len(flights) <= 1 {
		return
	}

	// This mutates the flights by adding scores.
	// Since 'sorted' is a deep copy of the slice structure (but shallow copy of elements),
	// modifying *Flight fields affects the original if pointers are shared, but here Flight is a struct value in slice.
	s.calculateBestValueScores(flights)

	sort.SliceStable(flights, func(i, j int) bool {
		scoreI, scoreJ := 0.0, 0.0
		if flights[i].BestValueScore != nil {
			scoreI = *flights[i].BestValueScore
		}
		if flights[j].BestValueScore != nil {
			scoreJ = *flights[j].BestValueScore
		}

		if order == "desc" {
			return scoreI > scoreJ
		}
		return scoreI < scoreJ
	})
}

func (s *Service) calculateBestValueScores(flights []Flight) {
	var minPrice, maxPrice uint64 = math.MaxUint64, 0
	var minDuration, maxDuration uint32 = math.MaxUint32, 0
	var minStops, maxStops uint32 = math.MaxUint32, 0

	// STEP 1: SCAN
	// Find the boundaries (Min/Max) of the current dataset.
	// ---------------------------------------------------------
	// SCENARIO:
	// Flight A: Price 100, Duration 300
	// Flight B: Price 150, Duration 180
	// Flight C: Price 200, Duration 600
	//
	// RESULT AFTER LOOP:
	// minPrice = 100, maxPrice = 200
	// minDuration = 180, maxDuration = 600
	// ---------------------------------------------------------
	for _, f := range flights {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minDuration {
			minDuration = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxDuration {
			maxDuration = f.Duration.TotalMinutes
		}
		if f.Stops < minStops {
			minStops = f.Stops
		}
		if f.Stops > maxStops {
			maxStops = f.Stops
		}
	}

	for i := range flights {
		// STEP 2: SCORE
		// ---------------------------------------------------------
		// SIMULATION: Calculating Score for "Flight B" (Price 150, Dur 180)
		//
		// 1. Normalize Price (150)
		//    Range: 100 to 200. 150 is exactly in the middle.
		//    Logic: normalize(150, 100, 200) -> returns 0.5
		//
		// 2. Normalize Duration (180)
		//    Range: 180 to 600. 180 is the Minimum (Best).
		//    Logic: normalize(180, 180, 600) -> returns 1.0 (Perfect score)
		//
		// 3. Apply Weights
		//    Price Score:    0.5 * 0.45 (Weight) = 0.225
		//    Duration Score: 1.0 * 0.35 (Weight) = 0.350
		//    Stops Score:    (Assume 0 stops = 1.0) * 0.20 = 0.200
		//
		// 4. Final Score
		//    0.225 + 0.350 + 0.200 = 0.775
		// ---------------------------------------------------------
		normPrice := normalize(float64(flights[i].Price.Amount), float64(minPrice), float64(maxPrice))
		normDuration := normalize(float64(flights[i].Duration.TotalMinutes), float64(minDuration), float64(maxDuration))
		normStops := normalize(float64(flights[i].Stops), float64(minStops), float64(maxStops))

		score := (priceWeight * normPrice) + (durationWeight * normDuration) + (stopsWeight * normStops)
		flights[i].BestValueScore = &score
	}
}

// normalize converts a value into a 0.0 to 1.0 scale relative to the range.
// Lower values (cheaper price, shorter duration) get HIGHER scores.
func normalize(val, min, max float64) float64 {
	// Guard against division by zero if all flights have the exact same price
	if max > min {
		// FORMULA BREAKDOWN:
		// (val - min)       -> How much more expensive is this than the cheapest option?
		// (max - min)       -> What is the total price gap in the market?
		// Division          -> Percentage of the gap. 0% = Cheapest, 100% = Most Expensive.
		// 1.0 - Result      -> Flip it. We want Cheapest to be 1.0 (100%).

		// Ex: Price 150, Min 100, Max 200
		// (150 - 100) = 50
		// (200 - 100) = 100
		// 50 / 100 = 0.5 (It is 50% towards the most expensive)
		// 1.0 - 0.5 = 0.5 Score.
		return 1.0 - (val-min)/(max-min)
	}

	// If max == min, all flights are equal in this metric. Give them all perfect scores.
	return 1.0
}
