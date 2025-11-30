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

# Check and copy .env file
if [ ! -f .env ]; then
    print_info "Creating .env file from .env.example..."
    cp .env.example .env
    print_info ".env file created successfully"
else
    print_info ".env file already exists"
fi

# Load environment variables from .env
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

APP_PORT=${APP_PORT:-8080}
MOCK_PORT=${MOCK_PORT:-8081}

print_info "Using ports - APP: $APP_PORT, MOCK: $MOCK_PORT"

# Stop any existing containers
print_info "Stopping any existing containers..."
$DOCKER_COMPOSE_CMD down 2>/dev/null || true

# Build and start all services
print_info "Building and starting all services..."
$DOCKER_COMPOSE_CMD up -d --build

# Wait for services to be healthy
print_info "Waiting for services to be ready..."
sleep 5

# Check if all services are running
print_info "Checking service health..."

MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    REDIS_STATUS=$($DOCKER_COMPOSE_CMD ps redis --format json | grep -o '"Health":"[^"]*"' | cut -d'"' -f4)
    MOCK_STATUS=$($DOCKER_COMPOSE_CMD ps mock-server --format json | grep -o '"Health":"[^"]*"' | cut -d'"' -f4)
    APP_STATUS=$($DOCKER_COMPOSE_CMD ps app --format json | grep -o '"Health":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$REDIS_STATUS" = "healthy" ] && [ "$MOCK_STATUS" = "healthy" ] && [ "$APP_STATUS" = "healthy" ]; then
        print_info "All services are healthy!"
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        print_error "Services failed to become healthy after $MAX_RETRIES attempts"
        print_info "Service status:"
        $DOCKER_COMPOSE_CMD ps
        print_info "Logs:"
        $DOCKER_COMPOSE_CMD logs --tail=50
        exit 1
    fi
    
    echo -n "."
    sleep 2
done

echo ""
print_info "=================================="
print_info "All services started successfully!"
print_info "=================================="
echo ""
print_info "Services:"
print_info "  - Redis:            localhost:6379"
print_info "  - Mock Server:      http://localhost:$MOCK_PORT"
print_info "  - Main Application: http://localhost:$APP_PORT"
print_info "  - Swagger UI:       http://localhost:$APP_PORT/swagger/index.html"
print_info "  - API Docs:         http://localhost:$APP_PORT/docs"
echo ""
print_info "Logs:"
print_info "  - All services:  docker-compose logs -f"
print_info "  - Main App:      docker-compose logs -f app"
print_info "  - Mock Server:   docker-compose logs -f mock-server"
print_info "  - Redis:         docker-compose logs -f redis"
echo ""
print_info "To stop all services, run: docker-compose down"
print_info "To rebuild services, run: docker-compose up -d --build"
echo ""