# Synapse Build Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
AGENT_BINARY=sutra-agent
SYNCTL_BINARY=sutra-ctl

# Build directories
AGENT_SRC=./cmd/sutra-agent
SYNCTL_SRC=./cmd/sutra-ctl
BIN_DIR=./bin

# Platform-specific binary extensions
ifeq ($(OS),Windows_NT)
    AGENT_BINARY := $(AGENT_BINARY).exe
    SYNCTL_BINARY := $(SYNCTL_BINARY).exe
endif

.PHONY: all build clean test deps help agent synctl

all: build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download and install dependencies
	$(GOMOD) download
	$(GOMOD) tidy

build: deps agent synctl ## Build all binaries

agent: ## Build sutra-agent
	@echo Building sutra-agent...
	@if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(AGENT_BINARY) $(AGENT_SRC)
	@echo Built $(BIN_DIR)/$(AGENT_BINARY)

synctl: ## Build sutra-ctl
	@echo Building sutra-ctl...
	@if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(SYNCTL_BINARY) $(SYNCTL_SRC)
	@echo Built $(BIN_DIR)/$(SYNCTL_BINARY)

test: ## Run tests
	$(GOTEST) -v ./...

test-quick: agent synctl ## Run quick single agent test
	@echo "Running quick test..."
	powershell.exe -ExecutionPolicy RemoteSigned .\quick_test.ps1

test-mesh: agent synctl ## Run full multi-agent mesh test
	@echo "Running mesh test..."
	powershell.exe -ExecutionPolicy RemoteSigned .\test_mesh.ps1

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

proto: ## Generate protobuf code (requires protoc)
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/v1/synapse.proto
	@echo "✓ Generated protobuf code"

install: build ## Install binaries to GOPATH/bin
	@echo "Installing binaries..."
	cp $(BIN_DIR)/$(AGENT_BINARY) $(GOPATH)/bin/
	cp $(BIN_DIR)/$(SYNCTL_BINARY) $(GOPATH)/bin/
	@echo "✓ Installed to $(GOPATH)/bin/"

run-agent: agent ## Build and run sutra-agent with example config
	@echo "Starting sutra-agent..."
	$(BIN_DIR)/$(AGENT_BINARY) -config configs/agent.example.yaml

demo: build ## Run a quick demo
	@echo "=== Synapse V1.0 Demo ==="
	@echo "1. Starting agent in background..."
	@$(BUILD_DIR)/$(AGENT_BINARY) -config configs/agent.example.yaml &
	@sleep 2
	@echo "2. Testing connectivity..."
	@$(BUILD_DIR)/$(SYNCTL_BINARY) status || echo "Agent not ready yet"
	@echo "3. Demo complete (agent may still be running)"

format: ## Format Go code
	gofmt -s -w .

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

scout-csv: ## Run CSV scout demo
	@echo "Running CSV scout demo..."
	cd examples/python-scouts/chronicler-csv-scout && python scout.py --csv sample_data.csv --verbose

scout-llm: ## Run LLM scout demo
	@echo "Running LLM scout demo..."
	cd examples/python-scouts/interpreter-llm-scout && python scout.py --file sample_text.txt --enhance --verbose

# Development targets
dev-setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	$(GOGET) -u google.golang.org/protobuf/cmd/protoc-gen-go
	$(GOGET) -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@echo "✓ Development environment ready"

dev-watch: ## Watch for changes and rebuild (requires entr)
	find . -name "*.go" | entr -r make build
