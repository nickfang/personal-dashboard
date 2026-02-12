# Backend Services

This directory contains the Go microservices and jobs for the Personal Dashboard.

## Overview

The backend is built as a set of decoupled services communicating via Firestore (async) or gRPC (sync).

### Shared Architecture
*   **Language:** Go 1.25+
*   **Database:** Google Cloud Firestore (NoSQL).
*   **Logging:** Structured JSON logging (`slog`) for Cloud Logging compatibility.
*   **Configuration:** 12-Factor App methodology (Environment Variables).
*   **Local Development:** Supports both native execution (`go run`) and Docker containers with local credential mounting.

---

## Service Reference

### 1. Weather Collector (`services/weather-collector`)
**Type:** Cloud Run Job (Batch)
**Description:** Fetches weather data from Google Weather API and writes to Firestore.

**Configuration (`.env`):**
| Variable | Description | Example |
| :--- | :--- | :--- |
| `GCP_PROJECT_ID` | Google Cloud Project ID | `fang-gcp` |
| `GOOGLE_MAPS_API_KEY` | API Key for Google Weather/Maps | `AIzaSy...` |
| `DEBUG` | Enable verbose logging | `true` |

**Links:**
*   [Architecture Doc](../../docs/ARCHITECTURE_SERVICE_WEATHER_COLLECTOR.md)

### 2. Weather Provider (`services/weather-provider`)
**Type:** Cloud Run Service (gRPC)
**Description:** Serves aggregated weather history and pressure analysis to the frontend.

**Configuration (`.env`):**
| Variable | Description | Example |
| :--- | :--- | :--- |
| `GCP_PROJECT_ID` | Google Cloud Project ID | `fang-gcp` |
| `PORT` | gRPC Server Port (Default: 50051) | `50051` |
| `DEBUG` | Enable verbose logging | `true` |

**Links:**
*   [Architecture Doc](../../docs/ARCHITECTURE_SERVICE_WEATHER_PROVIDER.md)

---

## Local Development Guide

### 1. Prerequisites
*   **Go**: Version 1.25 or later.
*   **Docker**: For running containerized services locally.
*   **Google Cloud SDK (`gcloud`)**: Required for authentication.
*   **Buf**: For gRPC management.
    ```bash
    brew install bufbuild/buf/buf
    ```

### 2. Authentication
Run `gcloud auth application-default login` on your host machine. This creates the credentials needed for Firestore access.

### 3. Configuration
Copy the example files and fill in your values:

```bash
# Collector
cp services/weather-collector/.env.example services/weather-collector/.env

# Provider
cp services/weather-provider/.env.example services/weather-provider/.env
```

### 4. Running with Make
Use the root `Makefile` to run services.

**Native Execution (Go):**
```bash
make dev-collector
make dev-provider
```

**Docker Execution:**
When running via `make docker-run-...`, the Makefile automatically mounts your local `~/.config/gcloud` directory into the container. This allows the container to authenticate as YOU, without needing Service Account keys.
```bash
make docker-run-collector
make docker-run-provider
```
