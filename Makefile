.PHONY: all build run dev test test-coverage test-integration fmt vet tidy mocks clean-mocks check clean docker-redis docker-redis-stop help

# Build settings
BUILD_DIR := build
BINARY_API := $(BUILD_DIR)/api

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod

# Default target
all: fmt test vet

# Build API binary
build:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BINARY_API) ./cmd/api

# Run the API (build first)
run: build
	./$(BINARY_API)

# Run in development mode (no build)
dev:
	$(GOCMD) run ./cmd/api

# Run tests
test:
	$(GOTEST) ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run integration tests only
test-integration:
	$(GOTEST) -v ./internal/proxy/redis/...

# Format code and fix imports
fmt:
	go fmt ./...
	goimports -w .

# Run go vet
vet:
	go vet ./...

# Tidy go.mod
tidy:
	$(GOMOD) tidy

# Generate mocks using mockery
mocks:
	mockery --config config/mockery/mockery.yaml

# Clean existing mocks (except logs/mocks which is manually maintained)
clean-mocks:
	find . -path "./internal/common/logs/mocks" -prune -o -name "mocks" -type d -exec rm -rf {} + 2>/dev/null || true

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Clean, regenerate mocks, and run tests
check: clean-mocks mocks test

# Start Redis container for local development
docker-redis:
	docker run -d --name redis-proxy -p 6379:6379 redis:alpine || docker start redis-proxy

# Stop Redis container
docker-redis-stop:
	docker stop redis-proxy

# Show help
help:
	@echo "Available commands:"
	@echo "  make build           - Build API binary to ./build/"
	@echo "  make run             - Build and run API"
	@echo "  make dev             - Run API without building (go run)"
	@echo "  make test            - Run unit tests"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make test-integration- Run integration tests"
	@echo "  make fmt             - Format code"
	@echo "  make vet             - Run go vet"
	@echo "  make tidy            - Run go mod tidy"
	@echo "  make mocks           - Generate mocks using mockery"
	@echo "  make clean-mocks     - Remove generated mocks"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make check           - Clean mocks, regenerate, and test"
	@echo "  make docker-redis    - Start Redis container"
	@echo "  make docker-redis-stop - Stop Redis container"

