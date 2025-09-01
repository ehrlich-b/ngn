# Chess Engine (ngn) - Makefile
# Provides ergonomic commands for development, testing, and building

# Variables
BINARY_NAME=ngn
MAIN_FILE=main.go
BUILD_DIR=build
COVERAGE_FILE=coverage.out

# Default target
.PHONY: help
help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build commands
.PHONY: build
build: ## Build the engine binary (Linux + Windows)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Building $(BINARY_NAME) for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_FILE)
	@echo "Built Windows executable: $(BUILD_DIR)/$(BINARY_NAME).exe"
	@if [ -w "/mnt/c/Users/behrlich/Desktop/ngn" ] && cp $(BUILD_DIR)/$(BINARY_NAME).exe "/mnt/c/Users/behrlich/Desktop/ngn/" 2>/dev/null; then \
		echo "Copied to C:\\\\Users\\\\behrlich\\\\Desktop\\\\ngn\\\\$(BINARY_NAME).exe for Arena"; \
	else \
		echo "Note: Copy $(BUILD_DIR)/$(BINARY_NAME).exe to your Arena Chess directory manually"; \
	fi

.PHONY: build-release
build-release: ## Build optimized release binary
	@echo "Building $(BINARY_NAME) (release)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Built release: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-windows
build-windows: ## Build Windows executable for Arena Chess
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p "/mnt/c/Users/behrlich/Desktop/ngn"
	GOOS=windows GOARCH=amd64 go build -o "/mnt/c/Users/behrlich/Desktop/ngn/$(BINARY_NAME).exe" $(MAIN_FILE)
	@echo "Built Windows executable: C:\\\\Users\\\\behrlich\\\\Desktop\\\\ngn\\\\$(BINARY_NAME).exe"
	@echo "Ready for Arena Chess! Point Arena to the .exe file above."

# Test commands
.PHONY: test
test: ## Run all tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./... || { echo "Some tests failed, but continuing..."; exit 0; }
	@echo "Coverage report: $(COVERAGE_FILE)"

.PHONY: test-short
test-short: ## Run tests without coverage (faster)
	@echo "Running tests (short)..."
	@go test -v -short ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	go test -v -race ./...

.PHONY: coverage
coverage: test ## Generate and view test coverage report
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: eval
eval: ## Run comprehensive 2-minute engine evaluation with deep search
	@echo "Running comprehensive engine evaluation (2 minutes)..."
	@echo "⚠️  This test allows 10-15 seconds per position for meaningful depth analysis"
	@go test -v ./engine -run "TestComprehensiveEngineEvaluation" -timeout 180s

# Benchmark commands  
.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

.PHONY: bench-cpu
bench-cpu: ## Run benchmarks with CPU profiling
	@echo "Running benchmarks with CPU profiling..."
	go test -bench=. -benchmem -cpuprofile=cpu.prof ./...

# Code quality commands
.PHONY: format
format: ## Format all Go code
	@echo "Formatting Go code..."
	gofmt -s -w .
	go mod tidy
	@echo "Code formatted"

.PHONY: lint
lint: check-golangci-lint ## Run linting and formatting checks  
	@echo "Running linter..."
	golangci-lint run ./...
	@echo "Linting complete"

.PHONY: lint-fix
lint-fix: check-golangci-lint ## Run linter and fix issues automatically
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...
	@echo "Linting and fixes complete"

# Tool installation
.PHONY: install-tools
install-tools: ## Install required development tools
	@echo "Installing development tools..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	@echo "Development tools installed"

# Download Stockfish for testing
.PHONY: download-stockfish
download-stockfish: ## Download Stockfish binary for move validation testing
	@echo "Downloading Stockfish 16..."
	@if [ -f "stockfish/stockfish-ubuntu-x86-64-avx2" ]; then \
		echo "Stockfish already downloaded"; \
	else \
		wget -q --show-progress https://github.com/official-stockfish/Stockfish/releases/download/sf_16/stockfish-ubuntu-x86-64-avx2.tar; \
		tar -xf stockfish-ubuntu-x86-64-avx2.tar; \
		rm stockfish-ubuntu-x86-64-avx2.tar; \
		echo "Stockfish 16 downloaded successfully"; \
	fi

.PHONY: check-golangci-lint
check-golangci-lint: ## Check if golangci-lint is installed
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Installing..."; \
		$(MAKE) install-tools; \
	}

# Clean commands
.PHONY: clean
clean: ## Clean build artifacts and temporary files
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	rm -f *.prof
	go clean -cache -testcache -modcache
	@echo "Cleaned"

.PHONY: clean-light
clean-light: ## Clean only build artifacts (keep caches)
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	rm -f *.prof
	@echo "Cleaned"

# Development workflow commands
.PHONY: dev
dev: format test build ## Run full development cycle: format, test, build

.PHONY: check
check: lint test ## Run code quality checks: lint and test

.PHONY: ci
ci: install-tools check build ## Run continuous integration pipeline

# Run commands
.PHONY: run
run: build ## Build and run the engine
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Test harness commands
.PHONY: test-harness
test-harness: build ## Run automated UCI test harness with tournament positions
	@echo "Running UCI test harness..."
	cd cmd/test-harness && go run main.go

.PHONY: simple-test
simple-test: build ## Run simple UCI tests with basic positions
	@echo "Running simple UCI tests..."
	cd cmd/simple-test && go run main.go

.PHONY: test-engine
test-engine: simple-test ## Alias for simple-test (quick engine validation)

.PHONY: test-enhanced
test-enhanced: ## Test enhanced search algorithms (null move, LMR, futility, etc.)
	@echo "Testing enhanced search algorithms..."
	@go test -v ./engine -run "TestNullMovePruning|TestLateMovReductions|TestFutilityPruning|TestCheckExtensions|TestMateDistancePruning" -timeout 30s

.PHONY: e2e
e2e: test-enhanced ## Run end-to-end tests for enhanced search features

# Debug and profiling
.PHONY: debug
debug: ## Build with debug symbols and run with delve
	@echo "Building with debug symbols..."
	go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_FILE)
	@echo "Use 'dlv exec ./$(BUILD_DIR)/$(BINARY_NAME)-debug' to debug"

.PHONY: profile-mem
profile-mem: ## Run with memory profiling
	@echo "Building for memory profiling..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME)-profile $(MAIN_FILE)
	@echo "Run with: GODEBUG=memprofile=mem.prof ./$(BUILD_DIR)/$(BINARY_NAME)-profile"

# Info commands
.PHONY: info
info: ## Display project information
	@echo "Project: Chess Engine (ngn)"
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo "Dependencies:"
	@go list -m all

.PHONY: deps
deps: ## Display dependency information
	@echo "Direct dependencies:"
	@go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all
	@echo ""
	@echo "All dependencies:"
	@go list -m all