package flightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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

type airAsiaStop struct {
	Airport string `json:"airport"`
}

type airAsiaFlight struct {
	FlightCode    string        `json:"flight_code"`
	Airline       string        `json:"airline"`
	FromAirport   string        `json:"from_airport"`
	ToAirport     string        `json:"to_airport"`
	DepartTime    FlexibleTime  `json:"depart_time"`
	ArriveTime    FlexibleTime  `json:"arrive_time"`
	DurationHours float64       `json:"duration_hours"`
	DirectFlight  bool          `json:"direct_flight"`
	PriceIDR      uint64        `json:"price_idr"`
	Seats         uint32        `json:"seats"`
	CabinClass    string        `json:"cabin_class"`
	BaggageNote   string        `json:"baggage_note"`
	Stops         []airAsiaStop `json:"stops"`
}

func (a *AirAsiaClient) SearchFlights(ctx context.Context, req flight.SearchRequest) (*airAsiaFlightResponse, error) {
	url := fmt.Sprintf("%s/airasia/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("airasia: failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("airasia: failed to build request: %w", err)
	}

	r.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("airasia: external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("airasia: external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp airAsiaFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("airasia: failed to decode json response: %w", err)
	}

	return &apiResp, nil
}

func (f *FlightManager) mapAirAsiaFlights(resp *airAsiaFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Flights))

	for _, aaFlight := range resp.Flights {
		totalMinutes := uint32(math.Round(aaFlight.DurationHours * 60))
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		stopCount := uint32(0)
		if !aaFlight.DirectFlight {
			stopCount = uint32(len(aaFlight.Stops))
			if stopCount == 0 {
				stopCount = 1
			}
		}

		domainFlight := flight.Flight{
			ID:       aaFlight.FlightCode + "_" + aaFlight.Airline,
			Provider: "AirAsia",
			Airline: flight.Airline{
				Name: aaFlight.Airline,
				Code: aaFlight.FlightCode[0:2],
			},
			FlightNumber: aaFlight.FlightCode,
			Departure: flight.LocationTime{
				Airport:   aaFlight.FromAirport,
				Datetime:  aaFlight.DepartTime.Time,
				Timestamp: aaFlight.DepartTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   aaFlight.ToAirport,
				Datetime:  aaFlight.ArriveTime.Time,
				Timestamp: aaFlight.ArriveTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: stopCount,
			Price: flight.Price{
				Amount:   aaFlight.PriceIDR,
				Currency: "IDR",
			},
			AvailableSeats: aaFlight.Seats,
			CabinClass:     aaFlight.CabinClass,
			Baggage: flight.Baggage{
				Checked: aaFlight.BaggageNote,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}
