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

type lionAirFlightData struct {
	AvailableFlights []LionAirFlight `json:"available_flights"`
}

type LionAirFlightResponse struct {
	Data lionAirFlightData `json:"data"`
}

type lionAirCarrier struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type lionAirRoute struct {
	From lionAirLocation `json:"from"`
	To   lionAirLocation `json:"to"`
}

type lionAirSchedule struct {
	Departure         FlexibleTime `json:"departure"`
	DepartureTimezone string       `json:"departure_timezone"`
	Arrival           FlexibleTime `json:"arrival"`
	ArrivalTimezone   string       `json:"arrival_timezone"`
}

type lionAirLayover struct {
	Airport string `json:"airport"`
}

type lionAirPricing struct {
	Total    uint64 `json:"total"`
	Currency string `json:"currency"`
	FareType string `json:"fare_type"`
}

type lionAirBaggage struct {
	Cabin string `json:"cabin"`
	Hold  string `json:"hold"`
}

type lionAirServices struct {
	WifiAvailable    bool           `json:"wifi_available"`
	MealsIncluded    bool           `json:"meals_included"`
	BaggageAllowance lionAirBaggage `json:"baggage_allowance"`
}

type LionAirFlight struct {
	ID         string           `json:"id"`
	Carrier    lionAirCarrier   `json:"carrier"`
	Route      lionAirRoute     `json:"route"`
	Schedule   lionAirSchedule  `json:"schedule"`
	FlightTime uint32           `json:"flight_time"`
	IsDirect   bool             `json:"is_direct"`
	StopCount  uint32           `json:"stop_count,omitempty"`
	Layovers   []lionAirLayover `json:"layovers,omitempty"`
	Pricing    lionAirPricing   `json:"pricing"`
	SeatsLeft  uint32           `json:"seats_left"`
	PlaneType  string           `json:"plane_type"`
	Services   lionAirServices  `json:"services"`
}

type lionAirLocation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

func (a *LionAirClient) SearchFlights(ctx context.Context, req flight.SearchRequest) (*LionAirFlightResponse, error) {
	url := fmt.Sprintf("%s/lionair/v1/flights/search", a.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("lionair: failed to marshal request: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("lionair: failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("lionair: external api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lionair: external api returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp LionAirFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("lionair: failed to decode lionair response: %w", err)
	}

	return &apiResp, nil
}

func (f *FlightManager) mapLionAirFlights(resp *LionAirFlightResponse) ([]flight.Flight, error) {
	mapped := make([]flight.Flight, 0, len(resp.Data.AvailableFlights))

	for _, lFlight := range resp.Data.AvailableFlights {
		departureTime, err := f.applyTimezone(lFlight.Schedule.Departure.Time, lFlight.Schedule.DepartureTimezone)
		if err != nil {
			f.logger.Error("failed to apply departure timezone for lion air flight",
				logger.Field{Key: "flight_id", Value: lFlight.ID},
				logger.Field{Key: "timezone", Value: lFlight.Schedule.DepartureTimezone},
				logger.Field{Key: "err", Value: err})
			return nil, fmt.Errorf("lionair: failed to apply departure timezone: %w", err)
		}

		arrivalTime, err := f.applyTimezone(lFlight.Schedule.Arrival.Time, lFlight.Schedule.ArrivalTimezone)
		if err != nil {
			f.logger.Error("failed to apply arrival timezone for lion air flight",
				logger.Field{Key: "flight_id", Value: lFlight.ID},
				logger.Field{Key: "timezone", Value: lFlight.Schedule.ArrivalTimezone},
				logger.Field{Key: "err", Value: err})
			return nil, fmt.Errorf("lionair: failed to apply arrival timezone: %w", err)
		}

		totalMinutes := lFlight.FlightTime
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		stopCount := lFlight.StopCount
		if !lFlight.IsDirect && stopCount == 0 && len(lFlight.Layovers) > 0 {
			stopCount = uint32(len(lFlight.Layovers))
		}

		amenities := make([]string, 0)
		if lFlight.Services.WifiAvailable {
			amenities = append(amenities, "Wi-Fi")
		}
		if lFlight.Services.MealsIncluded {
			amenities = append(amenities, "Meal")
		}

		domainFlight := flight.Flight{
			ID:       lFlight.ID,
			Provider: lFlight.Carrier.Name,
			Airline: flight.Airline{
				Name: lFlight.Carrier.Name,
				Code: lFlight.Carrier.IATA,
			},
			FlightNumber: lFlight.ID,
			Departure: flight.LocationTime{
				Airport:   lFlight.Route.From.Code,
				City:      lFlight.Route.From.City,
				Datetime:  departureTime,
				Timestamp: departureTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   lFlight.Route.To.Code,
				City:      lFlight.Route.To.City,
				Datetime:  arrivalTime,
				Timestamp: arrivalTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: stopCount,
			Price: flight.Price{
				Amount:   lFlight.Pricing.Total,
				Currency: lFlight.Pricing.Currency,
			},
			AvailableSeats: lFlight.SeatsLeft,
			CabinClass:     lFlight.Pricing.FareType,
			Aircraft:       lFlight.PlaneType,
			Amenities:      amenities,
			Baggage: flight.Baggage{
				CarryOn: lFlight.Services.BaggageAllowance.Cabin,
				Checked: lFlight.Services.BaggageAllowance.Hold,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped, nil
}

// applyTimezone applies a timezone to a time.Time that was parsed without timezone info
func (f *FlightManager) applyTimezone(t time.Time, tzName string) (time.Time, error) {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %s: %w", tzName, err)
	}

	// Get the components of the time and recreate it in the specified timezone
	year, month, day := t.Date()
	hour, min, sec := t.Clock()

	return time.Date(year, month, day, hour, min, sec, t.Nanosecond(), loc), nil
}
