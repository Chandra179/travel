#!/bin/bash

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

# Function to check if port is valid
is_valid_port() {
    local port=$1
    if [[ "$port" =~ ^[0-9]+$ ]] && [ "$port" -ge 1 ] && [ "$port" -le 65535 ]; then
        return 0
    else
        return 1
    fi
}

# Parse port arguments
APP_PORT=${1:-8080}
MOCK_PORT=${2:-8081}

# Validate port arguments
if [ $# -eq 1 ]; then
    print_error "Both APP_PORT and MOCK_PORT must be provided, or none at all"
    print_info "Usage: $0 [APP_PORT MOCK_PORT]"
    print_info "Example: $0 8080 8081"
    exit 1
fi

if [ $# -ge 2 ]; then
    if ! is_valid_port "$APP_PORT"; then
        print_error "Invalid APP_PORT: $APP_PORT (must be between 1-65535)"
        exit 1
    fi

    if ! is_valid_port "$MOCK_PORT"; then
        print_error "Invalid MOCK_PORT: $MOCK_PORT (must be between 1-65535)"
        exit 1
    fi
fi

print_info "Stopping services on ports - APP: $APP_PORT, MOCK: $MOCK_PORT"

# Stop Go processes
print_info "Stopping Go applications..."
pkill -f "go run ./cmd/travel/main.go" 2>/dev/null && print_info "Main application stopped" || print_warning "Main application was not running"
pkill -f "go run.*mock" 2>/dev/null && print_info "Mock server stopped" || print_warning "Mock server was not running"

# Kill processes on specific ports
print_info "Killing processes on port $APP_PORT..."
if lsof -ti:$APP_PORT >/dev/null 2>&1; then
    lsof -ti:$APP_PORT | xargs kill -9 2>/dev/null && print_info "Processes on port $APP_PORT killed" || print_warning "Failed to kill some processes on port $APP_PORT"
else
    print_info "No processes found on port $APP_PORT"
fi

print_info "Killing processes on port $MOCK_PORT..."
if lsof -ti:$MOCK_PORT >/dev/null 2>&1; then
    lsof -ti:$MOCK_PORT | xargs kill -9 2>/dev/null && print_info "Processes on port $MOCK_PORT killed" || print_warning "Failed to kill some processes on port $MOCK_PORT"
else
    print_info "No processes found on port $MOCK_PORT"
fi

# Determine which docker compose command to use
DOCKER_COMPOSE_CMD="docker-compose"
if ! command -v docker-compose >/dev/null 2>&1; then
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE_CMD="docker compose"
    fi
fi

# Stop Docker services
print_info "Stopping Docker services..."
$DOCKER_COMPOSE_CMD down 2>/dev/null && print_info "Docker services stopped" || print_warning "Docker services were not running or failed to stop"

echo ""
print_info "All services stopped successfully!"
echo ""