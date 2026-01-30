# Developer Guide

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
