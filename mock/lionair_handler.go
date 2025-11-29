package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type LionAirResponse struct {
	Success bool        `json:"success"`
	Data    LionAirData `json:"data"`
}

type LionAirData struct {
	AvailableFlights []LionAirFlight `json:"available_flights"`
}

type LionAirFlight struct {
	ID         string        `json:"id"`
	Carrier    LionCarrier   `json:"carrier"`
	Route      LionRoute     `json:"route"`
	Schedule   LionSchedule  `json:"schedule"`
	FlightTime int           `json:"flight_time"`
	IsDirect   bool          `json:"is_direct"`
	StopCount  int           `json:"stop_count,omitempty"`
	Layovers   []LionLayover `json:"layovers,omitempty"`
	Pricing    LionPricing   `json:"pricing"`
	SeatsLeft  uint32        `json:"seats_left"`
	PlaneType  string        `json:"plane_type"`
	Services   LionServices  `json:"services"`
}

type LionCarrier struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type LionRoute struct {
	From LionLocation `json:"from"`
	To   LionLocation `json:"to"`
}

type LionLocation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type LionSchedule struct {
	Departure         string `json:"departure"` // "2025-12-15T05:30:00"
	DepartureTimezone string `json:"departure_timezone"`
	Arrival           string `json:"arrival"`
	ArrivalTimezone   string `json:"arrival_timezone"`
}

type LionPricing struct {
	Total    int    `json:"total"`
	Currency string `json:"currency"`
	FareType string `json:"fare_type"` // "ECONOMY"
}

type LionServices struct {
	WifiAvailable    bool `json:"wifi_available"`
	MealsIncluded    bool `json:"meals_included"`
	BaggageAllowance struct {
		Cabin string `json:"cabin"`
		Hold  string `json:"hold"`
	} `json:"baggage_allowance"`
}

type LionLayover struct {
	Airport         string `json:"airport"`
	DurationMinutes int    `json:"duration_minutes"`
}

var lionAirFlights = []LionAirFlight{
	{
		ID: "JT740", Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
		Route: LionRoute{
			From: LionLocation{Code: "CGK", Name: "Soekarno-Hatta International", City: "Jakarta"},
			To:   LionLocation{Code: "DPS", Name: "Ngurah Rai International", City: "Denpasar"},
		},
		Schedule: LionSchedule{
			Departure: "2025-12-15T05:30:00", DepartureTimezone: "Asia/Jakarta",
			Arrival: "2025-12-15T08:15:00", ArrivalTimezone: "Asia/Makassar",
		},
		FlightTime: 105, IsDirect: true,
		Pricing:   LionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
		SeatsLeft: 45, PlaneType: "Boeing 737-900ER",
		Services: LionServices{
			WifiAvailable: false, MealsIncluded: false,
			BaggageAllowance: struct {
				Cabin string `json:"cabin"`
				Hold  string `json:"hold"`
			}{Cabin: "7 kg", Hold: "20 kg"},
		},
	},
	{
		ID: "JT742", Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
		Route: LionRoute{
			From: LionLocation{Code: "CGK", Name: "Soekarno-Hatta International", City: "Jakarta"},
			To:   LionLocation{Code: "DPS", Name: "Ngurah Rai International", City: "Denpasar"},
		},
		Schedule: LionSchedule{
			Departure: "2025-12-15T11:45:00", DepartureTimezone: "Asia/Jakarta",
			Arrival: "2025-12-15T14:35:00", ArrivalTimezone: "Asia/Makassar",
		},
		FlightTime: 110, IsDirect: true,
		Pricing:   LionPricing{Total: 890000, Currency: "IDR", FareType: "ECONOMY"},
		SeatsLeft: 38, PlaneType: "Boeing 737-800",
		Services: LionServices{
			WifiAvailable: false, MealsIncluded: false,
			BaggageAllowance: struct {
				Cabin string `json:"cabin"`
				Hold  string `json:"hold"`
			}{Cabin: "7 kg", Hold: "20 kg"},
		},
	},
	{
		ID: "JT650", Carrier: LionCarrier{Name: "Lion Air", IATA: "JT"},
		Route: LionRoute{
			From: LionLocation{Code: "CGK", Name: "Soekarno-Hatta International", City: "Jakarta"},
			To:   LionLocation{Code: "DPS", Name: "Ngurah Rai International", City: "Denpasar"},
		},
		Schedule: LionSchedule{
			Departure: "2025-12-15T16:20:00", DepartureTimezone: "Asia/Jakarta",
			Arrival: "2025-12-15T21:10:00", ArrivalTimezone: "Asia/Makassar",
		},
		FlightTime: 230, IsDirect: false, StopCount: 1,
		Layovers:  []LionLayover{{Airport: "SUB", DurationMinutes: 75}},
		Pricing:   LionPricing{Total: 780000, Currency: "IDR", FareType: "ECONOMY"},
		SeatsLeft: 52, PlaneType: "Boeing 737-800",
		Services: LionServices{
			WifiAvailable: false, MealsIncluded: false,
			BaggageAllowance: struct {
				Cabin string `json:"cabin"`
				Hold  string `json:"hold"`
			}{Cabin: "7 kg", Hold: "20 kg"},
		},
	},
}

func LionAirSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Origin == "" || req.Destination == "" || req.DepartureDate == "" ||
		req.Passengers == 0 || req.CabinClass == "" {
		http.Error(w, "All fields are mandatory", http.StatusBadRequest)
		return
	}

	filtered := make([]LionAirFlight, 0)
	const lionTimeLayout = "2006-01-02T15:04:05"

	for _, f := range lionAirFlights {
		if !strings.EqualFold(f.Route.From.Code, req.Origin) || !strings.EqualFold(f.Route.To.Code, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.Pricing.FareType, req.CabinClass) {
			continue
		}

		if f.SeatsLeft < req.Passengers {
			continue
		}

		t, err := time.Parse(lionTimeLayout, f.Schedule.Departure)
		if err == nil {
			dbDate := t.Format("2006-01-02")
			if dbDate != req.DepartureDate {
				continue
			}
		}

		filtered = append(filtered, f)
	}

	w.Header().Set("Content-Type", "application/json")

	response := LionAirResponse{
		Success: true,
		Data: LionAirData{
			AvailableFlights: filtered,
		},
	}
	json.NewEncoder(w).Encode(response)
}
