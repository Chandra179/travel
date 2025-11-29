package flightclient

import (
	"fmt"
	"math"
	"strings"
	"time"
	"travel/internal/flight"
	"travel/pkg/logger"
)

type FlightManager struct {
	airAsiaClient  *AirAsiaClient
	batikAirClient *BatikAirClient
	garudaClient   *GarudaClient
	lionAirClient  *LionAirClient
	logger         logger.Client
}

func NewFlightClient(airAsiaClient *AirAsiaClient, batikAirClient *BatikAirClient,
	garudaClient *GarudaClient, lionAirClient *LionAirClient, logger logger.Client) *FlightManager {
	return &FlightManager{
		airAsiaClient:  airAsiaClient,
		batikAirClient: batikAirClient,
		garudaClient:   garudaClient,
		lionAirClient:  lionAirClient,
		logger:         logger,
	}
}

func (f *FlightManager) SearchFlights(req flight.SearchRequest) (*flight.FlightSearchResponse, error) {
	airAsiaResp, errAA := f.airAsiaClient.SearchFlights(req)
	if errAA != nil {
		f.logger.Error("failed to fetch airasia", logger.Field{Key: "err", Value: errAA})
		return nil, errAA
	}

	batikResp, errBatik := f.batikAirClient.SearchFlights(req)
	if errBatik != nil {
		f.logger.Error("failed to fetch batik", logger.Field{Key: "err", Value: errBatik})
		return nil, errBatik
	}

	garudaResp, errGaruda := f.garudaClient.SearchFlights(req)
	if errGaruda != nil {
		f.logger.Error("failed to fetch garuda", logger.Field{Key: "err", Value: errGaruda})
		return nil, errGaruda
	}

	lionAirResp, errLionAir := f.lionAirClient.SearchFlights(req)
	if errLionAir != nil {
		f.logger.Error("failed to fetch lion air", logger.Field{Key: "err", Value: errLionAir})
		return nil, errLionAir
	}

	flightsAA := f.mapAirAsiaFlights(airAsiaResp)
	flightsBatik := f.mapBatikFlights(batikResp)
	flightsGaruda := f.mapGarudaFlights(garudaResp)
	flightsLionAir := f.mapLionAirFlights(lionAirResp)

	allFlights := append(flightsAA, flightsBatik...)
	allFlights = append(allFlights, flightsGaruda...)
	allFlights = append(allFlights, flightsLionAir...)

	return &flight.FlightSearchResponse{
		Flights: allFlights,
		Metadata: flight.Metadata{
			TotalResults:       uint32(len(allFlights)),
			ProvidersQueried:   4,
			ProvidersSucceeded: 4,
		},
	}, nil
}

