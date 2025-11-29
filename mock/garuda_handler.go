package main

import (
	"encoding/json"
	"net/http"
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

var garudaFlights = []GarudaFlight{
	{
		FlightID: "GA400", Airline: "Garuda Indonesia", AirlineCode: "GA",
		Departure:       GarudaLocation{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T06:00:00+07:00", Terminal: "3"},
		Arrival:         GarudaLocation{Airport: "DPS", City: "Denpasar", Time: "2025-12-15T08:50:00+08:00", Terminal: "I"},
		DurationMinutes: 110, Stops: 0, Aircraft: "Boeing 737-800",
		Price:          GarudaPrice{Amount: 1250000, Currency: "IDR"},
		AvailableSeats: 28, FareClass: "economy",
		Baggage: GarudaBaggage{CarryOn: 1, Checked: 2}, Amenities: []string{"wifi", "meal", "entertainment"},
	},
	{
		FlightID: "GA410", Airline: "Garuda Indonesia", AirlineCode: "GA",
		Departure:       GarudaLocation{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T09:30:00+07:00", Terminal: "3"},
		Arrival:         GarudaLocation{Airport: "DPS", City: "Denpasar", Time: "2025-12-15T12:25:00+08:00", Terminal: "I"},
		DurationMinutes: 115, Stops: 0, Aircraft: "Airbus A330-300",
		Price:          GarudaPrice{Amount: 1450000, Currency: "IDR"},
		AvailableSeats: 15, FareClass: "economy",
		Baggage: GarudaBaggage{CarryOn: 1, Checked: 2}, Amenities: []string{"wifi", "power_outlet", "meal", "entertainment"},
	},
	{
		// This is the connecting flight (CGK->SUB->DPS)
		FlightID: "GA315", Airline: "Garuda Indonesia", AirlineCode: "GA",
		Departure:       GarudaLocation{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T14:00:00+07:00", Terminal: "3"},
		Arrival:         GarudaLocation{Airport: "SUB", City: "Surabaya", Time: "2025-12-15T15:30:00+07:00", Terminal: "2"},
		DurationMinutes: 90, Stops: 1, Aircraft: "Boeing 737",
		Price: GarudaPrice{Amount: 1850000, Currency: "IDR"},
		Segments: []GarudaSegment{
			{
				FlightNumber:    "GA315",
				Departure:       GarudaLocation{Airport: "CGK", Time: "2025-12-15T14:00:00+07:00"},
				Arrival:         GarudaLocation{Airport: "SUB", Time: "2025-12-15T15:30:00+07:00"},
				DurationMinutes: 90,
			},
			{
				FlightNumber:    "GA332",
				Departure:       GarudaLocation{Airport: "SUB", Time: "2025-12-15T17:15:00+07:00"},
				Arrival:         GarudaLocation{Airport: "DPS", Time: "2025-12-15T18:45:00+08:00"},
				DurationMinutes: 90, LayoverMinutes: 105,
			},
		},
		AvailableSeats: 22, FareClass: "economy",
		Baggage: GarudaBaggage{CarryOn: 1, Checked: 2},
	},
	{
		// Added a Business Class flight to verify filtering
		FlightID: "GA999", Airline: "Garuda Indonesia", AirlineCode: "GA",
		Departure:       GarudaLocation{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T07:00:00+07:00", Terminal: "3"},
		Arrival:         GarudaLocation{Airport: "DPS", City: "Denpasar", Time: "2025-12-15T09:50:00+08:00", Terminal: "I"},
		DurationMinutes: 110, Stops: 0, Aircraft: "Boeing 777",
		Price:          GarudaPrice{Amount: 3500000, Currency: "IDR"},
		AvailableSeats: 8, FareClass: "business",
		Baggage: GarudaBaggage{CarryOn: 2, Checked: 3}, Amenities: []string{"wifi", "meal", "entertainment", "lounge"},
	},
}

func GarudaSearchHandler(w http.ResponseWriter, r *http.Request) {
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

	filtered := make([]GarudaFlight, 0)

	for _, f := range garudaFlights {
		if !strings.EqualFold(f.Departure.Airport, req.Origin) {
			continue
		}

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

		if !strings.EqualFold(f.FareClass, req.CabinClass) {
			continue
		}

		if f.AvailableSeats < req.Passengers {
			continue
		}

		t, err := time.Parse(time.RFC3339, f.Departure.Time)
		if err == nil {
			dbDate := t.Format("2006-01-02")
			if dbDate != req.DepartureDate {
				continue
			}
		}

		filtered = append(filtered, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GarudaResponse{Status: "success", Flights: filtered})
}
