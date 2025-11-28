package flightclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"travel/pkg/logger"
)

type GarudaClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logger.Client
}

func NewGarudaClient(httpClient *http.Client, baseURL string, logger logger.Client) *GarudaClient {
	return &GarudaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		logger:     logger,
	}
}

type garudaFlightResponse struct {
	Status  string         `json:"status"`
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID        string          `json:"flight_id"`
	Airline         string          `json:"airline"`
	AirlineCode     string          `json:"airline_code"`
	Departure       garudaLocation  `json:"departure"`
	Arrival         garudaLocation  `json:"arrival"`
	DurationMinutes uint32          `json:"duration_minutes"`
	Stops           uint32          `json:"stops"`
	Aircraft        string          `json:"aircraft"`
	Price           garudaPrice     `json:"price"`
	AvailableSeats  uint32          `json:"available_seats"`
	FareClass       string          `json:"fare_class"`
	Baggage         garudaBaggage   `json:"baggage"`
	Amenities       []string        `json:"amenities"`
	Segments        []garudaSegment `json:"segments,omitempty"` // Handle complex flights
}

type garudaLocation struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"` // ISO 8601
	Terminal string `json:"terminal"`
}

type garudaPrice struct {
	Amount   uint64 `json:"amount"`
	Currency string `json:"currency"`
}

type garudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type garudaSegment struct {
	FlightNumber string         `json:"flight_number"`
	Departure    garudaLocation `json:"departure"`
	Arrival      garudaLocation `json:"arrival"`
}

func (a *GarudaClient) GetFlights() (*garudaFlightResponse, error) {
	url := fmt.Sprintf("%s/garuda/v1/flights/search", a.baseURL)

	body := bytes.NewBuffer([]byte(`{}`))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		a.logger.Error("failed to build garuda request", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp garudaFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode garuda response: %w", err)
	}

	return &apiResp, nil
}
