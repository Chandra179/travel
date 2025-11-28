package flight

import (
	"time"
)

type FlightClient interface {
	GetFlights() (*FlightSearchResponse, error)
}

type Service struct {
	flightClient FlightClient
}

func NewFlightService(flightClient FlightClient) *Service {
	return &Service{
		flightClient: flightClient,
	}
}

type PriceRange struct {
	Low  uint64 `json:"low"`
	High uint64 `json:"high"`
}

type ArrivalTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DepartureTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type SearchRequest struct {
	FlightDuration uint32        `json:"flight_duration"`
	StopCount      uint32        `json:"stop_count"`
	Passengers     uint32        `json:"passengers"`
	CabinClass     string        `json:"cabin_class"`
	Origin         string        `json:"origin"`
	Destination    string        `json:"destination"`
	DepartureDate  string        `json:"departure_date"`
	ReturnDate     string        `json:"return_date"`
	SortBy         string        `json:"sort_by"`
	Currency       string        `json:"currency"`
	Airlines       string        `json:"airlines"`
	DepartureTime  DepartureTime `json:"departure_time"`
	ArrivalTime    ArrivalTime   `json:"arrival_time"`
	PriceRange     PriceRange    `json:"price_range"`
}

type FlightSearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    uint32 `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       uint32 `json:"total_results"`
	ProvidersQueried   uint32 `json:"providers_queried"`
	ProvidersSucceeded uint32 `json:"providers_succeeded"`
	ProvidersFailed    uint32 `json:"providers_failed"`
	SearchTimeMs       uint32 `json:"search_time_ms"`
	CacheHit           bool   `json:"cache_hit"`
}

type Flight struct {
	ID             string       `json:"id"`
	Provider       string       `json:"provider"`
	Airline        Airline      `json:"airline"`
	FlightNumber   string       `json:"flight_number"`
	Departure      LocationTime `json:"departure"`
	Arrival        LocationTime `json:"arrival"`
	Duration       Duration     `json:"duration"`
	Stops          uint32       `json:"stops"`
	Price          Price        `json:"price"`
	AvailableSeats uint32       `json:"available_seats"`
	CabinClass     string       `json:"cabin_class"`
	Aircraft       string       `json:"aircraft"`
	Amenities      []string     `json:"amenities"`
	Baggage        Baggage      `json:"baggage"`
}

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type LocationTime struct {
	Airport   string    `json:"airport"`
	City      string    `json:"city"`
	Datetime  time.Time `json:"datetime"`  // Go's time.Time handles the ISO 8601 format and timezone
	Timestamp int64     `json:"timestamp"` // Unix epoch time (int64)
}

type Duration struct {
	TotalMinutes uint32 `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount   uint64 `json:"amount"` // Stored in minor units (e.g., IDR or cents)
	Currency string `json:"currency"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

func (c *Service) SearchFlights(req SearchRequest) (*FlightSearchResponse, error) {
	return c.flightClient.GetFlights()
}
