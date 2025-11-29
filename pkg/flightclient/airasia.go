package flightclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"travel/internal/flight"
	"travel/pkg/logger"
)

type AirAsiaClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logger.Client
}

func NewAirAsiaClient(httpClient *http.Client, baseURL string, logger logger.Client) *AirAsiaClient {
	return &AirAsiaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		logger:     logger,
	}
}

type airAsiaFlightResponse struct {
	Status  string          `json:"status"`
	Flights []airAsiaFlight `json:"flights"`
}

type airAsiaFlight struct {
	FlightCode    string    `json:"flight_code"`
	Airline       string    `json:"airline"`
	FromAirport   string    `json:"from_airport"`
	ToAirport     string    `json:"to_airport"`
	DepartTime    time.Time `json:"depart_time"`
	ArriveTime    time.Time `json:"arrive_time"`
	DurationHours float64   `json:"duration_hours"`
	DirectFlight  bool      `json:"direct_flight"`
	PriceIDR      uint64    `json:"price_idr"`
	Seats         uint32    `json:"seats"`
	CabinClass    string    `json:"cabin_class"`
	BaggageNote   string    `json:"baggage_note"`
	Stops         []struct {
		Airport string `json:"airport"`
	} `json:"stops"`
}

func (a *AirAsiaClient) SearchFlights(req flight.SearchRequest) (*airAsiaFlightResponse, error) {
	url := fmt.Sprintf("%s/airasia/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		a.logger.Error("failed to marshal request body", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		a.logger.Error("failed to build airasia request", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	r.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		serverMessage := string(bodyBytes)

		a.logger.Error("external api error",
			logger.Field{Key: "status", Value: resp.StatusCode},
			logger.Field{Key: "message", Value: serverMessage},
		)

		return nil, fmt.Errorf("external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp airAsiaFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		a.logger.Error("failed to decode airasia json response", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	return &apiResp, nil
}
