# Backend Services

This directory contains the Go microservices and jobs for the Personal Dashboard.

## Overview

The backend is built as a set of decoupled services communicating via Firestore (async) or gRPC (sync).

### Shared Architecture
*   **Language:** Go 1.25+
*   **Database:** Google Cloud Firestore (NoSQL).
*   **API Protocol:** REST (Public Aggregator) and gRPC (Internal Services).
*   **Contract Management:** **Buf** with Distributed Generation (code lives within each service).
*   **Logging:** Structured JSON logging (`slog`).

---

## Service Reference

### 1. Dashboard API (`services/dashboard-api`)
**Type:** Cloud Run Service (HTTP/REST)
**Role:** Backend-for-Frontend (BFF). Aggregates data from multiple internal services.
*   **Architecture:** [ARCHITECTURE_SERVICE_DASHBOARD_API.md](../docs/ARCHITECTURE_SERVICE_DASHBOARD_API.md)

### 2. Weather Provider (`services/weather-provider`)
**Type:** Cloud Run Service (gRPC)
**Role:** Serves analyzed weather/pressure statistics from Firestore.
*   **Architecture:** [ARCHITECTURE_SERVICE_WEATHER_PROVIDER.md](../docs/ARCHITECTURE_SERVICE_WEATHER_PROVIDER.md)

### 3. Weather Collector (`services/weather-collector`)
**Type:** Cloud Run Job (Batch)
**Role:** Periodically fetches raw weather data from external APIs.
*   **Architecture:** [ARCHITECTURE_SERVICE_WEATHER_COLLECTOR.md](../docs/ARCHITECTURE_SERVICE_WEATHER_COLLECTOR.md)

### 4. Pollen Provider (`services/pollen-provider`)
**Type:** Cloud Run Service (gRPC)
**Role:** Serves pollen count and allergy risk data from Firestore.
*   **Architecture:** [ARCHITECTURE_SERVICE_POLLEN.md](../docs/ARCHITECTURE_SERVICE_POLLEN.md)

### 5. Pollen Collector (`services/pollen-collector`)
**Type:** Cloud Run Job (Batch)
**Role:** Fetches pollen data from the Google Pollen API twice daily.
*   **Architecture:** [ARCHITECTURE_SERVICE_POLLEN.md](../docs/ARCHITECTURE_SERVICE_POLLEN.md)

---

## gRPC Contracts (Buf)

We use a **Distributed Generation** strategy. Protos are defined centrally, but generated code lives inside the consuming services.

**To update contracts:**
1.  Edit files in `services/protos/`.
2.  Run `make proto` from the repository root.
3.  Commit the changed `.proto` and `.pb.go` files.

---

## Local Development Guide

### 1. Prerequisites
*   **Go**: v1.25+
*   **Buf**: For contract management.
*   **Docker**: For production-like local testing.

### 2. Authentication
Run `gcloud auth application-default login` to allow local services to access Firestore.

### 3. Running Services
Use the root `Makefile` targets:

| Service | Dev (Go) | Prod-like (Docker) |
| :--- | :--- | :--- |
| **Dashboard API** | `make da-dev` | `make da-build` |
| **Weather Provider** | `make wp-dev` | `make wp-build` |
| **Weather Collector** | `make wc-dev` | `make wc-build` |
| **Pollen Provider** | `make pp-dev` | `make pp-build` |
| **Pollen Collector** | `make pc-dev` | `make pc-build` |

### 4. Port Reference
| Service | Port | Protocol |
| :--- | :--- | :--- |
| **Dashboard API** | `8080` | HTTP/REST |
| **Weather Provider** | `50051` | gRPC |
| **Pollen Provider** | `50052` | gRPC |
