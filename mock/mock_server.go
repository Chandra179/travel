package main

import (
	"fmt"
	"log"
	"net/http"
)

type SearchRequest struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"` // Format: YYYY-MM-DD
	ReturnDate    string `json:"return_date"`    // Format: YYYY-MM-DD
	Passengers    uint32 `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

func main() {
	http.HandleFunc("/airasia/v1/flights/search", AirAsiaSearchHandler)
	http.HandleFunc("/batikair/v1/flights/search", BatikSearchHandler)
	http.HandleFunc("/garuda/v1/flights/search", GarudaSearchHandler)
	http.HandleFunc("/lionair/v1/flights/search", LionAirSearchHandler)

	fmt.Println("Go Mock Server running on port 8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
