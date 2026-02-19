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
proto: ## Generate Go code for all services via Buf
	cd services/weather-provider && buf generate ../protos
	cd services/dashboard-api && buf generate ../protos

proto-clean: ## Remove all generated proto files
	rm -rf services/weather-provider/internal/gen/go/*
	rm -rf services/dashboard-api/internal/gen/go/*

# ==============================================================================
# Service: Weather Collector (Job)
# ==============================================================================
##@ Weather Collector
wc-dev: ## Run Collector locally (Go)
	-cd services/weather-collector && go run main.go

wc-build: ## Build Weather Collector image
	docker build -t weather-collector services/weather-collector

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
	docker build -t weather-provider services/weather-provider

wp-test: ## Run Weather Provider tests
	cd services/weather-provider && go test ./...

# ==============================================================================
# Service: Dashboard API (Aggregator)
# ==============================================================================
##@ Dashboard API
da-dev: ## Run Dashboard API locally (Go)
	-cd services/dashboard-api && go run cmd/server/main.go

da-build: ## Build Dashboard API image
	docker build -t dashboard-api services/dashboard-api

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