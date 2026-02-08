# Weather Provider Service Architecture

## 1. Overview
The **Weather Provider** (`services/weather-provider`) is a high-performance gRPC service responsible for serving weather data to the Personal Dashboard.

Unlike the *Collector* (which is a write-heavy background job), the *Provider* is a read-only service that acts as the source of truth for weather data displayed on the frontend. It abstracts the underlying storage implementation (Firestore) from the API consumers.

## 2. Requirements

### Functional Requirements
*   **GetWeatherHistory**: Retrieve the current conditions and 24-48h history for a specific location.
*   **Data Transformation**: Map the internal Firestore schema (e.g., `WeatherPoint`) to the public API Protobuf definition.
*   **Error Handling**: Return appropriate gRPC error codes (e.g., `NOT_FOUND` if a location doesn't exist).

### Non-Functional Requirements
*   **Latency**: Responses should be fast (<200ms) as this powers the main dashboard view.
*   **Statelessness**: The service should be horizontally scalable (Cloud Run).
*   **Testability**: Core logic must be unit-testable without requiring a live Firestore connection.
*   **Observability**: Structured logging (slog) and error reporting.

## 3. Data Flow
1.  **Request**: Consumer (Aggregator or Frontend via Gateway) sends `GetWeatherRequest(location_id)`.
2.  **Lookup**: Service queries the `weather_cache` Firestore collection for the document matching `location_id`.
3.  **Response**: Service maps the Firestore document to the `GetWeatherResponse` Proto message and returns it.

## 4. Implementation Strategy: Standard Layered Architecture

We will utilize a **Standard Layered Architecture**. This approach isolates the Transport (gRPC) from the Business Logic and Data Access, making the code modular and easy to test.

### Folder Structure
```text
services/weather-provider/
├── cmd/
│   └── server/          # Application Entry Point
│       └── main.go      # Wires layers together, handles Graceful Shutdown
├── gen/                 # Generated Go Code (Service-Local)
│   └── v1/              # Server stubs and message types
├── internal/
│   ├── repository/      # Data Access Layer
│   │   ├── firestore.go # Implementation (with Query Limits)
│   │   └── reader.go    # Interface definition
│   ├── service/         # Business Logic Layer
│   │   ├── weather.go   # Implementation
│   │   └── service.go   # Interface definition
│   └── transport/       # Transport Layer
│       └── handler.go   # Implements the gRPC interface
├── Dockerfile           # Production container (Alpine)
└── go.mod               # Service module
```

### Layer Responsibilities
1.  **Transport (`internal/transport`)**:
    *   **Input**: Receives `GetWeatherRequest` (imported from `gen/v1`).
    *   **Action**: Validates input, calls the Service layer.
    *   **Output**: Maps domain models to `GetWeatherResponse` (Proto).

2.  **Service (`internal/service`)**:
    *   **Responsibility**: Pure business logic. Orchestrates data retrieval.
    *   **Independence**: Agnostic of Firestore or gRPC.

3.  **Repository (`internal/repository`)**:
    *   **Responsibility**: Firestore interactions.
    *   **Safeguards**: Limits queries to 100 documents to prevent OOM. Logs invalid documents without failing the request.

4.  **Main (`cmd/server`)**:
    *   **Configuration**: Loads `.env` via `godotenv`.
    *   **Lifecycle**: Implements Graceful Shutdown (SIGTERM/SIGINT) to finish in-flight requests.
    *   **Health**: Registers standard gRPC Health Checks for Cloud Run probes.

## 5. Local Development

We use a consolidated **Makefile** in the project root to manage all services.

### Prerequisites
*   `gcloud auth application-default login` (Required for Firestore access)

### Commands
*   **Native Go**: `make dev-provider` (Fastest)
*   **Docker**: `make docker-run-provider` (Mounts local GCP credentials)

See `services/README.md` for detailed configuration.

## 6. Testing Strategy (TDD)
(Retain existing testing content...)

## 7. Infrastructure (Terraform)
The service is deployed as a **private Cloud Run Service**.

### Resource Configuration
*   **Service Account:** `weather-provider-sa` (Dedicated identity).
*   **Permissions:** `roles/datastore.viewer` (Read-Only access to Firestore).
*   **Networking:** Private ingress (internal traffic only).
*   **Protocol:** HTTP/2 (Required for gRPC).
*   **Port:** 50051 (Exposed via Dockerfile).

### Bootstrap Strategy
We use a "Bootstrap + CD" pattern in Terraform (`infra/weather_provider.tf`):
1.  **Bootstrap:** A `null_resource` builds and pushes the initial image (`gcloud builds submit`).
2.  **Service Definition:** `google_cloud_run_v2_service` defines the runtime config.
3.  **Lifecycle Ignore:** Terraform ignores changes to `image` and `client_version` to allow CI/CD to manage day-to-day deployments without interference.

## 8. Continuous Deployment (GitHub Actions)
The pipeline is defined in `.github/workflows/deploy-weather-provider.yml`.

### Triggers
*   **Push:** To `main` branch.
*   **Paths:** `services/weather-provider/**` (Only deploys when provider code changes).

### Pipeline Steps
1.  **Auth:** Workload Identity Federation (WIF).
2.  **Build:** Standard `docker build` (leveraging the internalized `gen/` code).
3.  **Push:** Push to Artifact Registry with Git SHA tag.
4.  **Deploy:** `gcloud run services update` pointing to the specific SHA.

## 9. Verification
Since the service is private, verification requires proxying:

```bash
# 1. Start Proxy
gcloud run services proxy weather-provider --port=50051

# 2. Test with grpcurl
grpcurl -plaintext -d '{"location_id": "house-nick"}' localhost:50051 weather_provider.v1.PressureStatsService/GetPressureStats
```
