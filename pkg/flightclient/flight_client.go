package flightclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"travel/internal/flight"
	"travel/pkg/logger"
)

type FlightManager struct {
	airAsiaClient  *AirAsiaClient
	batikAirClient *BatikAirClient
	garudaClient   *GarudaClient
	lionAirClient  *LionAirClient
	logger         logger.Client
}

func NewFlightClient(airAsiaClient *AirAsiaClient, batikAirClient *BatikAirClient,
	garudaClient *GarudaClient, lionAirClient *LionAirClient, logger logger.Client) *FlightManager {
	return &FlightManager{
		airAsiaClient:  airAsiaClient,
		batikAirClient: batikAirClient,
		garudaClient:   garudaClient,
		lionAirClient:  lionAirClient,
		logger:         logger,
	}
}

type providerResult struct {
	provider  string
	flights   []flight.Flight
	err       error
	errorCode flight.ErrorCode
}

func (f *FlightManager) SearchFlights(ctx context.Context, req flight.SearchRequest) (*flight.FlightSearchResponse, error) {
	// TODO: Flights context timeout (moved to .env)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resultChan := make(chan providerResult, 4)
	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		defer wg.Done()
		resp, err := f.airAsiaClient.SearchFlights(ctx, req)
		if err != nil {
			errCode := categorizeError(err)
			f.logger.Error("failed to fetch airasia", logger.Field{Key: "err", Value: err.Error()})
			resultChan <- providerResult{provider: "AirAsia", err: err, errorCode: errCode}
			return
		}
		flights := f.mapAirAsiaFlights(resp)
		resultChan <- providerResult{provider: "AirAsia", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.batikAirClient.SearchFlights(ctx, req)
		if err != nil {
			errCode := categorizeError(err)
			f.logger.Error("failed to fetch batik", logger.Field{Key: "err", Value: err.Error()})
			resultChan <- providerResult{provider: "Batik Air", err: err, errorCode: errCode}
			return
		}
		flights := f.mapBatikFlights(resp)
		resultChan <- providerResult{provider: "Batik Air", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.garudaClient.SearchFlights(ctx, req)
		if err != nil {
			errCode := categorizeError(err)
			f.logger.Error("failed to fetch garuda", logger.Field{Key: "err", Value: err.Error()})
			resultChan <- providerResult{provider: "Garuda Indonesia", err: err, errorCode: errCode}
			return
		}
		flights := f.mapGarudaFlights(resp)
		resultChan <- providerResult{provider: "Garuda Indonesia", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.lionAirClient.SearchFlights(ctx, req)
		if err != nil {
			errCode := categorizeError(err)
			f.logger.Error("failed to fetch lion air", logger.Field{Key: "err", Value: err.Error()})
			resultChan <- providerResult{provider: "Lion Air", err: err, errorCode: errCode}
			return
		}
		flights, err := f.mapLionAirFlights(resp)
		if err != nil {
			errCode := categorizeError(err)
			f.logger.Error("failed to map lion air flights", logger.Field{Key: "err", Value: err.Error()})
			resultChan <- providerResult{provider: "Lion Air", err: err, errorCode: errCode}
			return
		}
		resultChan <- providerResult{provider: "Lion Air", flights: flights}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allFlights []flight.Flight
	var providerErrors []flight.ProviderError
	providersSucceeded := uint32(0)
	providersFailed := uint32(0)
	providersQueried := uint32(4)

	for result := range resultChan {
		if result.err == nil {
			allFlights = append(allFlights, result.flights...)
			providersSucceeded++
		} else {
			providersFailed++
			providerErrors = append(providerErrors, flight.ProviderError{
				Provider: result.provider,
				Code:     result.errorCode,
			})
		}
	}

	return &flight.FlightSearchResponse{
		Flights: allFlights,
		Metadata: flight.Metadata{
			TotalResults:       uint32(len(allFlights)),
			ProvidersQueried:   providersQueried,
			ProvidersSucceeded: providersSucceeded,
			ProvidersFailed:    providersFailed,
			ProviderErrors:     providerErrors,
		},
	}, nil
}

func categorizeError(err error) flight.ErrorCode {
	if err == nil {
		return ""
	}
	errMsg := err.Error()

	if errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") {
		return flight.ErrorCodeTimeout
	}

	return flight.ErrorCodeInternalFailure
}

// FlexibleTime handles multiple time formats from different airline providers
type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	formats := []string{
		time.RFC3339,               // Standard: 2006-01-02T15:04:05Z07:00 (AirAsia, Garuda)
		"2006-01-02T15:04:05-0700", // Batik Air: 2025-12-15T07:15:00+0700
		"2006-01-02T15:04:05",      // Lion Air: 2025-12-15T05:30:00 (no timezone)
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			ft.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse time: %s", s)
}
