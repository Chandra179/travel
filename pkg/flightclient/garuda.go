package flightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"travel/internal/flight"
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
	Segments        []garudaSegment `json:"segments,omitempty"`
}

type garudaLocation struct {
	Airport  string       `json:"airport"`
	City     string       `json:"city"`
	Time     FlexibleTime `json:"time"`
	Terminal string       `json:"terminal"`
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

func (a *GarudaClient) SearchFlights(ctx context.Context, req flight.SearchRequest) (*garudaFlightResponse, error) {
	url := fmt.Sprintf("%s/garuda/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("garuda: failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("garuda: failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("garuda: external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("garuda: external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp garudaFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("garuda: failed to decode garuda response: %w", err)
	}

	return &apiResp, nil
}

func (f *FlightManager) mapGarudaFlights(resp *garudaFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Flights))

	for _, gFlight := range resp.Flights {
		hours := gFlight.DurationMinutes / 60
		minutes := gFlight.DurationMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		finalArrival := gFlight.Arrival
		if len(gFlight.Segments) > 0 {
			lastSegment := gFlight.Segments[len(gFlight.Segments)-1]
			finalArrival = lastSegment.Arrival
		}

		baggageCabin := fmt.Sprintf("Cabin: %d", gFlight.Baggage.CarryOn)
		baggageChecked := fmt.Sprintf("Checked: %d", gFlight.Baggage.Checked)

		domainFlight := flight.Flight{
			ID:       gFlight.FlightID + "_" + gFlight.Airline,
			Provider: gFlight.Airline,
			Airline: flight.Airline{
				Name: gFlight.Airline,
				Code: gFlight.AirlineCode,
			},
			FlightNumber: gFlight.FlightID,
			Departure: flight.LocationTime{
				Airport:   gFlight.Departure.Airport,
				Datetime:  gFlight.Departure.Time.Time,
				City:      gFlight.Departure.City,
				Timestamp: gFlight.Departure.Time.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   finalArrival.Airport,
				Datetime:  gFlight.Arrival.Time.Time,
				City:      gFlight.Arrival.City,
				Timestamp: gFlight.Arrival.Time.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: gFlight.DurationMinutes,
				Formatted:    formattedDuration,
			},
			Stops: gFlight.Stops,
			Price: flight.Price{
				Amount:   gFlight.Price.Amount,
				Currency: gFlight.Price.Currency,
			},
			AvailableSeats: gFlight.AvailableSeats,
			CabinClass:     gFlight.FareClass,
			Aircraft:       gFlight.Aircraft,
			Amenities:      gFlight.Amenities,
			Baggage: flight.Baggage{
				CarryOn: baggageCabin,
				Checked: baggageChecked,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}
