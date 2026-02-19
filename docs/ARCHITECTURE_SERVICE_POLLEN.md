# Pollen Provider Architecture (Future)

## 1. Overview
The **Pollen Provider** is a planned microservice subsystem responsible for collecting and serving daily pollen count and allergy risk data. It follows the same architectural pattern as the Weather subsystem but operates on a much lower frequency (daily vs. hourly).

## 2. Requirements

### Functional Requirements
*   **Collection:** Fetch pollen data from external APIs (e.g., AccuWeather, Pollen.com) once per day.
*   **Storage:** Store historical pollen counts in Firestore.
*   **Serving:** Expose the latest pollen data via gRPC for the Dashboard API.

### Technical Constraints
*   **Frequency:** Daily execution (e.g., 6:00 AM local time).
*   **Data Structure:** Unlike pressure (continuous), pollen is discrete (Low, Med, High) or numeric index.

## 3. Architecture & Data Flow

```mermaid
sequenceDiagram
    participant Scheduler as Cloud Scheduler (Daily)
    participant Collector as Pollen Collector (Job)
    participant Store as Firestore (pollen_history)
    participant API as Dashboard API
    
    Scheduler->>Collector: Trigger Job
    Collector->>ExternalAPI: Fetch Pollen
    Collector->>Store: Save Daily Record
    
    API->>Store: (via Provider) GetLatestPollen()
    API-->>Frontend: {"pollen": { "level": "High", "index": 9.2 }}
```

## 4. Implementation Strategy

### Components
1.  **Pollen Collector (Job):**
    *   A Go binary running in Google Cloud Run (Job).
    *   Triggered by Cloud Scheduler.
    *   Responsible for fetching and normalizing external data.

2.  **Pollen Provider (Server):**
    *   A gRPC service that serves the collected pollen data.
    *   **Decision:** This may be implemented as a standalone service or as an extension of the `weather-provider` if the domain logic remains simple.

### Infrastructure Changes
*   **Terraform:**
    *   New `google_cloud_scheduler_job` resource for daily execution.
    *   New `google_cloud_run_v2_job` resource for the collector container.
    *   IAM roles for the new service account.

### Dependency Management
*   **Contract First:** Managed via Buf in `services/protos`.
*   **Local Contracts:** Following the **Distributed Generation** strategy, this service will generate its own copy of gRPC stubs in `internal/gen/go`.
*   **Build Strategy:** Self-contained Docker builds from the service directory context.

## 5. Decision Log
*   **Why separate from Weather Collector?**
    *   Different frequency (Daily vs Hourly).
    *   Different failure modes (Pollen API might be down while Weather is up).
    *   Separation of Concerns: Maintain modularity for easier debugging.
*   **Naming Convention:** Renamed from `pollen-service` to `pollen-provider` to align with the `weather-provider` pattern.
