package flightclient

import (
	"fmt"
	"math"
	"travel/internal/flight"
	"travel/pkg/logger"
)

type FlightManager struct {
	airAsiaClient *AirAsiaClient
	logger        logger.Client
}

func NewFlightClient(airAsiaClient *AirAsiaClient, logger logger.Client) *FlightManager {
	return &FlightManager{
		airAsiaClient: airAsiaClient,
		logger:        logger,
	}
}

func (f *FlightManager) GetFlights() (*flight.FlightSearchResponse, error) {
	airAsiaResp, err := f.airAsiaClient.GetFlights()
	if err != nil {
		return nil, err
	}

	domainFlights := make([]flight.Flight, 0, len(airAsiaResp.Flights))

	for _, aaFlight := range airAsiaResp.Flights {
		totalMinutes := uint32(math.Round(aaFlight.DurationHours * 60))
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		// Calculate Stops
		stopCount := uint32(0)
		if !aaFlight.DirectFlight {
			// If not direct, count the stops array.
			// If array is empty but DirectFlight is false, default to 1
			stopCount = uint32(len(aaFlight.Stops))
			if stopCount == 0 {
				stopCount = 1
			}
		}

		domainFlight := flight.Flight{
			ID:       aaFlight.FlightCode,
			Provider: "AirAsia",
			Airline: flight.Airline{
				Name: aaFlight.Airline,
				Code: aaFlight.FlightCode[0:2], // Assuming first 2 chars are IATA code
			},
			FlightNumber: aaFlight.FlightCode,
			Departure: flight.LocationTime{
				Airport: aaFlight.FromAirport,
				// City:      aaFlight.FromAirport,
				Datetime:  aaFlight.DepartTime,
				Timestamp: aaFlight.DepartTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport: aaFlight.ToAirport,
				// City:      aaFlight.ToAirport,
				Datetime:  aaFlight.ArriveTime,
				Timestamp: aaFlight.ArriveTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: stopCount,
			Price: flight.Price{
				Amount:   aaFlight.PriceIDR,
				Currency: "IDR",
			},
			AvailableSeats: aaFlight.Seats,
			CabinClass:     aaFlight.CabinClass,
			// Aircraft:       "",
			Amenities: []string{},
			Baggage: flight.Baggage{
				Checked: aaFlight.BaggageNote, // Mapping general note to checked
				// CarryOn: "7kg",
			},
		}

		domainFlights = append(domainFlights, domainFlight)
	}

	return &flight.FlightSearchResponse{
		Flights: domainFlights,
		Metadata: flight.Metadata{
			TotalResults:     uint32(len(domainFlights)),
			ProvidersQueried: 1,
			// Other metadata (SearchTimeMs, etc.) would usually be calculated
			// by the Service layer calling this manager.
		},
		// SearchCriteria is left empty here because this function signature
		// didn't receive the original search parameters.
	}, nil
}
