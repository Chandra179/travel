package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
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

func BatikSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Read JSON file
	data, err := os.ReadFile("mock/files/batik_air_search_response.json")
	if err != nil {
		http.Error(w, "Failed to read flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal to struct
	var fileResponse struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Results []BatikFlight `json:"results"`
	}
	if err := json.Unmarshal(data, &fileResponse); err != nil {
		http.Error(w, "Failed to parse flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filtering
	filtered := make([]BatikFlight, 0)
	const batikLayout = "2006-01-02T15:04:05-0700"

	for _, f := range fileResponse.Results {
		if req.Origin != "" && !strings.EqualFold(f.Origin, req.Origin) {
			continue
		}

		if req.Destination != "" && !strings.EqualFold(f.Destination, req.Destination) {
			continue
		}

		// Request sends "Economy", Data has "Y". Request sends "Business", Data has "C".
		if req.CabinClass != "" {
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
		}

		if req.Passengers > 0 && f.SeatsAvailable < req.Passengers {
			continue
		}

		if req.DepartureDate != "" {
			t, err := time.Parse(batikLayout, f.DepartureDateTime)
			if err == nil {
				dbDate := t.Format("2006-01-02")
				if dbDate != req.DepartureDate {
					continue
				}
			}
		}

		filtered = append(filtered, f)
	}

	// Simulate random delay (200-400ms)
	delay := 200 + rand.Intn(201) // 200 to 400ms
	time.Sleep(time.Duration(delay) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BatikResponse{Results: filtered})
}
