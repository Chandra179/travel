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

func NewService(flightClient FlightClient, cache cache.Cache, ttlMinutes int, logger logger.Client) *Service {
	return &Service{
		flightClient: flightClient,
		cache:        cache,
		ttl:          time.Duration(ttlMinutes) * time.Minute,
		logger:       logger,
	}
}

type PriceRange struct {
	Low  uint64 `json:"low"`
	High uint64 `json:"high"`
}

type ArrivalTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DepartureTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type SearchRequest struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	ReturnDate    string `json:"return_date"`
	Passengers    uint32 `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type FlightSearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    uint32 `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       uint32 `json:"total_results"`
	ProvidersQueried   uint32 `json:"providers_queried"`
	ProvidersSucceeded uint32 `json:"providers_succeeded"`
	ProvidersFailed    uint32 `json:"providers_failed"`
	SearchTimeMs       uint32 `json:"search_time_ms"`
	CacheKey           string `json:"cache_key"`
	CacheHit           bool   `json:"cache_hit"`
}

type Flight struct {
	ID             string       `json:"id"`
	Provider       string       `json:"provider"`
	Airline        Airline      `json:"airline"`
	FlightNumber   string       `json:"flight_number"`
	Departure      LocationTime `json:"departure"`
	Arrival        LocationTime `json:"arrival"`
	Duration       Duration     `json:"duration"`
	Stops          uint32       `json:"stops"`
	Price          Price        `json:"price"`
	AvailableSeats uint32       `json:"available_seats"`
	CabinClass     string       `json:"cabin_class"`
	Aircraft       string       `json:"aircraft"`
	Amenities      []string     `json:"amenities"`
	Baggage        Baggage      `json:"baggage"`
}

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type LocationTime struct {
	Airport   string    `json:"airport"`
	City      string    `json:"city"`
	Datetime  time.Time `json:"datetime"`
	Timestamp int64     `json:"timestamp"`
}

type Duration struct {
	TotalMinutes uint32 `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount   uint64 `json:"amount"`
	Currency string `json:"currency"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

type FilterOptions struct {
	PriceRange    *PriceRange    `json:"price_range,omitempty"`
	MaxStops      *uint32        `json:"max_stops,omitempty"`
	DepartureTime *DepartureTime `json:"departure_time,omitempty"`
	ArrivalTime   *ArrivalTime   `json:"arrival_time,omitempty"`
	Airlines      []string       `json:"airlines,omitempty"`
	MaxDuration   *uint32        `json:"max_duration,omitempty"`
}

type FilterRequest struct {
	SearchRequest
	Filters FilterOptions `json:"filters"`
}

// generateCacheKey creates a deterministic key from search parameters
func (s *Service) generateCacheKey(req SearchRequest) string {
	key := fmt.Sprintf("flight:%s:%s:%s:%s:%d:%s",
		req.Origin,
		req.Destination,
		req.DepartureDate,
		req.ReturnDate,
		req.Passengers,
		req.CabinClass,
	)

	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("flight:search:%x", hash[:16])
}

func (s *Service) SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error) {
	cacheKey := s.generateCacheKey(req)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		s.logger.Info("Cache hit for search", logger.Field{Key: "cache_key", Value: cacheKey})

		var response FlightSearchResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			response.Metadata.CacheHit = true
			response.Metadata.CacheKey = cacheKey
			return &response, nil
		}
		s.logger.Error("Failed to unmarshal cached data", logger.Field{Key: "err", Value: err})
	}

	// Cache miss - fetch from providers
	s.logger.Info("Cache miss for search", logger.Field{Key: "cache_key", Value: cacheKey})

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
		s.logger.Error("Failed to marshal response", logger.Field{Key: "err", Value: err})
		return response, nil // Return response even if caching fails
	}

	if err := s.cache.Set(ctx, cacheKey, string(responseBytes), s.ttl); err != nil {
		s.logger.Error("Failed to cache response", logger.Field{Key: "err", Value: err})
	}

	return response, nil
}

