package flight

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"travel/pkg/cache"
	"travel/pkg/logger"
)

type FlightClient interface {
	SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error)
}

type Service struct {
	flightClient FlightClient
	cache        cache.Cache
	ttl          time.Duration
	logger       logger.Client
}

func NewService(flightClient FlightClient, cache cache.Cache, ttlSeconds int, logger logger.Client) *Service {
	return &Service{
		flightClient: flightClient,
		cache:        cache,
		ttl:          time.Duration(ttlSeconds) * time.Second,
		logger:       logger,
	}
}

// getOrFetchFlights is the Centralized Data Access Layer.
// It handles Cache checking, API fetching, and background Cache setting.
func (s *Service) getOrFetchFlights(ctx context.Context, req SearchRequest) ([]Flight, Metadata, error) {
	cacheKey := s.generateCacheKey(req)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var response FlightSearchResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			response.Metadata.CacheHit = true
			response.Metadata.CacheKey = cacheKey
			return response.Flights, response.Metadata, nil
		}
		s.logger.Error("cache_unmarshal_err", logger.Field{Key: "err", Value: err})
	}

	//  Fallback: Fetch from Provider
	response, err := s.flightClient.SearchFlights(ctx, req)
	if response == nil || err != nil {
		// Return empty slice on failure/empty to avoid nil pointer issues
		return []Flight{}, Metadata{}, err
	}

	response.Metadata.CacheHit = false
	response.Metadata.CacheKey = cacheKey

	// Cache in background (Fire and Forget)
	// Use WithoutCancel so the cache write completes even if the HTTP request finishes early
	bgCtx := context.WithoutCancel(ctx)
	s.cacheFlightResponse(bgCtx, cacheKey, response)

	return response.Flights, response.Metadata, nil
}

func (s *Service) cacheFlightResponse(ctx context.Context, key string, resp *FlightSearchResponse) {
	go func() {
		data, err := json.Marshal(resp)
		if err != nil {
			s.logger.Error("cache_marshal_err", logger.Field{Key: "err", Value: err})
			return
		}
		if err := s.cache.Set(ctx, key, string(data), s.ttl); err != nil {
			s.logger.Error("cache_set_err", logger.Field{Key: "err", Value: err})
		}
	}()
}

func (s *Service) generateCacheKey(req SearchRequest) string {
	key := fmt.Sprintf("flight:%s:%s:%s:%d:%s",
		req.Origin,
		req.Destination,
		req.DepartureDate,
		req.Passengers,
		req.CabinClass,
	)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("flight:search:%x", hash[:16])
}

func (r SearchRequest) Validate() error {
	if len(r.Origin) != 3 {
		return fmt.Errorf("origin must be a 3-letter IATA code")
	}
	if len(r.Destination) != 3 {
		return fmt.Errorf("destination must be a 3-letter IATA code")
	}
	if strings.EqualFold(r.Origin, r.Destination) {
		return fmt.Errorf("origin and destination cannot be the same")
	}

	if r.Passengers < 1 {
		return fmt.Errorf("passengers must be at least 1")
	}
	if r.Passengers > 9 {
		return fmt.Errorf("cannot book more than 9 passengers in one search")
	}

	const layout = "2006-01-02" // YYYY-MM-DD format

	depTime, err := time.Parse(layout, r.DepartureDate)
	if err != nil {
		return fmt.Errorf("invalid departure_date format, expected YYYY-MM-DD")
	}

	// Ensure Departure isn't in the past
	// truncate 'now' to 00:00:00 so flights 'today' represent valid logic
	today := time.Now().Truncate(24 * time.Hour)
	if depTime.Before(today) {
		return fmt.Errorf("departure_date cannot be in the past")
	}

	if r.ReturnDate != "" {
		retTime, err := time.Parse(layout, r.ReturnDate)
		if err != nil {
			return fmt.Errorf("invalid return_date format, expected YYYY-MM-DD")
		}

		// Return Date cannot be before Departure Date
		if retTime.Before(depTime) {
			return fmt.Errorf("return_date cannot be before departure_date")
		}
	}

	return nil
}
