.PHONY: build run test test-coverage lint fmt clean docker-build docker-up docker-down migrate-up migrate-down migrate-create  deps

# Variables
APP_NAME=transaction-service
BINARY_NAME=server
DOCKER_IMAGE=transaction-service:latest
DB_URL=postgres://admin:admin@localhost:5432/transactions?sslmode=disable

# Build the application
build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/server

# Run the application
run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server/main.go

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./... -count=1

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
	fi

# Format code
fmt: ## Format code with gofmt
	@echo "Formatting code..."
	@gofmt -s -w .
	@go mod tidy

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-up: ## Start all services with docker-compose
	@echo "Starting services with docker-compose..."
	@docker compose --env-file stack.env up -d

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker compose --env-file stack.env down

docker-logs: ## View docker-compose logs
	@docker compose --env-file stack.env logs -f

compose-up:
	@echo "Running docker compose up --build..."
	docker compose --env-file stack.env up --build

# Staging Docker commands
staging-up: ## Start all services with staging docker-compose
	@echo "Starting staging services..."
	@docker compose -f docker-compose.staging.yml --env-file stack.env up -d

staging-down: ## Stop staging services
	@echo "Stopping staging services..."
	@docker compose -f docker-compose.staging.yml --env-file stack.env down

staging-logs: ## View staging docker-compose logs
	@docker compose -f docker-compose.staging.yml --env-file stack.env logs -f

staging-compose-up: ## Build and start staging services
	@echo "Running staging docker compose up --build..."
	docker compose -f docker-compose.staging.yml --env-file stack.env up --build

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@if command -v atlas > /dev/null; then \
		atlas schema apply --env local --auto-approve; \
	else \
		echo "Atlas not installed. Install it from https://atlasgo.io/getting-started#installation"; \
	fi

migrate-status: ## Check migration status
	@echo "Checking migration status..."
	@if command -v atlas > /dev/null; then \
		atlas schema inspect --env local; \
	else \
		echo "Atlas not installed."; \
	fi

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=my_migration"; \
		exit 1; \
	fi
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	FILENAME="db/migrations/$${TIMESTAMP}_$(NAME).sql"; \
	touch "$$FILENAME"; \
	echo "-- Migration: $(NAME)" > "$$FILENAME"; \
	echo "-- Created: $$(date)" >> "$$FILENAME"; \
	echo "" >> "$$FILENAME"; \
	echo "-- Add your SQL statements below" >> "$$FILENAME"; \
	echo "" >> "$$FILENAME"; \
	echo "Created migration: $$FILENAME"

deps: ## Install project dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing development tools..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Dependencies installed!"