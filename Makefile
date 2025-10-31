.PHONY: build install clean run test test-unit test-coverage test-coverage-unit test-short test-integration bench fmt lint docker-build docker-run docker-stop docker-compose-up docker-compose-down help

# Build the binary
build:
	@echo "Building ipwatcher..."
	@go build -o ipwatcher ./cmd/ipwatcher

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run the service locally
run: build
	@echo "Running ipwatcher..."
	@./ipwatcher

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f ipwatcher
	@go clean

# Install as systemd service
install: build
	@echo "Installing as systemd service..."
	@sudo ./install.sh

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -v ./...

# Run all tests (unit + integration)
test:
	@echo "Running all tests (unit + integration)..."
	@go test -v -tags=integration ./...

# Run tests with coverage (includes integration tests)
test-coverage:
	@echo "Running tests with coverage (including integration tests)..."
	@go test -v -race -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run unit tests with coverage only
test-coverage-unit:
	@echo "Running unit tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with short mode (skip long-running tests)
test-short:
	@echo "Running short tests..."
	@go test -short -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run integration tests only (requires Cloudflare credentials)
test-integration:
	@echo "Running integration tests only..."
	@if [ -z "$$CLOUDFLARE_API_TOKEN" ]; then \
		echo "Error: CLOUDFLARE_API_TOKEN not set"; \
		echo "Please set the following environment variables:"; \
		echo "  - CLOUDFLARE_API_TOKEN"; \
		echo "  - CLOUDFLARE_TEST_ZONE_ID"; \
		echo "  - CLOUDFLARE_TEST_ZONE_NAME"; \
		echo "See internal/dnsmanager/INTEGRATION_TESTS.md for details"; \
		exit 1; \
	fi
	@go test -v -tags=integration ./internal/dnsmanager/

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Docker: Build image
docker-build:
	@echo "Building Docker image..."
	@docker build -t msyrus/ipwatcher:latest .

# Docker: Build for multiple architectures
docker-build-multiarch:
	@echo "Building multi-architecture Docker image..."
	@docker buildx build --platform linux/amd64,linux/arm64 -t msyrus/ipwatcher:latest --push .

# Docker: Run container
docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		--name ipwatcher \
		--restart unless-stopped \
		-e CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN}" \
		-v $(PWD)/config.yaml:/config/config.yaml:ro \
		-v $(PWD)/logs:/logs \
		--user $(shell id -u):$(shell id -g) \
		msyrus/ipwatcher:latest

# Docker: Stop and remove container
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop ipwatcher || true
	@docker rm ipwatcher || true

# Docker: View logs
docker-logs:
	@docker logs -f ipwatcher

# Docker Compose: Start services
docker-compose-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

# Docker Compose: Stop services
docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	@docker-compose down

# Docker Compose: View logs
docker-compose-logs:
	@docker-compose logs -f

# Docker Compose: Restart services
docker-compose-restart:
	@docker-compose restart

# Show help
help:
	@echo "Available targets:"
	@echo "  build                - Build the binary"
	@echo "  deps                 - Download and tidy dependencies"
	@echo "  run                  - Build and run locally"
	@echo "  clean                - Clean build artifacts"
	@echo "  install              - Install as systemd service (requires sudo)"
	@echo "  test                 - Run all tests (unit + integration)"
	@echo "  test-unit            - Run unit tests only"
	@echo "  test-coverage        - Run all tests with coverage report (includes integration)"
	@echo "  test-coverage-unit   - Run unit tests with coverage report"
	@echo "  test-short           - Run short tests"
	@echo "  test-integration     - Run integration tests only (requires Cloudflare credentials)"
	@echo "  bench                - Run benchmarks"
	@echo "  fmt                  - Format code"
	@echo "  lint                 - Lint code"
	@echo "  docker-build         - Build Docker image"
	@echo "  docker-build-multiarch - Build multi-architecture Docker image"
	@echo "  docker-run           - Run Docker container"
	@echo "  docker-stop          - Stop Docker container"
	@echo "  docker-logs          - View Docker container logs"
	@echo "  docker-compose-up    - Start services with Docker Compose"
	@echo "  docker-compose-down  - Stop services with Docker Compose"
	@echo "  docker-compose-logs  - View Docker Compose logs"
	@echo "  docker-compose-restart - Restart Docker Compose services"
	@echo "  help                 - Show this help message"
