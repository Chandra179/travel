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

if ! is_valid_port "$APP_PORT"; then
    print_error "Invalid APP_PORT: $APP_PORT (must be between 1-65535)"
    exit 1
fi

if ! is_valid_port "$MOCK_PORT"; then
    print_error "Invalid MOCK_PORT: $MOCK_PORT (must be between 1-65535)"
    exit 1
fi

if [ "$APP_PORT" -eq "$MOCK_PORT" ]; then
    print_error "APP_PORT and MOCK_PORT cannot be the same: $APP_PORT"
    exit 1
fi

print_info "Using ports - APP: $APP_PORT, MOCK: $MOCK_PORT"

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
    print_info ".env file already exists"
fi

# Update APP_PORT in .env file
print_info "Updating APP_PORT in .env file..."
if grep -q "^APP_PORT=" .env; then
    # Update existing APP_PORT
    sed -i.bak "s/^APP_PORT=.*/APP_PORT=$APP_PORT/" .env && rm -f .env.bak
else
    # Add APP_PORT if it doesn't exist
    echo "APP_PORT=$APP_PORT" >> .env
fi
print_info "APP_PORT set to $APP_PORT in .env"

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
pkill -f "go run.*mock" 2>/dev/null || true
pkill -f "mock/main.go" 2>/dev/null || true

# Kill processes on the ports we're about to use
print_info "Checking for processes on ports $APP_PORT and $MOCK_PORT..."
lsof -ti:$APP_PORT | xargs kill -9 2>/dev/null || true
lsof -ti:$MOCK_PORT | xargs kill -9 2>/dev/null || true

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

# Get absolute paths
PROJECT_ROOT=$(pwd)
MOCK_LOG_FILE="$PROJECT_ROOT/mock-server.log"

# Check if mock directory exists
if [ ! -d "mock" ]; then
    print_error "Mock directory not found!"
    exit 1
fi

# Check if mock/main.go exists
if [ ! -f "mock/main.go" ]; then
    print_error "mock/main.go not found!"
    exit 1
fi

# Start Mock Server
print_info "Starting Mock Server on port $MOCK_PORT..."
print_info "Mock directory: $(pwd)/mock"
print_info "Log file: $MOCK_LOG_FILE"

# Clear old log file
> "$MOCK_LOG_FILE"

# Start mock server in background
cd mock || exit 1
go run . $MOCK_PORT >> "$MOCK_LOG_FILE" 2>&1 &
MOCK_PID=$!
echo $MOCK_PID > "$PROJECT_ROOT/.mock.pid"
cd "$PROJECT_ROOT"

sleep 4  # Wait for mock server to start

# Check if PID file was created
if [ ! -f ".mock.pid" ]; then
    print_error "Failed to create mock server PID file"
    print_info "Log contents:"
    cat "$MOCK_LOG_FILE"
    exit 1
fi

# Verify PID from file
MOCK_PID=$(cat .mock.pid)
print_info "Mock Server PID: $MOCK_PID"

# Check if process is actually running (give it some grace period)
sleep 1
if ! ps -p $MOCK_PID > /dev/null 2>&1; then
    print_error "Mock Server process (PID: $MOCK_PID) is not running"
    print_info "Log contents:"
    cat "$MOCK_LOG_FILE"
    exit 1
fi

# Check if port is listening
MAX_RETRIES=10
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if lsof -i:$MOCK_PORT > /dev/null 2>&1; then
        print_info "Mock Server is listening on port $MOCK_PORT"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        print_error "Mock Server not listening on port $MOCK_PORT after $MAX_RETRIES attempts"
        print_info "Process status:"
        ps -p $MOCK_PID || echo "Process not found"
        print_info "Log contents:"
        cat "$MOCK_LOG_FILE"
        exit 1
    fi
    sleep 1
done

print_info "Mock Server is running (PID: $MOCK_PID)"

# Start Main Application
print_info "Starting Main Application on port $APP_PORT..."
> app.log  # Clear old log
nohup go run ./cmd/travel/main.go >> app.log 2>&1 &
APP_PID=$!
echo $APP_PID > .app.pid
sleep 4  # Wait for app to start

# Check if app is running
if ! ps -p $APP_PID > /dev/null 2>&1; then
    print_error "Main Application process (PID: $APP_PID) is not running"
    print_info "Log contents:"
    tail -30 app.log
    exit 1
fi

# Check if port is listening
MAX_RETRIES=10
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if lsof -i:$APP_PORT > /dev/null 2>&1; then
        print_info "Main Application is listening on port $APP_PORT"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        print_error "Main Application not listening on port $APP_PORT after $MAX_RETRIES attempts"
        print_info "Process status:"
        ps -p $APP_PID || echo "Process not found"
        print_info "Log contents:"
        tail -30 app.log
        exit 1
    fi
    sleep 1
done

print_info "Main Application is running (PID: $APP_PID)"

echo ""
print_info "=================================="
print_info "All services started successfully!"
print_info "=================================="
echo ""
print_info "Services:"
print_info "  - Redis:            localhost:6379"
print_info "  - Mock Server:      http://localhost:$MOCK_PORT (PID: $MOCK_PID)"
print_info "  - Main Application: http://localhost:$APP_PORT (PID: $APP_PID)"
print_info "  - Swagger UI:       http://localhost:$APP_PORT/swagger/index.html"
print_info "  - API Docs:         http://localhost:$APP_PORT/docs"
echo ""
print_info "Logs:"
print_info "  - Main App:    tail -f app.log"
print_info "  - Mock Server: tail -f mock-server.log"
print_info "  - Redis:       docker logs -f flight-redis"
echo ""
print_info "To stop all services, run: make dev-stop $APP_PORT $MOCK_PORT"
echo ""