.PHONY: all build test lint run docker-build docker-run clean help

# Variables
BINARY_NAME=server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS=-ldflags="-w -s -X main.Version=$(VERSION)"
DOCKER_IMAGE=taskservice
REGISTRY=ghcr.io/buildguard-test-lab/taskservice
STAGING_CTX=kind-staging
PROD_CTX=kind-prod
NAMESPACE=taskservice

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

## Kind Clusters
kind-create:
	kind create cluster --config k8s/kind-staging.yaml
	kind create cluster --config k8s/kind-prod.yaml
	kubectl create namespace $(NAMESPACE) --context $(STAGING_CTX)
	kubectl create namespace $(NAMESPACE) --context $(PROD_CTX)

kind-delete:
	kind delete cluster --name staging
	kind delete cluster --name prod

## Deploy to Local Kind (from local build)
deploy-local-staging: docker-build
	kind load docker-image $(DOCKER_IMAGE):$(VERSION) --name staging
	kubectl set image deployment/taskservice taskservice=$(DOCKER_IMAGE):$(VERSION) \
		-n $(NAMESPACE) --context $(STAGING_CTX) 2>/dev/null || \
		kubectl apply -f k8s/local-deployment.yaml -n $(NAMESPACE) --context $(STAGING_CTX)
	kubectl rollout status deployment/taskservice -n $(NAMESPACE) --context $(STAGING_CTX)

deploy-local-prod: docker-build
	kind load docker-image $(DOCKER_IMAGE):$(VERSION) --name prod
	kubectl set image deployment/taskservice taskservice=$(DOCKER_IMAGE):$(VERSION) \
		-n $(NAMESPACE) --context $(PROD_CTX) 2>/dev/null || \
		kubectl apply -f k8s/local-deployment.yaml -n $(NAMESPACE) --context $(PROD_CTX)
	kubectl rollout status deployment/taskservice -n $(NAMESPACE) --context $(PROD_CTX)

## Deploy to Local Kind (from ghcr.io registry)
deploy-staging:
	@echo "Pulling $(REGISTRY):$(VERSION) from ghcr.io..."
	docker pull $(REGISTRY):$(VERSION)
	docker tag $(REGISTRY):$(VERSION) $(DOCKER_IMAGE):$(VERSION)
	kind load docker-image $(DOCKER_IMAGE):$(VERSION) --name staging
	kubectl set image deployment/taskservice taskservice=$(DOCKER_IMAGE):$(VERSION) \
		-n $(NAMESPACE) --context $(STAGING_CTX) 2>/dev/null || \
		kubectl apply -f k8s/local-deployment.yaml -n $(NAMESPACE) --context $(STAGING_CTX)
	kubectl rollout status deployment/taskservice -n $(NAMESPACE) --context $(STAGING_CTX)
	@echo "✅ Deployed $(VERSION) to staging"

deploy-prod:
	@echo "Pulling $(REGISTRY):$(VERSION) from ghcr.io..."
	docker pull $(REGISTRY):$(VERSION)
	docker tag $(REGISTRY):$(VERSION) $(DOCKER_IMAGE):$(VERSION)
	kind load docker-image $(DOCKER_IMAGE):$(VERSION) --name prod
	kubectl set image deployment/taskservice taskservice=$(DOCKER_IMAGE):$(VERSION) \
		-n $(NAMESPACE) --context $(PROD_CTX) 2>/dev/null || \
		kubectl apply -f k8s/local-deployment.yaml -n $(NAMESPACE) --context $(PROD_CTX)
	kubectl rollout status deployment/taskservice -n $(NAMESPACE) --context $(PROD_CTX)
	@echo "✅ Deployed $(VERSION) to prod"

## Port forward to access services
port-forward-staging:
	kubectl port-forward svc/taskservice 8081:80 -n $(NAMESPACE) --context $(STAGING_CTX)

port-forward-prod:
	kubectl port-forward svc/taskservice 9081:80 -n $(NAMESPACE) --context $(PROD_CTX)

## Status
status:
	@echo "=== STAGING ==="
	@kubectl get pods -n $(NAMESPACE) --context $(STAGING_CTX)
	@echo ""
	@echo "=== PROD ==="
	@kubectl get pods -n $(NAMESPACE) --context $(PROD_CTX)

## Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  test               - Run tests with coverage"
	@echo "  lint               - Run linter"
	@echo "  run                - Run locally"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-compose-up  - Run with docker-compose"
	@echo "  clean              - Clean build artifacts"
	@echo ""
	@echo "Kind Cluster Management:"
	@echo "  kind-create        - Create staging + prod Kind clusters"
	@echo "  kind-delete        - Delete Kind clusters"
	@echo "  status             - Show pod status in both clusters"
	@echo ""
	@echo "Deployment (local build):"
	@echo "  deploy-local-staging - Build and deploy to staging"
	@echo "  deploy-local-prod    - Build and deploy to prod"
	@echo ""
	@echo "Deployment (from ghcr.io):"
	@echo "  deploy-staging VERSION=<sha> - Pull from registry, deploy to staging"
	@echo "  deploy-prod VERSION=<sha>    - Pull from registry, deploy to prod"
	@echo ""
	@echo "Access:"
	@echo "  port-forward-staging - Forward staging to localhost:8081"
	@echo "  port-forward-prod    - Forward prod to localhost:9081"
