package flight

import (
	"context"
	"encoding/json"
	"time"
	"travel/pkg/logger"
)

func (s *Service) SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error) {
	cacheKey := s.generateCacheKey(req)
	cached, err := s.cache.Get(ctx, cacheKey)

	// Return immedeatly if response exist in cache
	if err == nil && cached != "" {
		var response FlightSearchResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			response.Metadata.CacheHit = true
			response.Metadata.CacheKey = cacheKey
			return &response, nil
		}
		s.logger.Error("SearchFlights", logger.Field{Key: "err", Value: err})
	}

	startTime := time.Now()
	response, err := s.flightClient.SearchFlights(ctx, req)
	if err != nil {
		return nil, err
	}

	searchTime := time.Since(startTime).Milliseconds()
	response.Metadata.SearchTimeMs = uint32(searchTime)
	response.Metadata.CacheHit = false
	response.Metadata.CacheKey = cacheKey

	responseBytes, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("SearchFlights", logger.Field{Key: "err_marshal", Value: err})
		return response, nil // Return response even if caching fails
	}

	if err := s.cache.Set(ctx, cacheKey, string(responseBytes), s.ttl); err != nil {
		s.logger.Error("SearchFlights", logger.Field{Key: "err_set_cache", Value: err})
	}

	return response, nil
}
