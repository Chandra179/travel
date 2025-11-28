package flightclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

type airAsiaResponse struct {
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

func (a *AirAsiaClient) GetFlights() (*airAsiaResponse, error) {
	url := fmt.Sprintf("%s/v1/flights/search", a.baseURL)

	a.logger.Debug("starting airasia flight search",
		logger.Field{Key: "url", Value: url},
		logger.Field{Key: "method", Value: http.MethodPost},
	)

	body := bytes.NewBuffer([]byte(`{"a":"b"}`))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		a.logger.Error("failed to build airasia request", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("external api call to airasia failed",
			logger.Field{Key: "url", Value: url},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("external api call failed: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("received response from airasia",
		logger.Field{Key: "status_code", Value: resp.StatusCode},
	)

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("airasia api returned non-200 status",
			logger.Field{Key: "status_code", Value: resp.StatusCode},
			logger.Field{Key: "url", Value: url},
		)
		return nil, fmt.Errorf("external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp airAsiaResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		a.logger.Error("failed to decode airasia json response", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	a.logger.Info("successfully retrieved flights from airasia",
		logger.Field{Key: "flight_count", Value: len(apiResp.Flights)},
		logger.Field{Key: "provider_status", Value: apiResp.Status},
	)

	return &apiResp, nil
}
