package flight

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"travel/pkg/logger"
)

func (s *Service) FilterFlights(ctx context.Context, req FilterRequest) (*FlightSearchResponse, error) {
	startTime := time.Now()
	cacheKey := s.generateCacheKey(req.SearchRequest)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var response FlightSearchResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			filteredFlights := response.Flights

			if req.Filters != nil {
				filteredFlights = s.applyFilters(response.Flights, *req.Filters)
			}

			if req.Sort != nil {
				filteredFlights = s.applySorting(filteredFlights, *req.Sort)
			}

			return &FlightSearchResponse{
				SearchCriteria: response.SearchCriteria,
				Metadata: Metadata{
					TotalResults:       uint32(len(filteredFlights)),
					ProvidersQueried:   response.Metadata.ProvidersQueried,
					ProvidersSucceeded: response.Metadata.ProvidersSucceeded,
					ProvidersFailed:    response.Metadata.ProvidersFailed,
					ProviderErrors:     response.Metadata.ProviderErrors,
					SearchTimeMs:       response.Metadata.SearchTimeMs,
					CacheKey:           cacheKey,
					CacheHit:           true,
				},
				Flights: filteredFlights,
			}, nil
		}
		s.logger.Error("Failed to unmarshal cached data", logger.Field{Key: "err", Value: err})
	}

	// Fetch fresh data from providers (Fallback if cache not found)
	response, err := s.flightClient.SearchFlights(ctx, req.SearchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh search results: %w", err)
	}

	searchTime := time.Since(startTime).Milliseconds()
	response.Metadata.SearchTimeMs = uint32(searchTime)
	response.Metadata.CacheHit = false
	response.Metadata.CacheKey = cacheKey

	// Cache in background without goroutine overhead
	responseBytes, err := json.Marshal(response)
	if err == nil {
		go func() {
			bgCtx := context.Background()
			if err := s.cache.Set(bgCtx, cacheKey, string(responseBytes), s.ttl); err != nil {
				s.logger.Error("Failed to cache refreshed results",
					logger.Field{Key: "err", Value: err},
					logger.Field{Key: "cache_key", Value: cacheKey},
				)
			}
		}()
	}

	filteredFlights := response.Flights
	if req.Filters != nil {
		filteredFlights = s.applyFilters(response.Flights, *req.Filters)
	}
	if req.Sort != nil {
		filteredFlights = s.applySorting(filteredFlights, *req.Sort)
	}

	return &FlightSearchResponse{
		SearchCriteria: SearchCriteria{
			Origin:        req.Origin,
			Destination:   req.Destination,
			DepartureDate: req.DepartureDate,
			Passengers:    req.Passengers,
			CabinClass:    req.CabinClass,
		},
		Metadata: Metadata{
			TotalResults:       uint32(len(filteredFlights)),
			ProvidersQueried:   response.Metadata.ProvidersQueried,
			ProvidersSucceeded: response.Metadata.ProvidersSucceeded,
			ProvidersFailed:    response.Metadata.ProvidersFailed,
			ProviderErrors:     response.Metadata.ProviderErrors,
			SearchTimeMs:       uint32(searchTime),
			CacheKey:           cacheKey,
			CacheHit:           false,
		},
		Flights: filteredFlights,
	}, nil
}