func (f *FlightManager) mapLionAirFlights(resp *lionAirFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Data.AvailableFlights))
	const timeLayout = "2006-01-02T15:04:05"

	for _, lFlight := range resp.Data.AvailableFlights {
		totalMinutes := lFlight.FlightTime
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		locDep, err := time.LoadLocation(lFlight.Schedule.DepartureTimezone)
		if err != nil {
			locDep = time.UTC
		}
		depTime, _ := time.ParseInLocation(timeLayout, lFlight.Schedule.Departure, locDep)

		locArr, err := time.LoadLocation(lFlight.Schedule.ArrivalTimezone)
		if err != nil {
			locArr = time.UTC
		}
		arrTime, _ := time.ParseInLocation(timeLayout, lFlight.Schedule.Arrival, locArr)

		stopCount := lFlight.StopCount
		if !lFlight.IsDirect && stopCount == 0 && len(lFlight.Layovers) > 0 {
			stopCount = uint32(len(lFlight.Layovers))
		}

		amenities := make([]string, 0)
		if lFlight.Services.WifiAvailable {
			amenities = append(amenities, "Wi-Fi")
		}
		if lFlight.Services.MealsIncluded {
			amenities = append(amenities, "Meal")
		}

		domainFlight := flight.Flight{
			ID:       lFlight.ID,
			Provider: "Lion Air",
			Airline: flight.Airline{
				Name: lFlight.Carrier.Name,
				Code: lFlight.Carrier.IATA,
			},
			FlightNumber: lFlight.ID,
			Departure: flight.LocationTime{
				Airport:   lFlight.Route.From.Code,
				City:      lFlight.Route.From.City,
				Datetime:  depTime,
				Timestamp: depTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   lFlight.Route.To.Code,
				City:      lFlight.Route.To.City,
				Datetime:  arrTime,
				Timestamp: arrTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: stopCount,
			Price: flight.Price{
				Amount:   lFlight.Pricing.Total,
				Currency: lFlight.Pricing.Currency,
			},
			AvailableSeats: lFlight.SeatsLeft,
			CabinClass:     lFlight.Pricing.FareType, // e.g., "ECONOMY"
			Aircraft:       lFlight.PlaneType,
			Amenities:      amenities,
			Baggage: flight.Baggage{
				CarryOn: lFlight.Services.BaggageAllowance.Cabin,
				Checked: lFlight.Services.BaggageAllowance.Hold,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}

func (f *FlightManager) mapGarudaFlights(resp *garudaFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Flights))

	for _, gFlight := range resp.Flights {
		hours := gFlight.DurationMinutes / 60
		minutes := gFlight.DurationMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		depTime, _ := time.Parse(time.RFC3339, gFlight.Departure.Time)

		// If segments exist (like GA315), the final arrival is in the last segment
		finalArrival := gFlight.Arrival
		if len(gFlight.Segments) > 0 {
			lastSegment := gFlight.Segments[len(gFlight.Segments)-1]
			finalArrival = lastSegment.Arrival
		}

		arrTime, _ := time.Parse(time.RFC3339, finalArrival.Time)
		baggageNote := fmt.Sprintf("Cabin: %d, Checked: %d", gFlight.Baggage.CarryOn, gFlight.Baggage.Checked)

		domainFlight := flight.Flight{
			ID:       gFlight.FlightID,
			Provider: "Garuda Indonesia",
			Airline: flight.Airline{
				Name: gFlight.Airline,
				Code: gFlight.AirlineCode,
			},
			FlightNumber: gFlight.FlightID,
			Departure: flight.LocationTime{
				Airport:   gFlight.Departure.Airport,
				Datetime:  depTime,
				Timestamp: depTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   finalArrival.Airport,
				Datetime:  arrTime,
				Timestamp: arrTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: gFlight.DurationMinutes,
				Formatted:    formattedDuration,
			},
			Stops: gFlight.Stops,
			Price: flight.Price{
				Amount:   gFlight.Price.Amount,
				Currency: gFlight.Price.Currency,
			},
			AvailableSeats: gFlight.AvailableSeats,
			CabinClass:     gFlight.FareClass,
			Aircraft:       gFlight.Aircraft,
			Amenities:      gFlight.Amenities,
			Baggage: flight.Baggage{
				Checked: baggageNote,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}

func (f *FlightManager) mapAirAsiaFlights(resp *airAsiaFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Flights))

	for _, aaFlight := range resp.Flights {
		totalMinutes := uint32(math.Round(aaFlight.DurationHours * 60))
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

		stopCount := uint32(0)
		if !aaFlight.DirectFlight {
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
				Code: aaFlight.FlightCode[0:2],
			},
			FlightNumber: aaFlight.FlightCode,
			Departure: flight.LocationTime{
				Airport:   aaFlight.FromAirport,
				Datetime:  aaFlight.DepartTime,
				Timestamp: aaFlight.DepartTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   aaFlight.ToAirport,
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
			Amenities:      []string{},
			Baggage: flight.Baggage{
				Checked: aaFlight.BaggageNote,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}

func (f *FlightManager) mapBatikFlights(resp *batikAirFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Results))
	const batikTimeLayout = "2006-01-02T15:04:05-0700"

	for _, btFlight := range resp.Results {
		depTime, _ := time.Parse(batikTimeLayout, btFlight.DepartureDateTime)
		arrTime, _ := time.Parse(batikTimeLayout, btFlight.ArrivalDateTime)

		totalMinutes, formattedDuration := f.parseBatikDuration(btFlight.TravelTime)

		domainFlight := flight.Flight{
			ID:       btFlight.FlightNumber,
			Provider: "Batik Air",
			Airline: flight.Airline{
				Name: btFlight.AirlineName,
				Code: btFlight.AirlineIATA,
			},
			FlightNumber: btFlight.FlightNumber,
			Departure: flight.LocationTime{
				Airport:   btFlight.Origin,
				Datetime:  depTime,
				Timestamp: depTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   btFlight.Destination,
				Datetime:  arrTime,
				Timestamp: arrTime.Unix(),
			},
			Duration: flight.Duration{
				TotalMinutes: totalMinutes,
				Formatted:    formattedDuration,
			},
			Stops: btFlight.NumberOfStops,
			Price: flight.Price{
				Amount:   btFlight.Fare.TotalPrice,
				Currency: btFlight.Fare.CurrencyCode,
			},
			AvailableSeats: btFlight.SeatsAvailable,
			CabinClass:     btFlight.Fare.Class,
			Aircraft:       btFlight.AircraftModel,
			Amenities:      btFlight.OnboardServices,
			Baggage: flight.Baggage{
				Checked: btFlight.BaggageInfo,
			},
		}
		mapped = append(mapped, domainFlight)
	}
	return mapped
}

func (f *FlightManager) parseBatikDuration(input string) (uint32, string) {
	cleanInput := strings.ReplaceAll(input, " ", "") // "1h 45m" -> "1h45m"
	d, err := time.ParseDuration(cleanInput)
	if err != nil {
		return 0, input
	}

	minutes := uint32(d.Minutes())
	h := minutes / 60
	m := minutes % 60
	return minutes, fmt.Sprintf("%dh %dm", h, m)
}
