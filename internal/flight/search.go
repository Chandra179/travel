package flight

import (
	"context"
)

func (s *Service) SearchFlights(ctx context.Context, req SearchRequest) (*FlightSearchResponse, error) {
	flights, metadata, err := s.getOrFetchFlights(ctx, req)
	if err != nil {
		return nil, err
	}

	return &FlightSearchResponse{
		Metadata: metadata,
		Flights:  flights,
	}, nil
}
