package main

import (
	"encoding/json"
	"net/http"
	"os"
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

func AirAsiaSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Read JSON file
	data, err := os.ReadFile("mock/files/airasia_search_response.json")
	if err != nil {
		http.Error(w, "Failed to read flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal to struct
	var fileResponse struct {
		Status  string          `json:"status"`
		Flights []AirAsiaFlight `json:"flights"`
	}
	if err := json.Unmarshal(data, &fileResponse); err != nil {
		http.Error(w, "Failed to parse flight data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filtering
	filteredFlights := make([]AirAsiaFlight, 0)

	for _, f := range fileResponse.Flights {
		if req.Origin != "" && !strings.EqualFold(f.FromAirport, req.Origin) {
			continue
		}

		if req.Destination != "" && !strings.EqualFold(f.ToAirport, req.Destination) {
			continue
		}

		if req.CabinClass != "" && !strings.EqualFold(f.CabinClass, req.CabinClass) {
			continue
		}

		if req.Passengers > 0 && f.Seats < req.Passengers {
			continue
		}

		if req.DepartureDate != "" {
			t, err := time.Parse(time.RFC3339, f.DepartTime)
			if err == nil {
				dbDate := t.Format("2006-01-02")
				if dbDate != req.DepartureDate {
					continue
				}
			}
		}

		filteredFlights = append(filteredFlights, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AirAsiaResponse{Flights: filteredFlights})
}
