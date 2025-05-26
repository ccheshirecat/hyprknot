# HyprKnot Makefile
# Lightweight HTTP API wrapper for KnotDNS

# Variables
BINARY_NAME=hyprknot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build
CONFIG_DIR=/etc/hyprknot
SYSTEMD_DIR=/etc/systemd/system
BINARY_DIR=/usr/local/bin
LOG_DIR=/var/log/hyprknot

# Go build flags
LDFLAGS=-ldflags "-X main.appVersion=$(VERSION) -s -w"
BUILD_FLAGS=-trimpath

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple architectures
.PHONY: build-all
build-all: clean
	@echo "Building $(BINARY_NAME) for multiple architectures..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

	# Linux ARM
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm .

	@echo "Multi-architecture build complete"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Run the application locally
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

# Install the binary and service files (requires root)
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."

	# Check if KnotDNS is installed
	@if ! command -v knotc >/dev/null 2>&1; then \
		echo "Error: knotc command not found. Please install KnotDNS first."; \
		echo "On Ubuntu/Debian: apt install knot"; \
		echo "On CentOS/RHEL: yum install knot"; \
		exit 1; \
	fi

	# Create hyprknot user if it doesn't exist
	@if ! id hyprknot >/dev/null 2>&1; then \
		echo "Creating hyprknot user..."; \
		sudo useradd --system --shell /bin/false --no-create-home hyprknot; \
	fi

	# Add hyprknot user to knot group for socket access
	@if getent group knot >/dev/null 2>&1; then \
		sudo usermod -a -G knot hyprknot; \
		echo "Added hyprknot user to knot group"; \
	else \
		echo "Warning: knot group not found. Ensure KnotDNS is properly installed."; \
	fi

	# Create directories
	sudo mkdir -p $(CONFIG_DIR)
	sudo mkdir -p $(LOG_DIR)

	# Install binary
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(BINARY_DIR)/
	sudo chmod +x $(BINARY_DIR)/$(BINARY_NAME)

	# Install configuration
	sudo cp config.yaml $(CONFIG_DIR)/
	sudo chown hyprknot:hyprknot $(CONFIG_DIR)/config.yaml
	sudo chmod 640 $(CONFIG_DIR)/config.yaml

	# Install systemd service
	sudo cp hyprknot.service $(SYSTEMD_DIR)/
	sudo systemctl daemon-reload

	# Set permissions
	sudo chown hyprknot:hyprknot $(LOG_DIR)
	sudo chown hyprknot:hyprknot $(CONFIG_DIR)

	@echo "Installation complete"
	@echo "Edit $(CONFIG_DIR)/config.yaml and run 'sudo systemctl enable --now hyprknot' to start"

# Uninstall the application (requires root)
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."

	# Stop and disable service
	-sudo systemctl stop hyprknot
	-sudo systemctl disable hyprknot

	# Remove files
	-sudo rm -f $(BINARY_DIR)/$(BINARY_NAME)
	-sudo rm -f $(SYSTEMD_DIR)/hyprknot.service
	-sudo rm -rf $(CONFIG_DIR)
	-sudo rm -rf $(LOG_DIR)

	# Reload systemd
	sudo systemctl daemon-reload

	@echo "Uninstall complete"

# Create a release package
.PHONY: package
package: build-all
	@echo "Creating release package..."
	@mkdir -p $(BUILD_DIR)/release

	# Copy binaries
	cp $(BUILD_DIR)/$(BINARY_NAME)-* $(BUILD_DIR)/release/

	# Copy configuration and service files
	cp config.yaml $(BUILD_DIR)/release/
	cp hyprknot.service $(BUILD_DIR)/release/
	cp README.md $(BUILD_DIR)/release/ 2>/dev/null || true

	# Create tarball
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION).tar.gz release/

	@echo "Release package created: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION).tar.gz"

# Development helpers
.PHONY: dev
dev:
	@echo "Starting development server with auto-reload..."
	air -c .air.toml || go run . -config config.yaml

# Check dependencies
.PHONY: deps
deps:
	@echo "Checking dependencies..."
	go mod verify
	go mod download

# Security scan
.PHONY: security
security:
	@echo "Running security scan..."
	gosec ./...

# Generate API documentation
.PHONY: docs
docs:
	@echo "API documentation available at: http://localhost:8080/api/v1/docs"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  build-all  - Build for multiple architectures"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  test-race  - Run tests with race detection"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  tidy       - Tidy dependencies"
	@echo "  run        - Run the application locally"
	@echo "  install    - Install the application (requires root)"
	@echo "  uninstall  - Uninstall the application (requires root)"
	@echo "  package    - Create release package"
	@echo "  dev        - Start development server"
	@echo "  deps       - Check dependencies"
	@echo "  security   - Run security scan"
	@echo "  docs       - Show API documentation info"
	@echo "  help       - Show this help"