func (s *Service) applyFilters(flights []Flight, filters FilterOptions) []Flight {
	filtered := make([]Flight, 0, len(flights))

	// Pre-parse time filters to timestamps (parse once, not per flight)
	var depFromSec, depToSec, arrFromSec, arrToSec int64
	hasDepartureFilter := filters.DepartureTime != nil
	hasArrivalFilter := filters.ArrivalTime != nil

	if hasDepartureFilter {
		depFromSec = parseTimeToSeconds(filters.DepartureTime.From)
		depToSec = parseTimeToSeconds(filters.DepartureTime.To)
	}
	if hasArrivalFilter {
		arrFromSec = parseTimeToSeconds(filters.ArrivalTime.From)
		arrToSec = parseTimeToSeconds(filters.ArrivalTime.To)
	}

	for _, f := range flights {
		if filters.PriceRange != nil {
			if f.Price.Amount < filters.PriceRange.Low || f.Price.Amount > filters.PriceRange.High {
				continue
			}
		}

		if filters.MaxStops != nil && f.Stops > *filters.MaxStops {
			continue
		}

		if hasDepartureFilter {
			depSec := getSecondsFromMidnight(f.Departure.Datetime)
			if depSec < depFromSec || depSec > depToSec {
				continue
			}
		}

		if hasArrivalFilter {
			arrSec := getSecondsFromMidnight(f.Arrival.Datetime)
			if arrSec < arrFromSec || arrSec > arrToSec {
				continue
			}
		}

		if len(filters.Airlines) > 0 {
			matched := false
			for _, airline := range filters.Airlines {
				if strings.EqualFold(f.Airline.Code, airline) || strings.EqualFold(f.Airline.Name, airline) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if filters.MaxDuration != nil && f.Duration.TotalMinutes > *filters.MaxDuration {
			continue
		}

		filtered = append(filtered, f)
	}

	return filtered
}

func parseTimeToSeconds(timeStr string) int64 {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0
	}
	return int64(t.Hour()*3600 + t.Minute()*60)
}

func getSecondsFromMidnight(dt time.Time) int64 {
	return int64(dt.Hour()*3600 + dt.Minute()*60 + dt.Second())
}

func (s *Service) applySorting(flights []Flight, sort SortOptions) []Flight {
	if len(flights) == 0 {
		return flights
	}

	sorted := make([]Flight, len(flights))
	copy(sorted, flights)

	if sort.By == "best_value" {
		sorted = s.calculateBestValueScores(sorted)
	}

	switch sort.By {
	case "price":
		s.sortByPrice(sorted, sort.Order)
	case "duration":
		s.sortByDuration(sorted, sort.Order)
	case "departure_time":
		s.sortByDepartureTime(sorted, sort.Order)
	case "arrival_time":
		s.sortByArrivalTime(sorted, sort.Order)
	case "best_value":
		s.sortByBestValue(sorted, sort.Order)
	default:
		s.logger.Warn("Invalid sort criteria", logger.Field{Key: "sort_by", Value: sort.By})
	}

	return sorted
}

// TODO: scoring
func (s *Service) calculateBestValueScores(flights []Flight) []Flight {
	if len(flights) == 0 {
		return flights
	}

	return flights
}

func (s *Service) sortByPrice(flights []Flight, order string) {
	sort.Slice(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Price.Amount > flights[j].Price.Amount
		}
		return flights[i].Price.Amount < flights[j].Price.Amount
	})
}

func (s *Service) sortByDuration(flights []Flight, order string) {
	sort.Slice(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Duration.TotalMinutes > flights[j].Duration.TotalMinutes
		}
		return flights[i].Duration.TotalMinutes < flights[j].Duration.TotalMinutes
	})
}

func (s *Service) sortByDepartureTime(flights []Flight, order string) {
	sort.Slice(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Departure.Timestamp > flights[j].Departure.Timestamp
		}
		return flights[i].Departure.Timestamp < flights[j].Departure.Timestamp
	})
}

func (s *Service) sortByArrivalTime(flights []Flight, order string) {
	sort.Slice(flights, func(i, j int) bool {
		if order == "desc" {
			return flights[i].Arrival.Timestamp > flights[j].Arrival.Timestamp
		}
		return flights[i].Arrival.Timestamp < flights[j].Arrival.Timestamp
	})
}

func (s *Service) sortByBestValue(flights []Flight, order string) {
	sort.Slice(flights, func(i, j int) bool {
		// Scores should already be calculated
		if flights[i].BestValueScore == nil || flights[j].BestValueScore == nil {
			return false
		}

		if order == "desc" {
			return *flights[i].BestValueScore > *flights[j].BestValueScore
		}
		return *flights[i].BestValueScore < *flights[j].BestValueScore
	})
}

// InvalidateCache manually invalidates cache for a specific route
func (s *Service) InvalidateCache(ctx context.Context, req SearchRequest) error {
	cacheKey := s.generateCacheKey(req)
	s.logger.Info("Invalidating cache", logger.Field{Key: "cache_key", Value: cacheKey})
	return s.cache.Del(ctx, cacheKey)
}
