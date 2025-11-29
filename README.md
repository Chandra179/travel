# Online Travel Agency

Flight Search & Aggregation System

## Project Structure

```
├── cmd/                               # Runnable applications / entrypoints
│   ├── travel/                        
│   │   └── main.go
├── pkg/                               # Reusable library packages
│   ├── cache/                         # Cache interfaces, Redis helpers, wrappers
│   ├── logger/                        # Zerolog wrapper & helpers
├── api/
├── cfg/                               # Centralized config files (YAML, JSON, HCL)
│   ├── config.go

```

## Features
fligt search
flight filter

architecture approach

cache search flight response from external call by combination of (origin + destination + derpature)
on cache expired (5 minutes (configurable)) fallback to API call search flight


1. endpoint search flights:
request: 
response:

2. endpoint filter flights:
including search request, why? because if the user on page after the search and because ttl expired 5 minutes 
and the user try to do filter the data will be gone, we dont want to throw back user to do search again, so we need 
the request if cache expired we fallback to call the api

im combine filter and sort to one endpoint because its same behavior and to reduce code redundancy
and im making it optional 

request: 
response:

Failure handling ideally we expect happy flow from other provider, for example return error code, clear error message if it behaves correctly but we still need to prevent
unhappy flow like only error message, no error code, so im planning to use error code so the client can handle api error flexible.

retry if failed, for partial error, 3 of 4 is error, we cache the 1 response, 
but we still need to try retry in background and update teh cache, otherwise user will wait until the ttl expired

robust error handling, handle error partially not stoping the entire request if any of the api call failed

flexible datetime handling

i choose mapping response manually for clarity and better control, if there is an error it obvious where its coming from

## Setup

## Improvement
using linter, import & field alignment, 