.PHONY: all test fmt vet tidy mocks clean-mocks refresh-mocks help

# Default target
all: fmt test vet

# Run tests
test:
	go test ./...

# Format code and fix imports
fmt:
	go fmt ./...
	goimports -w .

# Run go vet
vet:
	go vet ./...

# Tidy go.mod
tidy:
	go mod tidy

# Generate mocks using mockery
mocks:
	mockery

# Clean existing mocks (except logs/mocks which is manually maintained)
clean-mocks:
	find . -path "./internal/common/logs/mocks" -prune -o -name "mocks" -type d -exec rm -rf {} + 2>/dev/null || true

# Clean, regenerate mocks, and run tests
check: clean-mocks mocks test

# Show help
help:
	@echo "Available commands:"
	@echo "  make test          - Run unit tests"
	@echo "  make fmt           - Format code and fix imports using gofmt and goimports"
	@echo "  make vet           - Run go vet"
	@echo "  make tidy          - Run go mod tidy"
	@echo "  make mocks         - Generate mocks using mockery"
	@echo "  make clean-mocks   - Remove generated mocks"
	@echo "  make check         - Clean mocks, regenerate them, and run tests"
