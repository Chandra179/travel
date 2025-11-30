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

	// 1. Determine Ranges
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

	// 2. Normalize and Score
	for i := range flights {
		normPrice := normalize(float64(flights[i].Price.Amount), float64(minPrice), float64(maxPrice))
		normDuration := normalize(float64(flights[i].Duration.TotalMinutes), float64(minDuration), float64(maxDuration))
		normStops := normalize(float64(flights[i].Stops), float64(minStops), float64(maxStops))

		score := (priceWeight * normPrice) + (durationWeight * normDuration) + (stopsWeight * normStops)
		flights[i].BestValueScore = &score
	}
}

func normalize(val, min, max float64) float64 {
	if max > min {
		// Invert so that Lower (Price/Duration) = Higher Score (1.0)
		return 1.0 - (val-min)/(max-min)
	}
	return 1.0
}
