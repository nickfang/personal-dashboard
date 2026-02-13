.PHONY: help proto clean-proto test \
	dev-frontend \
	dev-collector docker-build-collector docker-run-collector \
	dev-provider docker-build-provider docker-run-provider \
	dev-dashboard docker-build-dashboard docker-run-dashboard

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-20s %s\n", $$1, $$2}'

# ==============================================================================
# Frontend
# ==============================================================================

dev-frontend: ## Run the Svelte frontend
	cd frontend && npm run dev

# ==============================================================================
# Services
# ==============================================================================

proto: ## Generate Go code for all services via Buf
	cd services && buf generate protos

clean-proto: ## Remove all generated proto files
	rm -rf services/gen/go/*

test: ## Run all Go tests
	@find services -name "go.mod" -exec dirname {} \; | while read dir; do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./...); \
	done

# ==============================================================================
# Service: Weather Collector (Job)
# ==============================================================================

dev-collector: ## Run Collector locally (Go)
	-cd services/weather-collector && go run main.go

docker-build-collector: ## Build Collector image
	docker build -t weather-collector services/weather-collector

docker-run-collector: docker-build-collector ## Run Collector container (One-off job)
	docker run --rm -it \
		--env-file services/weather-collector/.env \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		weather-collector

# ==============================================================================
# Service: Weather Provider (Server)
# ==============================================================================

dev-provider: ## Run Provider locally (Go)
	-cd services/weather-provider && go run cmd/server/main.go

docker-build-provider: ## Build Provider image
	docker build -t weather-provider -f services/weather-provider/Dockerfile .

docker-run-provider: docker-build-provider ## Run Provider container (Port 50051)
	docker run --rm -it \
		--env-file services/weather-provider/.env \
		-p 50051:50051 \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		weather-provider

# ==============================================================================
# Service: Dashboard API (Aggregator)
# ==============================================================================

dev-dashboard: ## Run Dashboard API locally (Go)
	-cd services/dashboard-api && go run cmd/server/main.go

docker-build-dashboard: ## Build Dashboard image
	docker build -t dashboard-api -f services/dashboard-api/Dockerfile .

docker-run-dashboard: docker-build-dashboard ## Run Dashboard container (Port 8080)
	docker run --rm -it \
		--env-file services/dashboard-api/.env \
		-p 8080:8080 \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		dashboard-api
