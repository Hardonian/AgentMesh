.PHONY: all build dev test test-race lint clean docker-build

all: build test

build:
	@echo "Building AgentMesh binaries..."
	@go build -o bin/agentmesh ./cmd/agentmesh
	@go build -o bin/agentmesh-proxy ./cmd/agentmesh-proxy
	@go build -o bin/agentmesh-controller ./cmd/agentmesh-controller
	@go build -o bin/agentmesh-worker ./cmd/agentmesh-worker
	@echo "Build complete. Binaries available in bin/"

dev:
	@echo "Starting AgentMesh Control Plane..."
	@go run ./cmd/agentmesh-controller

test:
	@echo "Running test suite..."
	@go test -v ./pkg/... ./internal/...

test-race:
	@echo "Running race detector test suite..."
	@go test -race -v ./pkg/... ./internal/...

lint:
	@echo "Checking formatting and vet..."
	@go fmt ./...
	@go vet ./...

clean:
	@echo "Cleaning artifacts..."
	@rm -rf bin/

docker-build:
	@echo "Building Docker images..."
	@docker build -f deploy/docker/Dockerfile.controller -t agentmesh-controller:latest .
	@docker build -f deploy/docker/Dockerfile.proxy -t agentmesh-proxy:latest .
