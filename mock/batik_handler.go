package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type BatikResponse struct {
	Results []BatikFlight `json:"results"`
}

type BatikFlight struct {
	FlightNumber      string    `json:"flightNumber"`
	AirlineName       string    `json:"airlineName"`
	AirlineIATA       string    `json:"airlineIATA"`
	Origin            string    `json:"origin"`
	Destination       string    `json:"destination"`
	DepartureDateTime string    `json:"departureDateTime"` // Format: 2025-12-15T07:15:00+0700
	ArrivalDateTime   string    `json:"arrivalDateTime"`
	TravelTime        string    `json:"travelTime"`
	NumberOfStops     uint32    `json:"numberOfStops"`
	Fare              BatikFare `json:"fare"`
	SeatsAvailable    uint32    `json:"seatsAvailable"`
	AircraftModel     string    `json:"aircraftModel"`
	BaggageInfo       string    `json:"baggageInfo"`
	OnboardServices   []string  `json:"onboardServices"`
}

type BatikFare struct {
	BasePrice    uint64 `json:"basePrice"`
	Taxes        uint64 `json:"taxes"`
	TotalPrice   uint64 `json:"totalPrice"`
	CurrencyCode string `json:"currencyCode"`
	Class        string `json:"class"` // "Y", "C", etc.
}

var batikFlights = []BatikFlight{
	{
		FlightNumber: "ID6514", AirlineName: "Batik Air", AirlineIATA: "ID",
		Origin: "CGK", Destination: "DPS",
		DepartureDateTime: "2025-12-15T07:15:00+0700", ArrivalDateTime: "2025-12-15T10:00:00+0800",
		TravelTime: "1h 45m", NumberOfStops: 0,
		Fare:           BatikFare{TotalPrice: 1100000, CurrencyCode: "IDR", Class: "Y"},
		SeatsAvailable: 32, AircraftModel: "Airbus A320", BaggageInfo: "7kg cabin, 20kg checked",
	},
	{
		FlightNumber: "ID6520", AirlineName: "Batik Air", AirlineIATA: "ID",
		Origin: "CGK", Destination: "DPS",
		DepartureDateTime: "2025-12-15T13:30:00+0700", ArrivalDateTime: "2025-12-15T16:20:00+0800",
		TravelTime: "1h 50m", NumberOfStops: 0,
		Fare:           BatikFare{TotalPrice: 1180000, CurrencyCode: "IDR", Class: "Y"},
		SeatsAvailable: 18, AircraftModel: "Boeing 737-800", BaggageInfo: "7kg cabin, 20kg checked",
	},
	{
		FlightNumber: "ID7042", AirlineName: "Batik Air", AirlineIATA: "ID",
		Origin: "CGK", Destination: "DPS",
		DepartureDateTime: "2025-12-15T18:45:00+0700", ArrivalDateTime: "2025-12-15T23:50:00+0800",
		TravelTime: "3h 5m", NumberOfStops: 1,
		Fare:           BatikFare{TotalPrice: 950000, CurrencyCode: "IDR", Class: "Y"},
		SeatsAvailable: 41, AircraftModel: "Airbus A320", BaggageInfo: "7kg cabin, 20kg checked",
	},
	// Adding a Business Class option to test filtering
	{
		FlightNumber: "ID9999", AirlineName: "Batik Air", AirlineIATA: "ID",
		Origin: "CGK", Destination: "DPS",
		DepartureDateTime: "2025-12-15T09:00:00+0700", ArrivalDateTime: "2025-12-15T11:45:00+0800",
		TravelTime: "1h 45m", NumberOfStops: 0,
		Fare:           BatikFare{TotalPrice: 2500000, CurrencyCode: "IDR", Class: "C"},
		SeatsAvailable: 8, AircraftModel: "Boeing 737-800", BaggageInfo: "10kg cabin, 30kg checked",
	},
}

func BatikSearchHandler(w http.ResponseWriter, r *http.Request) {
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

	filtered := make([]BatikFlight, 0)

	const batikLayout = "2006-01-02T15:04:05-0700"

	for _, f := range batikFlights {
		if !strings.EqualFold(f.Origin, req.Origin) || !strings.EqualFold(f.Destination, req.Destination) {
			continue
		}

		// Request sends "Economy", Data has "Y". Request sends "Business", Data has "C".
		reqClass := strings.ToLower(req.CabinClass)
		dataClass := f.Fare.Class

		isMatch := false
		if reqClass == "economy" && dataClass == "Y" {
			isMatch = true
		}
		if reqClass == "business" && (dataClass == "C" || dataClass == "J") {
			isMatch = true
		}

		if !isMatch {
			continue
		}

		if f.SeatsAvailable < req.Passengers {
			continue
		}

		t, err := time.Parse(batikLayout, f.DepartureDateTime)
		if err == nil {
			dbDate := t.Format("2006-01-02")
			if dbDate != req.DepartureDate {
				continue
			}
		}

		filtered = append(filtered, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatikResponse{Results: filtered})
}
