# k3k-tui Makefile

.PHONY: build build-all clean test lint run install help

BINARY_NAME=k3k-tui
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d %H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)' -X main.commitHash=$(COMMIT_HASH)"

# Default target
all: build

## Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## Build for all platforms
build-all: build-linux build-darwin build-windows

## Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

## Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .

## Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

## Run the application
run:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) . && ./$(BINARY_NAME)

## Run with specific kubeconfig
run-with-config:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) . && ./$(BINARY_NAME) --kubeconfig $(KUBECONFIG)

## Install the application
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) .

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Lint the code
lint:
	golangci-lint run

## Format the code
fmt:
	$(GOCMD) fmt ./...

## Update dependencies
update-deps:
	$(GOGET) -u ./...
	$(GOMOD) tidy

## Create release archives
release: build-all
	mkdir -p release
	tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 README.md LICENSE
	tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 README.md LICENSE
	tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 README.md LICENSE
	tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 README.md LICENSE
	zip release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe README.md LICENSE

## Show help
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Development helpers

## Watch for changes and rebuild
dev:
	while true; do \
		inotifywait -r -e modify,create,delete --include='\.go$$' .; \
		echo "Rebuilding..."; \
		make build; \
	done

## Run with demo data (requires a running k3k cluster)
demo:
	@echo "Starting k3k-tui demo..."
	@echo "Make sure you have k3k installed and configured in your cluster"
	$(MAKE) run

## Check if k3k CRDs are available
check-k3k:
	@echo "Checking for k3k CRDs..."
	@kubectl get crd clusters.k3k.io >/dev/null 2>&1 && echo "✓ k3k CRDs found" || echo "✗ k3k CRDs not found - please install k3k first"
	@kubectl get clusters.k3k.io --all-namespaces 2>/dev/null | head -10 || echo "No k3k clusters found"

## Initialize a new k3k cluster for testing
init-test-cluster:
	@echo "Creating test namespace and cluster..."
	kubectl create namespace k3k-test --dry-run=client -o yaml | kubectl apply -f -
	@cat <<EOF | kubectl apply -f - \
	apiVersion: k3k.io/v1beta1 \
	kind: Cluster \
	metadata: \
	  name: test-cluster \
	  namespace: k3k-test \
	spec: \
	  mode: shared \
	  servers: 1 \
	  agents: 0 \
	  persistence: \
	    type: ephemeral \
	EOF
	@echo "Test cluster created. Run 'make demo' to view it in the TUI."