#!/bin/bash

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_info "Checking prerequisites..."
if ! command_exists go; then
    print_error "Go is not installed. Please install Go first."
    exit 1
fi

if ! command_exists docker; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Determine which docker compose command to use
DOCKER_COMPOSE_CMD="docker-compose"
if ! command_exists docker-compose; then
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE_CMD="docker compose"
        print_info "Using 'docker compose' command"
    else
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
else
    print_info "Using 'docker-compose' command"
fi

# First-time setup
print_info "Running first-time setup checks..."

# Check and copy .env file
if [ ! -f .env ]; then
    print_info "Creating .env file from .env.example..."
    cp .env.example .env
    print_info ".env file created successfully"
else
    print_info ".env file already exists, skipping..."
fi

# Check and install dependencies
if [ ! -d "vendor" ]; then
    print_info "Installing Go dependencies..."
    go mod tidy
    go mod vendor
    print_info "Dependencies installed successfully"
else
    print_info "Vendor directory exists, skipping dependency installation..."
fi

# Stop any running services
print_info "Stopping any existing services..."
pkill -f "go run ./cmd/travel/main.go" 2>/dev/null || true
pkill -f "go run ./mock" 2>/dev/null || true
$DOCKER_COMPOSE_CMD down 2>/dev/null || true

# Start Redis
print_info "Starting Redis..."
$DOCKER_COMPOSE_CMD up -d
sleep 2  # Wait for Redis to be ready

# Check if Redis is running
if ! docker ps | grep -q flight-redis; then
    print_error "Failed to start Redis"
    exit 1
fi
print_info "Redis is running"

# Start Mock Server
print_info "Starting Mock Server on port 8081..."
cd mock
nohup go run . > ../mock-server.log 2>&1 &
cd ..
sleep 2  # Wait for mock server to start

# Check if mock server is running
if ! pgrep -f "go run.*mock" > /dev/null; then
    print_error "Failed to start Mock Server"
    print_info "Check mock-server.log for details"
    exit 1
fi
print_info "Mock Server is running"

# Start Main Application
print_info "Starting Main Application on port 8080..."
nohup go run ./cmd/travel/main.go > app.log 2>&1 &
sleep 2  # Wait for app to start

# Check if app is running
if ! pgrep -f "go run ./cmd/travel/main.go" > /dev/null; then
    print_error "Failed to start Main Application"
    print_info "Check app.log for details"
    exit 1
fi
print_info "Main Application is running"

# Final status check
print_info "Waiting for services to be fully ready..."
sleep 3

echo ""
print_info "=================================="
print_info "All services started successfully!"
print_info "=================================="
echo ""
print_info "Services:"
print_info "  - Redis:            localhost:6379"
print_info "  - Mock Server:      http://localhost:8081"
print_info "  - Main Application: http://localhost:8080"
print_info "  - Swagger UI:       http://localhost:8080/swagger/index.html"
print_info "  - API Docs:         http://localhost:8080/docs"
echo ""
print_info "Logs:"
print_info "  - Main App:    tail -f app.log"
print_info "  - Mock Server: tail -f mock-server.log"
print_info "  - Redis:       docker logs -f flight-redis"
echo ""
print_info "To stop all services, run: make dev-stop"
echo ""