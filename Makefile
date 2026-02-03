.PHONY: help dev-frontend dev-weather proto clean-proto

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-20s %s\n", $$1, $$2}'

# gRPC Configuration
PROTO_SRC_DIR := services/protos
GEN_GO_DIR := services/gen/go

# Find all proto files
PROTO_FILES := $(shell find $(PROTO_SRC_DIR) -name "*.proto")
# Map them to their expected generated output (e.g., .pb.go)
GEN_GO_FILES := $(patsubst $(PROTO_SRC_DIR)/%.proto, $(GEN_GO_DIR)/%.pb.go, $(PROTO_FILES))

proto: $(GEN_GO_FILES) ## Generate Go code from all .proto files (Smart build)

# Pattern rule: Generate .pb.go from .proto
$(GEN_GO_DIR)/%.pb.go: $(PROTO_SRC_DIR)/%.proto
	@mkdir -p $(dir $@)
	protoc --proto_path=$(PROTO_SRC_DIR) \
		--go_out=$(GEN_GO_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_GO_DIR) --go-grpc_opt=paths=source_relative \
		$<
	@echo "  → Generated: $@"

clean-proto: ## Remove all generated proto files
	rm -rf $(GEN_GO_DIR)/*

dev-frontend: ## Run the Svelte frontend
	cd frontend && npm run dev

dev-weather: ## Run the Weather Collector service
	cd services/weather-collector && \
	export $$(grep -v '^#' .env | xargs) && \
	go run main.go
