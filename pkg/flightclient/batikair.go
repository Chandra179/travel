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

type BatikAirClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logger.Client
}

func NewBatikAirClient(httpClient *http.Client, baseURL string, logger logger.Client) *BatikAirClient {
	return &BatikAirClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		logger:     logger,
	}
}

type batikAirFlightResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Results []batikAirFlight `json:"results"`
}

type batikAirFlight struct {
	FlightNumber      string    `json:"flightNumber"`
	AirlineName       string    `json:"airlineName"`
	AirlineIATA       string    `json:"airlineIATA"`
	Origin            string    `json:"origin"`
	Destination       string    `json:"destination"`
	DepartureDateTime time.Time `json:"departureDateTime"`
	ArrivalDateTime   time.Time `json:"arrivalDateTime"`
	TravelTime        string    `json:"travelTime"`
	NumberOfStops     uint32    `json:"numberOfStops"`
	Fare              fare      `json:"fare"`
	SeatsAvailable    uint32    `json:"seatsAvailable"`
	AircraftModel     string    `json:"aircraftModel"`
	BaggageInfo       string    `json:"baggageInfo"`
	OnboardServices   []string  `json:"onboardServices"`
}

type fare struct {
	BasePrice    uint64 `json:"basePrice"`
	Taxes        uint64 `json:"taxes"`
	TotalPrice   uint64 `json:"totalPrice"`
	CurrencyCode string `json:"currencyCode"`
	Class        string `json:"class"`
}

func (a *BatikAirClient) SearchFlights(ctx context.Context, req flight.SearchRequest) (*batikAirFlightResponse, error) {
	url := fmt.Sprintf("%s/batikair/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		a.logger.Error("failed to marshal request body", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		a.logger.Error("failed to build batik request", logger.Field{Key: "error", Value: err})
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

	var apiResp batikAirFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode batik response: %w", err)
	}

	return &apiResp, nil
}
