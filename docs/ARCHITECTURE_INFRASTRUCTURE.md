# Infrastructure & Platform Architecture

## 1. Overview
The Personal Dashboard platform is hosted on Google Cloud Platform (GCP) and managed via Terraform ("Infrastructure as Code"). It supports a monorepo structure with multiple microservices.

## 2. Core Infrastructure (Foundation)
Defined in `infra/main.tf` and `infra/github_oidc.tf`.

*   **Artifact Registry:** A shared Docker repository (`personal-dashboard`) stores images for all services.
*   **Identity & Access Management (IAM):**
    *   **Workload Identity Federation (WIF):** Enables GitHub Actions to authenticate securely without long-lived keys.
    *   **Service Accounts:**
        *   `github-actions-sa`: Used by CI/CD pipelines to build/push images and deploy services.
        *   `weather-collector-sa`: Runtime identity for the Weather Collector service.

## 3. Deployment Strategy (Bootstrap + CD)
We utilize a hybrid pattern to support both Disaster Recovery (DR) and fast Continuous Deployment (CD).

### Terraform (The Stage & Bootstrap)
*   **Role**: Manages infrastructure (IAM, APIs, Schedule, Memory limits).
*   **Behavior**:
    *   On a fresh install (DR), Terraform ensures a "Bootstrap" image exists.
    *   It creates Cloud Run Jobs/Services pointing to this image.
    *   **Crucial Config**: Resources use `lifecycle { ignore_changes = [image] }`. This tells Terraform to **not revert** the image version if it detects a change.

### GitHub Actions (The Actor)
*   **Role**: Manages day-to-day code deployments.
*   **Behavior**:
    *   **CI (Verify):** On Pull Request, runs unit tests (`go test`) and build checks.
    *   **CD (Deploy):** On push to `main`, builds the new Docker image (tagged with `git sha`), pushes to Artifact Registry, and updates the Cloud Run resource.

## 4. Data Layer
Defined in `infra/firestore.tf`.

*   **Firestore (Native Mode):** The primary database.
*   **Database ID:** `weather-log` (Note: separate from the `(default)` database).
*   **Access Pattern:** Services connect using the Google Cloud Go SDK, authenticated via their runtime Service Account.

## 5. Development Workflow
*   **Local:** Developers use `go run` or `make` commands.
*   **Testing:** Automated CI workflows (`verify-*.yml`) run on every Pull Request.
*   **Production:** Automated CD workflows (`deploy-*.yml`) run on merge to `main`.
