package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	port := "8081"

	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	http.HandleFunc("/health", HealthCheckHandler)

	http.HandleFunc("/airasia/v1/flights/search", AirAsiaSearchHandler)
	http.HandleFunc("/batikair/v1/flights/search", BatikSearchHandler)
	http.HandleFunc("/garuda/v1/flights/search", GarudaSearchHandler)
	http.HandleFunc("/lionair/v1/flights/search", LionAirSearchHandler)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Go Mock Server running on port %s...\n", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
