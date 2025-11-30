package flight

import "time"

type ErrorCode string

const (
	ErrorCodeTimeout         ErrorCode = "TIMEOUT"
	ErrorCodeInternalFailure ErrorCode = "INTERNAL_FAILURE"
)

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
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	ReturnDate    string `json:"return_date"`
	Passengers    uint32 `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type FlightSearchResponse struct {
	Metadata Metadata `json:"metadata"`
	Flights  []Flight `json:"flights"`
}

type ProviderError struct {
	Provider string    `json:"provider"`
	Code     ErrorCode `json:"code"`
}

type Metadata struct {
	TotalResults       uint32          `json:"total_results"`
	ProvidersQueried   uint32          `json:"providers_queried"`
	ProvidersSucceeded uint32          `json:"providers_succeeded"`
	ProvidersFailed    uint32          `json:"providers_failed"`
	ProviderErrors     []ProviderError `json:"provider_errors,omitempty"`
	SearchTimeMs       uint32          `json:"search_time_ms,omitempty"`
	CacheHit           bool            `json:"cache_hit"`
	CacheKey           string          `json:"cache_key,omitempty"`
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
	BestValueScore *float64     `json:"best_value_score,omitempty"`
}

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type LocationTime struct {
	Airport   string    `json:"airport"`
	City      string    `json:"city"`
	Datetime  time.Time `json:"datetime"`
	Timestamp int64     `json:"timestamp"`
}

type Duration struct {
	TotalMinutes uint32 `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount   uint64 `json:"amount"`
	Currency string `json:"currency"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

type FilterOptions struct {
	PriceRange    *PriceRange    `json:"price_range,omitempty"`
	MaxStops      *uint32        `json:"max_stops,omitempty"`
	DepartureTime *DepartureTime `json:"departure_time,omitempty"`
	ArrivalTime   *ArrivalTime   `json:"arrival_time,omitempty"`
	Airlines      []string       `json:"airlines,omitempty"`
	MaxDuration   *uint32        `json:"max_duration,omitempty"`
}

type SortOptions struct {
	By    string `json:"by"`    // price, duration, departure_time, arrival_time, best_value
	Order string `json:"order"` // asc, desc
}

type FilterRequest struct {
	SearchRequest
	Filters *FilterOptions `json:"filters,omitempty"`
	Sort    *SortOptions   `json:"sort,omitempty"`
}
