package main

import (
	"encoding/json"
	"net/http"
	"os"
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

func LionAirSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Read JSON file
	data, err := os.ReadFile("mock/files/lion_air_search_response.json")
	if err != nil {
		http.Error(w, "Failed to read flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal to struct
	var fileResponse LionAirResponse
	if err := json.Unmarshal(data, &fileResponse); err != nil {
		http.Error(w, "Failed to parse flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filtering
	filtered := make([]LionAirFlight, 0)
	const lionTimeLayout = "2006-01-02T15:04:05"

	for _, f := range fileResponse.Data.AvailableFlights {
		if req.Origin != "" && !strings.EqualFold(f.Route.From.Code, req.Origin) {
			continue
		}

		if req.Destination != "" && !strings.EqualFold(f.Route.To.Code, req.Destination) {
			continue
		}

		if req.CabinClass != "" && !strings.EqualFold(f.Pricing.FareType, req.CabinClass) {
			continue
		}

		if req.Passengers > 0 && f.SeatsLeft < req.Passengers {
			continue
		}

		if req.DepartureDate != "" {
			t, err := time.Parse(lionTimeLayout, f.Schedule.Departure)
			if err == nil {
				dbDate := t.Format("2006-01-02")
				if dbDate != req.DepartureDate {
					continue
				}
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
