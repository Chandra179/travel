ins:
	go mod tidy && go mod vendor

up:
	docker compose up -d

build:
	docker compose up --build -d

run:
	go run cmd/travel/main.go

# Development environment setup and management
.PHONY: dev-setup
dev-setup:
	@chmod +x dev-setup.sh
	@./dev-setup.sh

.PHONY: dev-stop
dev-stop:
	@chmod +x dev-stop.sh
	@./dev-stop.sh

.PHONY: dev-restart
dev-restart: dev-stop dev-setup

.PHONY: dev-logs
dev-logs:
	@echo "=== Main Application Logs ==="
	@tail -f app.log

.PHONY: dev-logs-mock
dev-logs-mock:
	@echo "=== Mock Server Logs ==="
	@tail -f mock-server.log

.PHONY: dev-logs-redis
dev-logs-redis:
	@echo "=== Redis Logs ==="
	@docker logs -f flight-redis

.PHONY: dev-status
dev-status:
	@echo "=== Service Status ==="
	@echo -n "Redis:            "
	@docker ps | grep -q flight-redis && echo "✓ Running" || echo "✗ Not running"
	@echo -n "Mock Server:      "
	@pgrep -f "go run.*mock" > /dev/null && echo "✓ Running" || echo "✗ Not running"
	@echo -n "Main Application: "
	@pgrep -f "go run ./cmd/travel/main.go" > /dev/null && echo "✓ Running" || echo "✗ Not running"

.PHONY: dev-clean
dev-clean: dev-stop
	@echo "Cleaning up development files..."
	@rm -f app.log mock-server.log
	@docker-compose down -v
	@echo "Clean complete!"

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make ins           - Install dependencies"
	@echo "  make up            - Start Redis only"
	@echo "  make run           - Run main application only"
	@echo "  make build         - Build and start with docker-compose"
	@echo ""
	@echo "Development commands:"
	@echo "  make dev-setup     - Setup and start all services (Redis, Mock, App)"
	@echo "  make dev-stop      - Stop all services"
	@echo "  make dev-restart   - Restart all services"
	@echo "  make dev-status    - Check status of all services"
	@echo "  make dev-logs      - Show main application logs"
	@echo "  make dev-logs-mock - Show mock server logs"
	@echo "  make dev-logs-redis- Show Redis logs"
	@echo "  make dev-clean     - Stop services and clean up all files"
	@echo ""