.PHONY: help build run test test-unit test-race test-load docker-build docker-up docker-down clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building application..."
	go build -o bin/rate-limiter ./cmd/server

run: ## Run the application locally
	@echo "Running application..."
	go run ./cmd/server/main.go

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

test: ## Run all tests
	@echo "Running all tests..."
	go test -v ./tests/...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v ./tests/unit/

test-race: ## Run race condition tests
	@echo "Running race condition tests..."
	go test -race -v ./tests/unit/race_test.go

test-load: ## Run load and performance tests
	@echo "Running load tests..."
	go test -v ./tests/load/ -timeout 30m

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./tests/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker-compose build

docker-up: ## Start all services with Docker Compose
	@echo "Starting services..."
	docker-compose up --build -d
	@echo "Waiting for services to be healthy..."
	sleep 10
	@echo "Services started successfully!"
	@echo "Rate Limiter instances:"
	@echo "  - Instance 1: http://localhost:8080"
	@echo "  - Instance 2: http://localhost:8081"
	@echo "  - Instance 3: http://localhost:8082"

docker-down: ## Stop all Docker services
	@echo "Stopping services..."
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-clean: ## Stop services and remove volumes
	@echo "Cleaning up Docker resources..."
	docker-compose down -v
	docker system prune -f

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

.DEFAULT_GOAL := help
