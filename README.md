# Personal Dashboard Monorepo

This repository contains the frontend dashboard and backend services for the Personal Dashboard project.

## Project Structure

- **`frontend/`**: The primary user interface built with **SvelteKit**.
- **`services/`**: Backend microservices and jobs (primarily **Go**).
  - **`weather-collector`**: A Cloud Run Job that fetches weather data.
  - **`weather-provider`**: A gRPC Service that serves weather data.
  - **`dashboard-api`**: An HTTP Aggregator (BFF) that talks to internal gRPC services.
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

## Running Locally

There are two primary ways to run the backend services locally. **Both require `.env` files to be present in each service folder.**

### 1. Configuration (.env files)
Each service in `services/` contains a `.env.example` file. 
- Copy `.env.example` to `.env` in each service directory.
- Fill in the required variables (e.g., `GCP_PROJECT_ID`).
- **Note:** `.env` files are ignored by Git.

### 2. Docker Compose (Full Stack - Recommended)
The easiest way to run the entire backend with networking pre-configured.
```bash
make up
```
- **Dashboard API:** http://localhost:8080/api/v1/dashboard
- **Weather Provider:** localhost:50051 (gRPC)
- **Stop services:** `make down`

### 3. Native Go (Individual Development)
Useful for rapid iteration on a single service.
```bash
make dev-provider   # Runs weather-provider
make dev-dashboard  # Runs dashboard-api
```
- **Constraint:** When running natively, services use `localhost` to communicate. Ensure your `.env` files reflect this.

### 4. Frontend
```bash
make dev-frontend
# Opens at http://localhost:5173
```

## Testing
```bash
make test
```
This command runs tests across all backend services and the shared library.

## Documentation
*   [Developer Guide (Workflows, gRPC, Testing)](./docs/DEVELOPER_GUIDE.md)
*   [Dashboard API Implementation](./docs/IMPLEMENTATION_PROGRESS_DASHBOARD_API.md)
*   [Infrastructure Architecture](./docs/ARCHITECTURE_INFRASTRUCTURE.md)
