package flight

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
