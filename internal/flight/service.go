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
		return NewError(ErrorCodeValidation, "origin must be a 3-letter IATA code", 400)
	}
	if len(r.Destination) != 3 {
		return NewError(ErrorCodeValidation, "destination must be a 3-letter IATA code", 400)
	}
	if strings.EqualFold(r.Origin, r.Destination) {
		return NewError(ErrorCodeSameOriginDestination, "origin and destination cannot be the same", 400)
	}

	if r.Passengers < 1 {
		return NewError(ErrorCodeInvalidPassengerCount, "passengers must be at least 1", 400)
	}
	if r.Passengers > 9 {
		return NewError(ErrorCodeInvalidPassengerCount, "cannot book more than 9 passengers in one search", 400)
	}

	const layout = "2006-01-02"

	depTime, err := time.Parse(layout, r.DepartureDate)
	if err != nil {
		return NewError(ErrorCodeInvalidDateFormat, "invalid departure_date format, expected YYYY-MM-DD", 400)
	}

	today := time.Now().Truncate(24 * time.Hour)
	if depTime.Before(today) {
		return NewError(ErrorCodeDeparturePast, "departure_date cannot be in the past", 400)
	}

	if r.ReturnDate != "" {
		retTime, err := time.Parse(layout, r.ReturnDate)
		if err != nil {
			return NewError(ErrorCodeInvalidDateFormat, "invalid return_date format, expected YYYY-MM-DD", 400)
		}

		if retTime.Before(depTime) {
			return NewError(ErrorCodeReturnBeforeDeparture, "return_date cannot be before departure_date", 400)
		}
	}

	return nil
}
