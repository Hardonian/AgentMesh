.PHONY: all build dev test test-race lint clean docker-build demo release-check

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

demo: build
	@./bin/agentmesh demo run

test:
	@echo "Running test suite..."
	@go test -p 1 -v ./pkg/... ./internal/... ./tests

test-race:
	@echo "Running race detector test suite..."
	@go test -p 1 -race -v ./pkg/... ./internal/... ./tests

lint:
	@echo "Checking formatting and vet..."
	@go fmt ./...
	@go vet ./...

release-check: lint test-race
	@echo "Verifying 35-point Definition-of-Done certification suite..."
	@go test -p 1 -v ./tests -run TestPhase5DefinitionOfDone35Certifications
	@echo "Verifying 15-scenario Adversarial Red Team suite..."
	@go test -p 1 -v ./tests -run TestP0RedTeamScenarios
	@echo "Release verification: 100% PASS - Ready for v1.0.0"

clean:
	@echo "Cleaning artifacts..."
	@rm -rf bin/

docker-build:
	@echo "Building Docker images..."
	@docker build -f deploy/docker/Dockerfile.controller -t agentmesh-controller:latest .
	@docker build -f deploy/docker/Dockerfile.proxy -t agentmesh-proxy:latest .
