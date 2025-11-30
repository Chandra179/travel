#!/bin/bash

BASE_URL="http://localhost:8080/v1/flights"

echo "============================================"
echo "Flight Search"
echo "============================================"
curl -X POST "$BASE_URL/search" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy"
  }'
echo
echo

echo "============================================"
echo "Filter: Direct Flights Only"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": { "max_stops": 0 }
  }'
echo
echo

echo "============================================"
echo "Filter: Morning Departure"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": {
        "departure_time": { "from": "04:00", "to": "12:00" }
    }
  }'
echo
echo

echo "============================================"
echo "Filter: Max Duration (Short flights)"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": { "max_duration": 120 }
  }'
echo
echo

echo "============================================"
echo "Filter: Multiple Criteria"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": {
      "price_range": { "low": 400000, "high": 650000 },
      "max_stops": 0,
      "departure_time": { "from": "04:00", "to": "12:00" }
    }
  }'
echo
echo

echo "============================================"
echo "Filter: Airlines"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": { "airlines": ["AirAsia", "QZ"] }
  }'
echo
echo

echo "============================================"
echo "Filter: Evening Flights"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": {
      "departure_time": { "from": "18:00", "to": "23:59" }
    }
  }'
echo
echo

echo "============================================"
echo "Sort: Price ASC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "price", "order": "asc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Price DESC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "price", "order": "desc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Duration ASC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "duration", "order": "asc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Duration DESC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "duration", "order": "desc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Departure Time ASC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "departure_time", "order": "asc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Departure Time DESC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "departure_time", "order": "desc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Arrival Time ASC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "arrival_time", "order": "asc" }
  }'
echo
echo

echo "============================================"
echo "Sort: Best Value DESC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "sort": { "by": "best_value", "order": "desc" }
  }'
echo
echo

echo "============================================"
echo "Filter + Sort: Direct Flights, Price ASC"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": { "max_stops": 0 },
    "sort": { "by": "price", "order": "asc" }
  }'
echo
echo

echo "============================================"
echo "Filter + Sort: Morning Flights by Best Value"
echo "============================================"
curl -X POST "$BASE_URL/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": {
      "departure_time": { "from": "04:00", "to": "12:00" }
    },
    "sort": { "by": "best_value", "order": "asc" }
  }'
echo
echo
