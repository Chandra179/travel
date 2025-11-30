package flight

import (
	"strings"
	"time"
)

// filterContext holds parsed data so we don't re-parse inside the loop
type filterContext struct {
	opts    FilterOptions
	depFrom int64
	depTo   int64
	arrFrom int64
	arrTo   int64
}

func newFilterContext(opts FilterOptions) *filterContext {
	fc := &filterContext{opts: opts}

	if opts.DepartureTime != nil {
		fc.depFrom = parseTimeToSeconds(opts.DepartureTime.From)
		fc.depTo = parseTimeToSeconds(opts.DepartureTime.To)
	}
	if opts.ArrivalTime != nil {
		fc.arrFrom = parseTimeToSeconds(opts.ArrivalTime.From)
		fc.arrTo = parseTimeToSeconds(opts.ArrivalTime.To)
	}
	return fc
}

func (s *Service) applyFilters(flights []Flight, opts FilterOptions) []Flight {
	fc := newFilterContext(opts)

	// Pre-allocate assuming worst case (no flights filtered) to avoid resizing
	filtered := make([]Flight, 0, len(flights))

	for _, f := range flights {
		if fc.matches(f) {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// matches returns true only if ALL active filters pass
func (fc *filterContext) matches(f Flight) bool {
	// Price
	if fc.opts.PriceRange != nil {
		if f.Price.Amount < fc.opts.PriceRange.Low || f.Price.Amount > fc.opts.PriceRange.High {
			return false
		}
	}

	// Stops
	if fc.opts.MaxStops != nil {
		if f.Stops > *fc.opts.MaxStops {
			return false
		}
	}

	// Duration
	if fc.opts.MaxDuration != nil {
		if f.Duration.TotalMinutes > *fc.opts.MaxDuration {
			return false
		}
	}

	// Time Windows (Using pre-calculated seconds)
	if fc.opts.DepartureTime != nil {
		depSec := getSecondsFromMidnight(f.Departure.Datetime)
		if depSec < fc.depFrom || depSec > fc.depTo {
			return false
		}
	}

	if fc.opts.ArrivalTime != nil {
		arrSec := getSecondsFromMidnight(f.Arrival.Datetime)
		if arrSec < fc.arrFrom || arrSec > fc.arrTo {
			return false
		}
	}

	// Airlines (String comparison is heaviest, do last)
	if len(fc.opts.Airlines) > 0 {
		matched := false
		for _, airline := range fc.opts.Airlines {
			if strings.EqualFold(f.Airline.Code, airline) || strings.EqualFold(f.Airline.Name, airline) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// Helper functions for time conversion
func parseTimeToSeconds(timeStr string) int64 {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0
	}
	return int64(t.Hour()*3600 + t.Minute()*60)
}

func getSecondsFromMidnight(dt time.Time) int64 {
	return int64(dt.Hour()*3600 + dt.Minute()*60 + dt.Second())
}
