.PHONY: all build test lint run docker-build docker-run clean help

# Variables
BINARY_NAME=server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS=-ldflags="-w -s -X main.Version=$(VERSION)"
DOCKER_IMAGE=taskservice

all: lint test build

## Build
build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/server

## Test
test:
	go test -race -coverprofile=coverage.out ./...

test-short:
	go test -short ./...

test-integration:
	go test -tags=integration -race ./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html

## Lint
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

## Run
run:
	go run ./cmd/server

run-watch:
	air -c .air.toml

## Docker
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

docker-run: docker-build
	docker run -p 8080:8080 $(DOCKER_IMAGE):latest

docker-compose-up:
	docker-compose up --build

docker-compose-down:
	docker-compose down -v

## Database
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

## Clean
clean:
	rm -rf bin/ coverage.out coverage.html

## Dependencies
deps:
	go mod download
	go mod verify

deps-update:
	go get -u ./...
	go mod tidy

## Security
security-scan:
	gosec -fmt json -out security-report.json ./...
	trivy fs --severity HIGH,CRITICAL .

## Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  test            - Run tests with coverage"
	@echo "  lint            - Run linter"
	@echo "  run             - Run locally"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-compose-up - Run with docker-compose"
	@echo "  clean           - Clean build artifacts"
