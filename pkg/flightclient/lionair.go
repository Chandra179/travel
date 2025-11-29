package flightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"travel/internal/flight"
	"travel/pkg/logger"
)

type LionAirClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logger.Client
}

func NewLionAirClient(httpClient *http.Client, baseURL string, logger logger.Client) *LionAirClient {
	return &LionAirClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		logger:     logger,
	}
}

type lionAirFlightResponse struct {
	Data struct {
		AvailableFlights []lionAirFlight `json:"available_flights"`
	} `json:"data"`
}

type lionAirFlight struct {
	ID      string `json:"id"`
	Carrier struct {
		Name string `json:"name"`
		IATA string `json:"iata"`
	} `json:"carrier"`
	Route struct {
		From lionAirLocation `json:"from"`
		To   lionAirLocation `json:"to"`
	} `json:"route"`
	Schedule struct {
		Departure         time.Time `json:"departure"`
		DepartureTimezone string    `json:"departure_timezone"`
		Arrival           time.Time `json:"arrival"`
		ArrivalTimezone   string    `json:"arrival_timezone"`
	} `json:"schedule"`
	FlightTime uint32 `json:"flight_time"` // In minutes
	IsDirect   bool   `json:"is_direct"`
	StopCount  uint32 `json:"stop_count,omitempty"`
	Layovers   []struct {
		Airport string `json:"airport"`
	} `json:"layovers,omitempty"`
	Pricing struct {
		Total    uint64 `json:"total"`
		Currency string `json:"currency"`
		FareType string `json:"fare_type"`
	} `json:"pricing"`
	SeatsLeft uint32 `json:"seats_left"`
	PlaneType string `json:"plane_type"`
	Services  struct {
		WifiAvailable    bool `json:"wifi_available"`
		MealsIncluded    bool `json:"meals_included"`
		BaggageAllowance struct {
			Cabin string `json:"cabin"`
			Hold  string `json:"hold"`
		} `json:"baggage_allowance"`
	} `json:"services"`
}

type lionAirLocation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

func (a *LionAirClient) SearchFlights(ctx context.Context, req flight.SearchRequest) (*lionAirFlightResponse, error) {
	url := fmt.Sprintf("%s/lionair/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp lionAirFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode lionair response: %w", err)
	}

	return &apiResp, nil
}
