# Online Travel Agency

Flight Search & Aggregation System

## Project Structure

```
├── cmd/                               # Runnable applications / entrypoints
│   ├── travel/                        
│   │   └── main.go                    # Travel app entry point
├── pkg/                               # Reusable library packages
│   ├── cache/                         # Cache interfaces, Redis helpers, wrappers
│   ├── logger/                        # Zerolog wrapper & helpers
├── api/
├── cfg/                               # Centralized config files (YAML, JSON, HCL)
│   ├── config.go

```

## Features

1. architecture
we use 2 endpoint flight search and flight filter (combine filter and sort into 1 functions) because it have similar implementation. for search result we cache the response to redis, for cache missed we call the api. 


2. endpoint search flights:
caching resposne from api call

3. endpoint filter flights:
including search request, why? because if the user on page after the search redirect to the response page
and because ttl can expired like 5 minutes and the user try to do filter the flights, the data will be gone because of 
the data in redis expired or. we dont want to throw back user to do search page again, so we need 
the search request in case cache expired we fallback to call the api 

im combine filter and sort to one endpoint because its same behavior and to reduce code redundancy
and im making it optional so its flexible

Failure handling ideally we expect happy flow from other provider, for example return error code, clear error message if it behaves correctly but we still need to prevent
unhappy flow like only error message, no error code, so im planning to use error code so the client can handle api error flexible.

handle partial error, 3 of 4 api is error, we still cache the 1 response, 
but we still need to try retry in background and update the cache, otherwise user will wait until the ttl expired
the error is handled partially so its not stoping the entire request if any of the api call failed

i choose mapping response manually for clarity and better control, if there is an error it obvious where its coming from

4. cache data
we need to simulate case for cache data. 
for example ttl: 5 minutes, high traffic user do search flights like how many people, lets say there
are like 100 events accros city in date 29, so 100 keys (approx size in bytes)

second case we see the bigger picture in a month how many events can be occured in one month
is it holiday, tanggal merah, etc.., big event (the traffic would be high)

if we tied it to key (origin, destination, derpature date).



## Running
prerequisites: docker installed
for quick setup run `make dev-setup` it run the app on port `8080` localhost and the mock server at `8081`
for custom port change the port at `.env.example`  for `MOCK_PORT` and `APP_PORT`

## Testing
test files at search.http

## Tech stack
golang, swagger, redis