package flight

import (
	"context"
	"fmt"
)

func (s *Service) SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	flights, metadata, err := s.getOrFetchFlights(ctx, req)
	if err != nil {
		return nil, err
	}

	return &FlightSearchResponse{
		SearchCriteria: req,
		Metadata:       metadata,
		Flights:        flights,
	}, nil
}
