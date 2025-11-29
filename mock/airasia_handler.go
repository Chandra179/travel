package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AirAsiaResponse struct {
	Flights []AirAsiaFlight `json:"flights"`
}

type AirAsiaFlight struct {
	FlightCode    string  `json:"flight_code"`
	Airline       string  `json:"airline"`
	FromAirport   string  `json:"from_airport"`
	ToAirport     string  `json:"to_airport"`
	DepartTime    string  `json:"depart_time"` // ISO 8601
	ArriveTime    string  `json:"arrive_time"`
	DurationHours float64 `json:"duration_hours"`
	DirectFlight  bool    `json:"direct_flight"`
	PriceIDR      int     `json:"price_idr"`
	Seats         uint32  `json:"seats"`
	CabinClass    string  `json:"cabin_class"`
	BaggageNote   string  `json:"baggage_note"`
}

var airAsiaFlights = []AirAsiaFlight{
	{
		FlightCode: "QZ520", Airline: "AirAsia", FromAirport: "CGK", ToAirport: "DPS",
		DepartTime: "2025-12-15T04:45:00+07:00", ArriveTime: "2025-12-15T07:25:00+08:00",
		DurationHours: 1.67, DirectFlight: true, PriceIDR: 650000, Seats: 67,
		CabinClass: "economy", BaggageNote: "Cabin baggage only",
	},
	{
		FlightCode: "QZ524", Airline: "AirAsia", FromAirport: "CGK", ToAirport: "DPS",
		DepartTime: "2025-12-15T10:00:00+07:00", ArriveTime: "2025-12-15T12:45:00+08:00",
		DurationHours: 1.75, DirectFlight: true, PriceIDR: 720000, Seats: 54,
		CabinClass: "economy", BaggageNote: "Cabin baggage only",
	},
	{
		FlightCode: "QZ532", Airline: "AirAsia", FromAirport: "CGK", ToAirport: "DPS",
		DepartTime: "2025-12-15T19:30:00+07:00", ArriveTime: "2025-12-15T22:10:00+08:00",
		DurationHours: 1.67, DirectFlight: true, PriceIDR: 595000, Seats: 72,
		CabinClass: "economy", BaggageNote: "Cabin baggage only",
	},
	{
		FlightCode: "QZ7250", Airline: "AirAsia", FromAirport: "CGK", ToAirport: "DPS",
		DepartTime: "2025-12-15T15:15:00+07:00", ArriveTime: "2025-12-15T20:35:00+08:00",
		DurationHours: 4.33, DirectFlight: false, PriceIDR: 485000, Seats: 88,
		CabinClass: "economy", BaggageNote: "Cabin baggage only",
	},
	{
		FlightCode: "QZ999", Airline: "AirAsia", FromAirport: "CGK", ToAirport: "DPS",
		DepartTime: "2025-12-15T12:00:00+07:00", ArriveTime: "2025-12-15T14:45:00+08:00",
		DurationHours: 1.75, DirectFlight: true, PriceIDR: 1500000, Seats: 12,
		CabinClass: "business", BaggageNote: "Checked included",
	},
}

func AirAsiaSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Origin == "" || req.Destination == "" || req.DepartureDate == "" ||
		req.ReturnDate == "" || req.Passengers == 0 || req.CabinClass == "" {
		http.Error(w, "All fields (origin, destination, departure_date, return_date, passengers, cabin_class) are mandatory", http.StatusBadRequest)
		return
	}

	filteredFlights := make([]AirAsiaFlight, 0)

	for _, f := range airAsiaFlights {
		if !strings.EqualFold(f.FromAirport, req.Origin) || !strings.EqualFold(f.ToAirport, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.CabinClass, req.CabinClass) {
			continue
		}

		if f.Seats < req.Passengers {
			continue
		}

		t, err := time.Parse(time.RFC3339, f.DepartTime)
		if err == nil {
			dbDate := t.Format("2006-01-02") // Extract YYYY-MM-DD
			if dbDate != req.DepartureDate {
				continue
			}
		}

		filteredFlights = append(filteredFlights, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AirAsiaResponse{Flights: filteredFlights})
}
