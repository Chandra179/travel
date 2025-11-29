package flightclient

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
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

type providerResult struct {
	provider string
	flights  []flight.Flight
	err      error
}

func (f *FlightManager) SearchFlights(ctx context.Context, req flight.SearchRequest) (*flight.FlightSearchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resultChan := make(chan providerResult, 6)
	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		defer wg.Done()
		resp, err := f.airAsiaClient.SearchFlights(ctx, req)
		if err != nil {
			f.logger.Error("failed to fetch airasia", logger.Field{Key: "err", Value: err})
			resultChan <- providerResult{provider: "AirAsia", err: err}
			return
		}
		flights := f.mapAirAsiaFlights(resp)
		resultChan <- providerResult{provider: "AirAsia", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.batikAirClient.SearchFlights(ctx, req)
		if err != nil {
			f.logger.Error("failed to fetch batik", logger.Field{Key: "err", Value: err})
			resultChan <- providerResult{provider: "Batik Air", err: err}
			return
		}
		flights := f.mapBatikFlights(resp)
		resultChan <- providerResult{provider: "Batik Air", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.garudaClient.SearchFlights(ctx, req)
		if err != nil {
			f.logger.Error("failed to fetch garuda", logger.Field{Key: "err", Value: err})
			resultChan <- providerResult{provider: "Garuda Indonesia", err: err}
			return
		}
		flights := f.mapGarudaFlights(resp)
		resultChan <- providerResult{provider: "Garuda Indonesia", flights: flights}
	}()

	go func() {
		defer wg.Done()
		resp, err := f.lionAirClient.SearchFlights(ctx, req)
		if err != nil {
			f.logger.Error("failed to fetch lion air", logger.Field{Key: "err", Value: err})
			resultChan <- providerResult{provider: "Lion Air", err: err}
			return
		}
		flights := f.mapLionAirFlights(resp)
		resultChan <- providerResult{provider: "Lion Air", flights: flights}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allFlights []flight.Flight
	providersSucceeded := uint32(0)
	providersQueried := uint32(4)

	for result := range resultChan {
		if result.err == nil {
			allFlights = append(allFlights, result.flights...)
			providersSucceeded++
		}
	}

	return &flight.FlightSearchResponse{
		Flights: allFlights,
		Metadata: flight.Metadata{
			TotalResults:       uint32(len(allFlights)),
			ProvidersQueried:   providersQueried,
			ProvidersSucceeded: providersSucceeded,
		},
	}, nil
}

func (f *FlightManager) mapLionAirFlights(resp *LionAirFlightResponse) []flight.Flight {
	mapped := make([]flight.Flight, 0, len(resp.Data.AvailableFlights))

	for _, lFlight := range resp.Data.AvailableFlights {
		totalMinutes := lFlight.FlightTime
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		formattedDuration := fmt.Sprintf("%dh %dm", hours, minutes)

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
			Provider: lFlight.Carrier.Name,
			Airline: flight.Airline{
				Name: lFlight.Carrier.Name,
				Code: lFlight.Carrier.IATA,
			},
			FlightNumber: lFlight.ID,
			Departure: flight.LocationTime{
				Airport:   lFlight.Route.From.Code,
				City:      lFlight.Route.From.City,
				Datetime:  lFlight.Schedule.Departure,
				Timestamp: lFlight.Schedule.Departure.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   lFlight.Route.To.Code,
				City:      lFlight.Route.To.City,
				Datetime:  lFlight.Schedule.Arrival,
				Timestamp: lFlight.Schedule.Arrival.Unix(),
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
			CabinClass:     lFlight.Pricing.FareType,
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

		finalArrival := gFlight.Arrival
		if len(gFlight.Segments) > 0 {
			lastSegment := gFlight.Segments[len(gFlight.Segments)-1]
			finalArrival = lastSegment.Arrival
		}

		baggageCabin := fmt.Sprintf("Cabin: %d", gFlight.Baggage.CarryOn)
		baggageChecked := fmt.Sprintf("Checked: %d", gFlight.Baggage.Checked)

		domainFlight := flight.Flight{
			ID:       gFlight.FlightID,
			Provider: gFlight.Airline,
			Airline: flight.Airline{
				Name: gFlight.Airline,
				Code: gFlight.AirlineCode,
			},
			FlightNumber: gFlight.FlightID,
			Departure: flight.LocationTime{
				Airport:   gFlight.Departure.Airport,
				Datetime:  gFlight.Departure.Time,
				City:      gFlight.Departure.City,
				Timestamp: gFlight.Departure.Time.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   finalArrival.Airport,
				Datetime:  gFlight.Arrival.Time,
				City:      gFlight.Arrival.City,
				Timestamp: gFlight.Arrival.Time.Unix(),
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
				CarryOn: baggageCabin,
				Checked: baggageChecked,
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
				// City: "",
			},
			Arrival: flight.LocationTime{
				Airport:   aaFlight.ToAirport,
				Datetime:  aaFlight.ArriveTime,
				Timestamp: aaFlight.ArriveTime.Unix(),
				// City: "",
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
			// Amenities:      []string{},
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

	for _, btFlight := range resp.Results {

		totalMinutes, formattedDuration := f.parseBatikDuration(btFlight.TravelTime)

		domainFlight := flight.Flight{
			ID:       btFlight.FlightNumber,
			Provider: btFlight.AirlineName,
			Airline: flight.Airline{
				Name: btFlight.AirlineName,
				Code: btFlight.AirlineIATA,
			},
			FlightNumber: btFlight.FlightNumber,
			Departure: flight.LocationTime{
				Airport:   btFlight.Origin,
				Datetime:  btFlight.DepartureDateTime,
				Timestamp: btFlight.DepartureDateTime.Unix(),
			},
			Arrival: flight.LocationTime{
				Airport:   btFlight.Destination,
				Datetime:  btFlight.ArrivalDateTime,
				Timestamp: btFlight.ArrivalDateTime.Unix(),
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
	cleanInput := strings.ReplaceAll(input, " ", "")
	d, err := time.ParseDuration(cleanInput)
	if err != nil {
		return 0, input
	}

	minutes := uint32(d.Minutes())
	h := minutes / 60
	m := minutes % 60
	return minutes, fmt.Sprintf("%dh %dm", h, m)
}
