# Personal Dashboard Monorepo

This repository contains the frontend dashboard and backend services for the Personal Dashboard project.

## Project Structure

- **`frontend/`**: The primary user interface built with **SvelteKit**.
- **`services/`**: Backend microservices and jobs (primarily **Go**).
  - **`weather-collector`**: A Cloud Run Job that fetches weather data.
  - **`weather-provider`**: A gRPC Service that serves weather data.
- **`infra/`**: Terraform configuration for GCP infrastructure.

## Getting Started

### Prerequisites
1.  **Go**: v1.25+
2.  **Node.js**: v20+
3.  **Docker**: For running containerized services locally.
4.  **Google Cloud SDK (`gcloud`)**: For authentication and deployment.

### Authentication (Crucial)
Backend services require access to Google Cloud Firestore. For local development, use **Application Default Credentials (ADC)**.

Run this command once on your machine:
```bash
gcloud auth application-default login
```
This creates a local credential file that Docker containers will mount to authenticate.

### Quick Start

**1. Frontend**
```bash
make dev-frontend
# Opens at http://localhost:5173
```

**2. Backend Services**
See [services/README.md](./services/README.md) for detailed configuration.

*   **Native Go (Fastest):**
    ```bash
    make dev-provider   # Starts the gRPC server
    make dev-collector  # Runs the collector job once
    ```
*   **Docker (Production-like):**
    ```bash
    make docker-run-provider
    make docker-run-collector
    ```

## Documentation
*   [Backend Services Documentation](./services/README.md)
*   [Infrastructure Architecture](./docs/ARCHITECTURE_INFRASTRUCTURE.md)