# Travel Flight API

A Go-based flight aggregation service that searches multiple airline providers concurrently and provides filtering/sorting capabilities.

## Table of Contents

- [Overview](#overview)
- [Running the Application](#running-the-application)
- [Architecture](#architecture)
- [Key Design Decisions](#key-design-decisions)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)

## Running
Prerequisites : Need Docker to be installed

```bash
git clone <repository-url>
cd travel

# automates checking for docker installation, copying .env.example, build docker image, start server, detailed on (dev-setup.sh)
make dev-setup
```

## Overview

This service aggregates flight search results from multiple airline providers (AirAsia, Batik Air, Garuda Indonesia, Lion Air) and provides:

- **Concurrent API calls** to multiple providers for fast results
- **Redis caching** to reduce API calls and improve response times
- **Flexible filtering and sorting** on cached results
- **Partial result handling** when some providers fail
- **Swagger/OpenAPI documentation** for easy API exploration

## Architecture
![Architecture](arch.png)

## Key Design Decisions

### 1. Concurrent Provider Queries

**Decision**: Query all 4 airline providers simultaneously using goroutines.

**Reason**:
- Minimizes total search time (limited by slowest provider, not sum of all)
- Each provider has 5-second timeout to prevent one slow API from blocking others

**Trade-off**: Higher resource usage (4 concurrent HTTP connections per search)

### 2. Partial Results Strategy

**Decision**: Return results even if some providers fail (timeout/error).

**Reason**:
- Providers are independent companies; one failure shouldn't block the entire search
- Users still get value from available results
- Metadata includes `providers_failed` and `provider_errors` for transparency

**Example Response**:
```json
{
  "metadata": {
    "total_results": 45,
    "providers_queried": 4,
    "providers_succeeded": 3,
    "providers_failed": 1,
    "provider_errors": [
      {
        "provider": "Lion Air",
        "code": "TIMEOUT"
      }
    ]
  },
  "flights": [...]
}
```

**Trade-off**: Users may not see all available flights if providers fail.

### 3. Cache Key Strategy

**Decision**: Cache results based on `(origin, destination, departure_date, passengers, cabin_class)`.

**Reason**:
- Airline APIs return different results for different passenger counts and cabin classes
- Each unique search combination must have its own cache entry
- Ensures users always get accurate prices and availability

**Cache Key Format**:
```go
key := fmt.Sprintf("flight:%s:%s:%s:%d:%s",
    req.Origin,
    req.Destination,
    req.DepartureDate,
    req.Passengers,
    req.CabinClass,
)
hash := sha256.Sum256([]byte(key))
cacheKey := fmt.Sprintf("flight:search:%x", hash[:16])
```

**Trade-off**: Less cache reuse (more unique keys), but ensures data correctness.

### 4. TTL-Based Cache Expiration

**Decision**: Use simple TTL (Time-To-Live) with configurable duration (default: 5 minutes).

**Reason**:
- Flight prices and availability change rapidly
- Simple to implement and reason about
- Configurable via environment variable for different environments

**Trade-off**: Stale data possible within TTL window. No active invalidation mechanism.

### 5. Filter/Sort on Cached Data

**Decision**: Separate `/search` and `/filter` endpoints. Filter operates on cached results.

**User Flow**:
1. User searches flights → `/v1/flights/search`
   - Queries all providers concurrently
   - Caches raw results
   - Returns all flights
2. User applies filters → `/v1/flights/filter`
   - Checks cache for original search
   - If **cache hit**: Filters in-memory (instant response)
   - If **cache miss**: Falls back to `/search` logic

**Reason**:
- Instant filtering for users (no additional API calls)
- Supports multiple filter/sort operations on same search
- Assumes maximum ~1000 flights per search (in-memory filtering is fast)

**Trade-off**: 
- Cache miss forces slow re-search (bad UX if cache expires, so fallback to search call API)
- `FilterRequest` must include original `SearchRequest` parameters for fallback

### 6. Flexible Time Parsing

**Decision**: Handle multiple time formats from different providers in the client layer.

**Reason**:
- Each airline API uses different datetime formats:
  - AirAsia: RFC3339 (`2006-01-02T15:04:05Z07:00`)
  - Batik Air: `2006-01-02T15:04:05-0700`
  - Lion Air: `2006-01-02T15:04:05` (no timezone)
  - Garuda: RFC3339
- Centralizing parsing logic simplifies provider-specific clients

**Implementation**:
```go
type FlexibleTime struct {
    time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
    // Tries multiple formats in order
}
```

**Trade-off**: Slightly more complex unmarshaling, but better maintainability.
