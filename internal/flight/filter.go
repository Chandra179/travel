package flight

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) FilterFlights(ctx context.Context, req FilterRequest) (*FlightSearchResponse, error) {
	startTime := time.Now()
	if err := req.SearchRequest.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	flights, metadata, err := s.getOrFetchFlights(ctx, req.SearchRequest)
	if err != nil {
		return nil, err
	}
	if req.Filters != nil {
		flights = s.applyFilters(flights, *req.Filters)
	}
	if req.Sort != nil {
		flights = s.applySorting(flights, *req.Sort)
	}
	metadata.TotalResults = uint32(len(flights))
	metadata.SearchTimeMs = uint32(time.Since(startTime).Milliseconds())

	return &FlightSearchResponse{
		SearchCriteria: req.SearchRequest,
		Metadata:       metadata,
		Flights:        flights,
	}, nil
}
