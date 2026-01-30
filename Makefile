.PHONY: dev-frontend dev-weather

# Run the Svelte frontend
dev-frontend:
	cd frontend && npm run dev

# Run the Weather Collector service
# Loads the .env file from the service directory before running
dev-weather:
	cd services/weather-collector && \
	export $$(grep -v '^#' .env | xargs) && \
	go run main.go
