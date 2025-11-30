package flightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	FlightNumber      string       `json:"flightNumber"`
	AirlineName       string       `json:"airlineName"`
	AirlineIATA       string       `json:"airlineIATA"`
	Origin            string       `json:"origin"`
	Destination       string       `json:"destination"`
	DepartureDateTime FlexibleTime `json:"departureDateTime"`
	ArrivalDateTime   FlexibleTime `json:"arrivalDateTime"`
	TravelTime        string       `json:"travelTime"`
	NumberOfStops     uint32       `json:"numberOfStops"`
	Fare              fare         `json:"fare"`
	SeatsAvailable    uint32       `json:"seatsAvailable"`
	AircraftModel     string       `json:"aircraftModel"`
	BaggageInfo       string       `json:"baggageInfo"`
	OnboardServices   []string     `json:"onboardServices"`
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
		return nil, fmt.Errorf("batikair: failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("batikair: failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("batikair: external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batikair: external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp batikAirFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("batikair: failed to decode batik response: %w", err)
	}

	return &apiResp, nil
}

func (f *FlightManager) mapBatikFlights(resp *batikAirFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Results))

	for _, btFlight := range resp.Results {
		totalMinutes, formattedDuration := f.parseBatikDuration(btFlight.TravelTime)

		domainFlight := flight.Flight{
			ID:       btFlight.FlightNumber + "_" + "BatikAir",
			Provider: btFlight.AirlineName,
			Airline: flight.Airline{
				Name: btFlight.AirlineName,
				Code: btFlight.AirlineIATA,
			},
			FlightNumber: btFlight.FlightNumber,
			Departure: flight.LocationTime{
				Airport:   btFlight.Origin,
				Datetime:  btFlight.DepartureDateTime.Time,
				Timestamp: btFlight.DepartureDateTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   btFlight.Destination,
				Datetime:  btFlight.ArrivalDateTime.Time,
				Timestamp: btFlight.ArrivalDateTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: btFlight.NumberOfStops,
			Price: flight.Price{
				Amount:   btFlight.Fare.TotalPrice,
				Currency: btFlight.Fare.CurrencyCode,
			},
			AvailableSeats: btFlight.SeatsAvailable,
			CabinClass:     btFlight.Fare.Class,
			Aircraft:       btFlight.AircraftModel,
			Amenities:      btFlight.OnboardServices,
			Baggage: flight.Baggage{
				Checked: btFlight.BaggageInfo,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}

func (f *FlightManager) parseBatikDuration(input string) (uint32, string) {
	cleanInput := strings.ReplaceAll(input, " ", "")
	d, err := time.ParseDuration(cleanInput)
	if err != nil {
		return 0, input
	}

	minutes := uint32(d.Minutes())
	h := minutes / 60
	m := minutes % 60
	return minutes, fmt.Sprintf("%dh %dm", h, m)
}
