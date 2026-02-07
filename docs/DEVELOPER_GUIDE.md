# Developer Guide

## Makefiles

Our `Makefile` acts as the entry point for common development tasks.

*   **Default Target:** The `help` target MUST always be the first target defined. This ensures that running `make` without arguments displays the help menu rather than executing a destructive or long-running command.
*   **Self-Documentation:** All targets should be documented with a `## Description` comment on the same line. The `help` target parses these comments to generate the menu.

## gRPC & API Contracts

### 1. Structure
*   **`/services/protos` (The Source of Truth):** Contains raw `.proto` files. This is the centralized registry for all API contracts.
*   **Service-Local `gen/` Directories:** Each service (e.g., `/services/weather-provider/gen`) contains its own copy of the generated Go code.

### 2. Checking in Generated Code
**We commit all generated Go code to Git.** 
*   **Why:** This ensures the project can be built without requiring `protoc` to be installed on every machine (including CI/CD).
*   **Docker Compatibility:** By keeping the generated code *inside* the service folder, each microservice is a self-contained unit. This allows us to run `docker build` from within the service directory, which is required for our "Bootstrap + CD" deployment pattern in GCP.

### 3. Generating Code
We use a centralized `Makefile` at the root to handle code generation. It reads from the shared `/services/protos` registry and writes to the specific service's `gen/` folder.

```bash
make proto
```

### 4. Usage in Services
Services should import the generated code from their own internal `gen` package.

```go
import "github.com/nickfang/personal-dashboard/services/weather-provider/gen/v1"
```

## Development Workflow

We follow a **Trunk-Based Development** workflow with strict CI checks.

### 1. Branching Strategy
*   **Main Branch (`main`):** Represents the production state. Direct pushes should be disabled.
*   **Feature Branches:** Create a new branch for every task (e.g., `feat/add-retry-logic` or `fix/timestamp-bug`).

### 2. The Lifecycle
1.  **Code:** Work locally. Use `go run main.go` or `make` commands.
2.  **Test:** Run unit tests locally before pushing:
    ```bash
    cd services/weather-collector
    go test -v ./...
    ```
3.  **Push:** Push your feature branch to GitHub.
4.  **Verify (CI):** Open a Pull Request. GitHub Actions (`verify-*.yml`) will automatically run tests and build checks. You cannot merge if this fails.
5.  **Deploy (CD):** Merge the PR into `main`. GitHub Actions (`deploy-*.yml`) will build the Docker image and update Cloud Run automatically.

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
