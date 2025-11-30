package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type GarudaResponse struct {
	Status  string         `json:"status"`
	Flights []GarudaFlight `json:"flights"`
}

type GarudaFlight struct {
	FlightID        string          `json:"flight_id"`
	Airline         string          `json:"airline"`
	AirlineCode     string          `json:"airline_code"`
	Departure       GarudaLocation  `json:"departure"`
	Arrival         GarudaLocation  `json:"arrival"`
	DurationMinutes int             `json:"duration_minutes"`
	Stops           int             `json:"stops"`
	Aircraft        string          `json:"aircraft"`
	Price           GarudaPrice     `json:"price"`
	AvailableSeats  uint32          `json:"available_seats"`
	FareClass       string          `json:"fare_class"` // "economy", "business"
	Baggage         GarudaBaggage   `json:"baggage"`
	Amenities       []string        `json:"amenities"`
	Segments        []GarudaSegment `json:"segments,omitempty"`
}

type GarudaLocation struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type GarudaPrice struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type GarudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type GarudaSegment struct {
	FlightNumber    string         `json:"flight_number"`
	Departure       GarudaLocation `json:"departure"`
	Arrival         GarudaLocation `json:"arrival"`
	DurationMinutes int            `json:"duration_minutes"`
	LayoverMinutes  int            `json:"layover_minutes,omitempty"`
}

func GarudaSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Read JSON file
	data, err := os.ReadFile("mock/files/garuda_indonesia_search_response.json")
	if err != nil {
		http.Error(w, "Failed to read flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal to struct
	var fileResponse GarudaResponse
	if err := json.Unmarshal(data, &fileResponse); err != nil {
		http.Error(w, "Failed to parse flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filtering
	filtered := make([]GarudaFlight, 0)

	for _, f := range fileResponse.Flights {
		if req.Origin != "" && !strings.EqualFold(f.Departure.Airport, req.Origin) {
			continue
		}

		// Check destination (including segments for connecting flights)
		if req.Destination != "" {
			matchDest := false

			if strings.EqualFold(f.Arrival.Airport, req.Destination) {
				matchDest = true
			} else if len(f.Segments) > 0 {
				lastSeg := f.Segments[len(f.Segments)-1]
				if strings.EqualFold(lastSeg.Arrival.Airport, req.Destination) {
					matchDest = true
				}
			}

			if !matchDest {
				continue
			}
		}

		if req.CabinClass != "" && !strings.EqualFold(f.FareClass, req.CabinClass) {
			continue
		}

		if req.Passengers > 0 && f.AvailableSeats < req.Passengers {
			continue
		}

		if req.DepartureDate != "" {
			t, err := time.Parse(time.RFC3339, f.Departure.Time)
			if err == nil {
				dbDate := t.Format("2006-01-02")
				if dbDate != req.DepartureDate {
					continue
				}
			}
		}

		filtered = append(filtered, f)
	}

	delay := 50 + rand.Intn(51) // 50 to 100ms
	time.Sleep(time.Duration(delay) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GarudaResponse{Status: "success", Flights: filtered})
}
