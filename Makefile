.PHONY: help proto clean-proto test \
	dev-frontend \
	dev-collector docker-build-collector docker-run-collector \
	dev-provider docker-build-provider docker-run-provider

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-20s %s\n", $$1, $$2}'

# ==============================================================================
# Global
# ==============================================================================

# gRPC Configuration
PROTO_SRC_DIR := services/protos
# This maps the central contract to the local service implementation folder
WEATHER_PROVIDER_DIR := services/protos/weather-provider/v1
WEATHER_PROVIDER_PROTO := weather_provider.proto
WEATHER_PROVIDER_OUT := services/weather-provider/gen/v1

proto: proto-weather-provider ## Generate Go code for all services

proto-weather-provider: ## Generate Go code for the weather-provider service
	@mkdir -p $(WEATHER_PROVIDER_OUT)
	protoc --proto_path=$(WEATHER_PROVIDER_DIR) \
		--proto_path=$(PROTO_SRC_DIR) \
		--go_out=$(WEATHER_PROVIDER_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(WEATHER_PROVIDER_OUT) --go-grpc_opt=paths=source_relative \
		$(WEATHER_PROVIDER_PROTO)
	@echo "  → Generated: Weather Provider (Server stubs)"

clean-proto: ## Remove all generated proto files from service directories
	rm -rf $(WEATHER_PROVIDER_OUT)/*.pb.go

test: ## Run all Go tests
	@find services -name "go.mod" -exec dirname {} \; | while read dir; do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./...); \
	done

# ==============================================================================
# Frontend
# ==============================================================================

dev-frontend: ## Run the Svelte frontend
	cd frontend && npm run dev

# ==============================================================================
# Service: Weather Collector (Job)
# ==============================================================================

dev-collector: ## Run Collector locally (Go)
	cd services/weather-collector && \
	export $$(grep -v '^#' .env | xargs) && \
	go run main.go

docker-build-collector: ## Build Collector image
	docker build -t weather-collector services/weather-collector

docker-run-collector: docker-build-collector ## Run Collector container (One-off job)
	docker run --rm -it \
		--env-file services/weather-collector/.env \
		weather-collector

# ==============================================================================
# Service: Weather Provider (Server)
# ==============================================================================

dev-provider: ## Run Provider locally (Go)
	cd services/weather-provider && \
	export $$(grep -v '^#' .env | xargs) && \
	go run cmd/server/main.go

docker-build-provider: ## Build Provider image
	docker build -t weather-provider services/weather-provider

docker-run-provider: docker-build-provider ## Run Provider container (Port 50051)
	docker run --rm -it \
		--env-file services/weather-provider/.env \
		-p 50051:50051 \
		weather-provider