// FilterFlights - Filter with auto-refresh on cache miss
func (s *Service) FilterFlights(ctx context.Context, req FilterRequest) (*FlightSearchResponse, error) {
	cacheKey := s.generateCacheKey(req.SearchRequest)
	cached, err := s.cache.Get(ctx, cacheKey)

	if err == nil && cached != "" {
		s.logger.Info("Cache hit for filter", logger.Field{Key: "cache_key", Value: cacheKey})

		var response FlightSearchResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			filteredFlights := s.applyFilters(response.Flights, req.Filters)

			return &FlightSearchResponse{
				SearchCriteria: response.SearchCriteria,
				Metadata: Metadata{
					TotalResults:       uint32(len(filteredFlights)),
					ProvidersQueried:   response.Metadata.ProvidersQueried,
					ProvidersSucceeded: response.Metadata.ProvidersSucceeded,
					ProvidersFailed:    response.Metadata.ProvidersFailed,
					SearchTimeMs:       response.Metadata.SearchTimeMs,
					CacheKey:           cacheKey,
					CacheHit:           true,
				},
				Flights: filteredFlights,
			}, nil
		}
		s.logger.Error("Failed to unmarshal cached data", logger.Field{Key: "err", Value: err})
	}

	s.logger.Info("Cache miss for filter - auto-refreshing",
		logger.Field{Key: "cache_key", Value: cacheKey},
		logger.Field{Key: "route", Value: fmt.Sprintf("%s->%s", req.Origin, req.Destination)},
	)

	startTime := time.Now()
	response, err := s.flightClient.SearchFlights(ctx, req.SearchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh search results: %w", err)
	}

	searchTime := time.Since(startTime).Milliseconds()
	response.Metadata.SearchTimeMs = uint32(searchTime)
	response.Metadata.CacheHit = false
	response.Metadata.CacheKey = cacheKey

	// Cache in background (non-blocking
	go func() {
		bgCtx := context.Background()

		responseBytes, err := json.Marshal(response)
		if err != nil {
			s.logger.Error("Failed to marshal response for caching",
				logger.Field{Key: "err", Value: err},
				logger.Field{Key: "cache_key", Value: cacheKey},
			)
			return
		}

		if err := s.cache.Set(bgCtx, cacheKey, string(responseBytes), s.ttl); err != nil {
			s.logger.Error("Failed to cache refreshed results",
				logger.Field{Key: "err", Value: err},
				logger.Field{Key: "cache_key", Value: cacheKey},
			)
		} else {
			s.logger.Info("Cached refreshed search results in background",
				logger.Field{Key: "cache_key", Value: cacheKey},
				logger.Field{Key: "ttl_minutes", Value: s.ttl.Minutes()},
			)
		}
	}()

	filteredFlights := s.applyFilters(response.Flights, req.Filters)

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
			SearchTimeMs:       uint32(searchTime),
			CacheKey:           cacheKey,
			CacheHit:           false, // Was a cache miss, had to refresh
		},
		Flights: filteredFlights,
	}, nil
}

func (s *Service) applyFilters(flights []Flight, filters FilterOptions) []Flight {
	filtered := make([]Flight, 0, len(flights))

	for _, f := range flights {
		if filters.PriceRange != nil {
			if f.Price.Amount < filters.PriceRange.Low || f.Price.Amount > filters.PriceRange.High {
				continue
			}
		}

		if filters.MaxStops != nil && f.Stops > *filters.MaxStops {
			continue
		}

		if filters.DepartureTime != nil {
			depTime := f.Departure.Datetime.Format("15:04")
			if depTime < filters.DepartureTime.From || depTime > filters.DepartureTime.To {
				continue
			}
		}

		if filters.ArrivalTime != nil {
			arrTime := f.Arrival.Datetime.Format("15:04")
			if arrTime < filters.ArrivalTime.From || arrTime > filters.ArrivalTime.To {
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
