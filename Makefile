.PHONY: help \
	compose-up compose-down compose-logs \
	proto-gen proto-clean test-go \
	wc-dev wc-build wc-run wc-test \
	wp-dev wp-build wp-test \
	da-dev da-build da-test \
	fe-dev fe-test \

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$|^##@' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; /^##@/{printf "\n\033[1m%s\033[0m\n", substr($$0, 4); next} {printf "  make %-20s %s\n", $$1, $$2}'

# ==============================================================================
# Global
# ==============================================================================
##@ Global
compose-up: ## Start all services via Docker Compose
	docker compose up --build

compose-down: ## Stop all services
	docker compose down

compose-logs: ## View logs for all services
	docker compose logs -f

test-go: ## Run all Go tests
	@find services -name "go.mod" -exec dirname {} \; | while read dir; do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./...); \
	done

##@ Proto
## Note: We use the --path flag to specify the path to the proto files.
## Clients need multiple paths to get all the client protos.
proto: ## Generate Go code for all services via Buf
	cd services/weather-provider && buf generate ../protos --path ../protos/weather-provider
	cd services/pollen-provider && buf generate ../protos --path ../protos/pollen-provider
	cd services/dashboard-api && buf generate ../protos \
		--path ../protos/weather-provider \
		--path ../protos/pollen-provider

proto-clean: ## Remove all generated proto files
	rm -rf services/weather-provider/internal/gen/go/*
	rm -rf services/pollen-provider/internal/gen/go/*
	rm -rf services/dashboard-api/internal/gen/go/*

proto-align-versions:
	cd services/weather-collector && go get -u google.golang.org/grpc
	cd services/weather-provider && go get -u google.golang.org/grpc
	cd services/dashboard-api && go get -u google.golang.org/grpc
	cd services/pollen-collector && go get -u google.golang.org/grpc
	cd services/pollen-provider && go get -u google.golang.org/grpc

# ==============================================================================
# Service: Weather Collector (Job)
# ==============================================================================
##@ Weather Collector
wc-dev: ## Run Collector locally (Go)
	-cd services/weather-collector && go run main.go

wc-build: ## Build Weather Collector image
	docker build -t weather-collector -f services/weather-collector/Dockerfile services

wc-run: wc-build ## Run Weather Collector container (One-off job)
	docker run --rm -it \
		--env-file services/weather-collector/.env \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		weather-collector

wc-test: ## Run Weather Collector tests
	cd services/weather-collector && go test ./...

# ==============================================================================
# Service: Weather Provider (Server)
# ==============================================================================
##@ Weather Provider
wp-dev: ## Run Weather Provider locally (Go)
	-cd services/weather-provider && go run cmd/server/main.go

wp-build: ## Build Weather Provider image
	docker build -t weather-provider -f services/weather-provider/Dockerfile services

wp-test: ## Run Weather Provider tests
	cd services/weather-provider && go test ./...

# ==============================================================================
# Service: Pollen Collector (Job)
# ==============================================================================
##@ Pollen Collector
pc-dev: ## Run Pollen Collector locally (Go)
	-cd services/pollen-collector && go run main.go

pc-build: ## Build Pollen Collector image
	docker build -t pollen-collector -f services/pollen-collector/Dockerfile services

pc-run: pc-build ## Run Pollen Collector container (One-off job)
	docker run --rm -it \
		--env-file services/pollen-collector/.env \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		pollen-collector

pc-test: ## Run Pollen Collector tests
	cd services/pollen-collector && go test ./...

# ==============================================================================
# Service: Pollen Provider (Server)
# ==============================================================================
##@ Pollen Provider
pp-dev: ## Run Pollen Provider locally (Go)
	-cd services/pollen-provider && go run cmd/server/main.go

pp-build: ## Build Pollen Provider image
	docker build -t pollen-provider -f services/pollen-provider/Dockerfile services

pp-test: ## Run Pollen Provider tests
	cd services/pollen-provider && go test ./...

# ==============================================================================
# Service: Dashboard API (Aggregator)
# ==============================================================================
##@ Dashboard API
da-dev: ## Run Dashboard API locally (Go)
	-cd services/dashboard-api && go run cmd/server/main.go

da-build: ## Build Dashboard API image
	docker build -t dashboard-api -f services/dashboard-api/Dockerfile services

da-test: ## Run Dashboard API tests
	cd services/dashboard-api && go test ./...


# ==============================================================================
# Frontend
# ==============================================================================
##@ Frontend
fe-dev: ## Run the Svelte frontend
	cd frontend && npm run dev

fe-test: ## Run the Svelte frontend tests
	cd frontend && npm test

# ==============================================================================
# Utilities
# ==============================================================================
##@ Utilities
util-proto-align-versions: ## Align the versions of the proto packages
	make proto-align-versions