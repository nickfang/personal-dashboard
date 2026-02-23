# Developer Guide

## Makefiles

Our `Makefile` acts as the entry point for common development tasks.

*   **Default Target:** The `help` target MUST always be the first target defined. This ensures that running `make` without arguments displays the help menu rather than executing a destructive or long-running command.
*   **Self-Documentation:** All targets should be documented with a `## Description` comment on the same line. The `help` target parses these comments to generate the menu.

## Getting Started

### Prerequisites
1.  **Go**: v1.25+
2.  **Node.js**: v20+
3.  **Docker**: For running containerized services locally.
4.  **Google Cloud SDK (`gcloud`)**: For authentication and deployment.
5.  **Buf**: For gRPC linting and code generation.

## gRPC & API Contracts

### 1. Structure
*   **`/services/protos` (The Source of Truth):** Contains raw `.proto` files managed by **Buf**.
*   **`/services/<service>/internal/gen` (Local Generation):** Each service generates its own copy of the gRPC code it needs. Each service has its own `buf.gen.yaml` that controls output paths.

### 2. Checking in Generated Code
**We commit all generated Go code to Git.**
*   **Why:** This ensures the project can be built without requiring `buf` to be installed on every machine (including CI/CD).
*   **Decoupling:** Each service is self-contained. `dashboard-api` doesn't crash if `weather-provider` breaks its own build.

### 3. Generating Code
We use **Buf** with distributed targets.

```bash
# From the repository root
make proto  # Runs 'buf generate' for ALL services
```

## Development Workflow

We follow a **Trunk-Based Development** workflow with strict CI checks.

### 1. Branching Strategy
*   **Main Branch (`main`):** Represents the production state. Direct pushes should be disabled.
*   **Feature Branches:** Create a new branch for every task (e.g., `feat/add-retry-logic` or `fix/timestamp-bug`).

### 2. The Lifecycle
1.  **Auth (Once):** `gcloud auth application-default login`
2.  **Code:** Work locally.
    *   `make dev-provider`
    *   `make dev-collector`
    
    **Dashboard API (Aggregator)**
    Runs on port **8080**.
    ```bash
    cd services/dashboard-api
    go run main.go
    ```
    *   **Dependencies:** Requires `weather-provider` running on port 50051 (or `WEATHER_PROVIDER_ADDR` set) and `pollen-provider` running on port 50052 (or `POLLEN_PROVIDER_ADDR` set).

3.  **Test:** Run unit tests locally before pushing:
    ```bash
    # Run all tests (Frontend + Backend)
    make test
    
    # Run specific service tests
    cd services/weather-collector && go test -v ./...
    ```
    *   **Note:** `make test` runs `go test ./...` in each service directory, avoiding root-level module conflicts.

4.  **Push:** Push your feature branch to GitHub.
5.  **Verify (CI):** Open a Pull Request. GitHub Actions (`verify-*.yml`) will automatically run tests and build checks. You cannot merge if this fails.
6.  **Deploy (CD):** Merge the PR into `main`. GitHub Actions (`deploy-*.yml`) will build the Docker image and update Cloud Run automatically.

## Infrastructure & Deployment

We use a **Hybrid "Bootstrap + CD" Pattern** to manage our cloud resources.

*   **Terraform (The Stage):** Manages the "hard" infrastructure (IAM, Networking, Databases, Service definitions).
    *   *Note:* Terraform configures the Cloud Run Job but is instructed to **ignore** changes to the container image version (`lifecycle { ignore_changes = [image] }`).
*   **GitHub Actions (The Actor):** Manages the application code.
    *   Every deploy updates the Cloud Run Job to a specific Docker tag (the git commit SHA).

**Why?** This allows Terraform to restore the environment from scratch (Disaster Recovery) without fighting against the day-to-day deployments managed by CI/CD.

## Commenting Philosophy

When writing or reviewing code, adhere to the following principle regarding documentation and comments:

**Document the *Why*, not the *What*.**

- **Avoid Redundancy:** Do not explain *what* the code is doing (e.g., `i++ // increment i`). The code itself should be readable enough to convey the "what".
- **Provide Context:** Use comments to explain the reasoning, trade-offs, and architectural decisions behind a specific implementation.
- **Explain the "Why":** Why was this specific library chosen? Why is there a pointer here instead of a value? Why is this constant set to 45 minutes?
- **Future-Proofing:** Write comments that help a future developer (including your future self) understand the intent and constraints that led to the current design.
