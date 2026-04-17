.DEFAULT_GOAL := help

GO ?= go
BINARY ?= ipwatcher
CMD_PKG ?= ./cmd/ipwatcher
IMAGE ?= msyrus/ipwatcher:latest
COMPOSE ?= docker compose
UID ?= $(shell id -u)
GID ?= $(shell id -g)
PKG_DNSMANAGER ?= ./internal/dnsmanager/
CLOUDFLARE_INTEGRATION_RUN ?= TestIntegration_(GetZoneIDByName|GetZoneIDByName_NotFound|GetDNSRecords|EnsureDNSRecords_CreateAndUpdate|EnsureDNSRecords_NoUpdatesNeeded|EnsureDNSRecords_ProxiedToggle|EnsureDNSRecords_EmptyIPs)$$
ROUTE53_INTEGRATION_RUN ?= TestIntegration_Route53_(GetZoneIDByName|EnsureDNSRecords_CreateUpdateAndCleanup)$$

.PHONY: build deps install clean run test test-all test-unit test-coverage test-coverage-unit test-short test-integration test-integration-cloudflare test-integration-route53 bench fmt lint docker-build docker-build-multiarch docker-run docker-stop docker-logs docker-compose-up docker-compose-down docker-compose-logs docker-compose-restart help

# Build the binary
build:
	@echo "Building ipwatcher..."
	@$(GO) build -o $(BINARY) $(CMD_PKG)

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

# Run the service locally
run: build
	@echo "Running ipwatcher..."
	@./$(BINARY)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@$(GO) clean

# Install as systemd service
install: build
	@echo "Installing as systemd service..."
	@sudo ./install.sh

# Run the default test suite (unit tests only)
test: test-unit

# Run all tests (unit + configured integration suites)
test-all: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@$(GO) test -v ./...

# Run tests with coverage (includes integration tests)
test-coverage:
	@echo "Running tests with coverage (including integration tests)..."
	@$(GO) test -v -p 1 -parallel 1 -race -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run unit tests with coverage only
test-coverage-unit:
	@echo "Running unit tests with coverage..."
	@$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with short mode (skip long-running tests)
test-short:
	@echo "Running short tests..."
	@$(GO) test -short -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./...

# Run integration tests for all configured providers
test-integration:
	@echo "Running configured integration test suites..."
	@ran=0; \
	if [ -n "$$CLOUDFLARE_API_TOKEN" ] && [ -n "$$CLOUDFLARE_TEST_ZONE_ID" ] && [ -n "$$CLOUDFLARE_TEST_ZONE_NAME" ]; then \
		$(MAKE) --no-print-directory test-integration-cloudflare; \
		ran=1; \
	else \
		echo "Skipping Cloudflare integration tests (set CLOUDFLARE_API_TOKEN, CLOUDFLARE_TEST_ZONE_ID, CLOUDFLARE_TEST_ZONE_NAME to enable)"; \
	fi; \
	if [ -n "$$ROUTE53_TEST_ZONE_NAME" ]; then \
		$(MAKE) --no-print-directory test-integration-route53; \
		ran=1; \
	else \
		echo "Skipping Route53 integration tests (set ROUTE53_TEST_ZONE_NAME to enable)"; \
	fi; \
	if [ $$ran -eq 0 ]; then \
		echo "No integration test suites were configured."; \
		echo "See internal/dnsmanager/INTEGRATION_TESTS.md for required environment variables."; \
		exit 1; \
	fi

# Run Cloudflare integration tests only
test-integration-cloudflare:
	@echo "Running Cloudflare integration tests sequentially..."
	@if [ -z "$$CLOUDFLARE_API_TOKEN" ] || [ -z "$$CLOUDFLARE_TEST_ZONE_ID" ] || [ -z "$$CLOUDFLARE_TEST_ZONE_NAME" ]; then \
		echo "Error: Cloudflare integration test environment is incomplete"; \
		echo "Please set:"; \
		echo "  - CLOUDFLARE_API_TOKEN"; \
		echo "  - CLOUDFLARE_TEST_ZONE_ID"; \
		echo "  - CLOUDFLARE_TEST_ZONE_NAME"; \
		exit 1; \
	fi
	@$(GO) test -v -p 1 -parallel 1 -tags=integration $(PKG_DNSMANAGER) -run '$(CLOUDFLARE_INTEGRATION_RUN)' -coverprofile=coverage-cloudflare.out -covermode=atomic

# Run Route53 integration tests only
test-integration-route53:
	@echo "Running Route53 integration tests sequentially..."
	@if [ -z "$$ROUTE53_TEST_ZONE_NAME" ]; then \
		echo "Error: ROUTE53_TEST_ZONE_NAME not set"; \
		echo "AWS credentials are resolved by the standard AWS SDK credential chain."; \
		exit 1; \
	fi
	@$(GO) test -v -p 1 -parallel 1 -tags=integration $(PKG_DNSMANAGER) -run '$(ROUTE53_INTEGRATION_RUN)' -coverprofile=coverage-route53.out -covermode=atomic

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Docker: Build image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(IMAGE) .

# Docker: Build for multiple architectures
docker-build-multiarch:
	@echo "Building multi-architecture Docker image..."
	@docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE) --push .

# Docker: Run container
docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		--name ipwatcher \
		--restart unless-stopped \
		-e CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN}" \
		-e AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
		-e AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
		-e AWS_SESSION_TOKEN="${AWS_SESSION_TOKEN}" \
		-e AWS_REGION="${AWS_REGION}" \
		-v $(PWD)/config.yaml:/config/config.yaml:ro \
		-v $(PWD)/logs:/logs \
		--user $(UID):$(GID) \
		$(IMAGE)

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
	@$(COMPOSE) up -d

# Docker Compose: Stop services
docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	@$(COMPOSE) down

# Docker Compose: View logs
docker-compose-logs:
	@$(COMPOSE) logs -f

# Docker Compose: Restart services
docker-compose-restart:
	@$(COMPOSE) restart

# Show help
help:
	@echo "Available targets:"
	@echo "  build                - Build the binary"
	@echo "  deps                 - Download and tidy dependencies"
	@echo "  run                  - Build and run locally"
	@echo "  clean                - Clean build artifacts"
	@echo "  install              - Install as systemd service (requires sudo)"
	@echo "  test                 - Run the default test suite (unit tests only)"
	@echo "  test-all             - Run unit tests and all configured integration suites"
	@echo "  test-unit            - Run unit tests only"
	@echo "  test-coverage        - Run all tests with coverage report (includes integration)"
	@echo "  test-coverage-unit   - Run unit tests with coverage report"
	@echo "  test-short           - Run short tests"
	@echo "  test-integration     - Run all configured integration test suites"
	@echo "  test-integration-cloudflare - Run Cloudflare integration tests only"
	@echo "  test-integration-route53 - Run Route53 integration tests only"
	@echo "  bench                - Run benchmarks"
	@echo "  fmt                  - Format code"
	@echo "  lint                 - Lint code"
	@echo "  docker-build         - Build Docker image"
	@echo "  docker-build-multiarch - Build multi-architecture Docker image"
	@echo "  docker-run           - Run Docker container with Cloudflare/AWS env passthrough"
	@echo "  docker-stop          - Stop Docker container"
	@echo "  docker-logs          - View Docker container logs"
	@echo "  docker-compose-up    - Start services with Docker Compose"
	@echo "  docker-compose-down  - Stop services with Docker Compose"
	@echo "  docker-compose-logs  - View Docker Compose logs"
	@echo "  docker-compose-restart - Restart Docker Compose services"
	@echo "  help                 - Show this help message"
