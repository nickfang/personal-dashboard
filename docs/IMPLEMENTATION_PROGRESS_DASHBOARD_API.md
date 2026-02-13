# Implementation Progress: Dashboard API

This document tracks the lifecycle of the Dashboard API service implementation.

## Status Summary
- **Target Platform:** Google Cloud Run (Service)
- **Framework:** Go 1.25 + Chi + gRPC
- **Current Phase:** Phase 2: Dashboard API Scaffolding

---

## Roadmap & Progress

### Phase 1: Shared Library Refactor (The "Enabler")
*Goal: Centralize gRPC code in `services/gen/go` using Buf (v2).*

**1. Create the Verification Test (TDD Start)**
- [x] Create `services/gen/go/verification_test.go` that imports the future package.
- [x] Verify it fails (compilation error) because the code doesn't exist yet.

**2. Configure Buf (The Tooling)**
- [x] **Initialize Module:** In `services/protos`, ensure `buf.yaml` (v2) exists.
    ```yaml
    version: v2
    name: buf.build/nickfang/personal-dashboard
    lint: { use: [STANDARD] }
    breaking: { use: [FILE] }
    ```
- [x] **Configure Generation:** Create `services/buf.gen.yaml` (v2).
    ```yaml
    version: v2
    plugins:
      - local: protoc-gen-go
        out: gen/go
        opt: paths=source_relative
      - local: protoc-gen-go-grpc
        out: gen/go
        opt: paths=source_relative
    ```

**3. Execute Generation**
- [x] **Run Buf:** From the `services/` directory:
    ```bash
    cd services
    buf generate protos
    ```
    *   *Check:* Do you see `.pb.go` files in `services/gen/go`?

**4. Initialize the Go Module**
- [x] **Create go.mod:** The generated code needs to be a package.
    ```bash
    cd services/gen/go
    go mod init github.com/nickfang/personal-dashboard/services/gen/go
    go mod tidy
    ```
- [x] **Verify TDD:** Run `go test ./services/gen/go`. It should now pass!

**5. Integrate with Existing Services**
- [x] **Refactor Makefile:** Update the `proto` target to run `cd services && buf generate`.
- [x] **Update Weather Provider:**
    *   Go to `services/weather-provider`.
    *   Run `go get github.com/nickfang/personal-dashboard/services/gen/go`.
    *   Update imports in `main.go` and `handler.go`.
- [x] **Update Docker:** Ensure `weather-provider/Dockerfile` builds from the root context (as documented in Architecture).

### Phase 2: Dashboard API Scaffolding
*Goal: Create the service skeleton.*

**1. Create Directory Structure**
- [x] **Create folders:**
    ```bash
    mkdir -p services/dashboard-api/cmd/server
    mkdir -p services/dashboard-api/internal/handlers
    mkdir -p services/dashboard-api/internal/middleware
    mkdir -p services/dashboard-api/internal/clients
    ```

**2. Initialize Go Module**
- [x] **Init Module:**
    ```bash
    cd services/dashboard-api
    go mod init github.com/nickfang/personal-dashboard/services/dashboard-api
    ```
- [x] **Add to Workspace:**
    ```bash
    # From root
    go work use ./services/dashboard-api
    ```

**3. Basic Server Implementation**
- [x] **Create main.go:** Create `services/dashboard-api/cmd/server/main.go` with a simple "Hello World" or Health Check.
    ```go
    package main
    import (
        "net/http"
        "github.com/go-chi/chi/v5"
    )
    func main() {
        r := chi.NewRouter()
        r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Dashboard API is healthy"))
        })
        http.ListenAndServe(":8080", r)
    }
    ```
- [x] **Install Deps:** Run `go mod tidy` in the service folder.

**4. Verify Local Execution**
- [x] **Run:** `go run services/dashboard-api/cmd/server/main.go`
- [x] **Test:** `curl localhost:8080/health`

### Phase 3: Core Implementation
*Goal: Aggregation and gRPC logic.*

**1. Weather Client (gRPC)**
- [x] **TDD:** Create `internal/clients/weather_client_test.go` (Ready).
- [x] **Implement:** `WeatherClient` in `internal/clients/weather-client.go`.
- [x] **Verify:** Run `go test ./services/dashboard-api/internal/clients/...` (Green).

**2. Dashboard Handler (REST)**
- [x] **TDD:** Create `internal/handlers/handler_test.go` (Ready).
- [x] **Refactor Handler:** Implemented `GetDashboard` returning `{"pressure": ...}` raw data.
- [x] **Update Test:** Adjusted test to verify nested JSON structure.

**3. Router Setup (Structure)**
- [x] **Create Router Package:** Created `services/dashboard-api/internal/app/router.go`.
- [x] **Register Routes:** Added `/api/v1/dashboard`.
- [x] **Middleware:** Added `Recoverer`, `RequestID`, and custom `SlogLogger`.

**4. Wiring (Main)**
- [x] **Update Main:** Moved `main.go` to root. Wired Clients, Handlers, and Router.

**5. Middleware & Security**
- [ ] Implement Cognito JWT validation middleware in `internal/middleware/auth.go`.
- [ ] Handle CORS for Svelte frontend.

### Phase 4: Infrastructure & Deployment
*Goal: Go live.*

**1. Dockerize**
- [ ] Create `services/dashboard-api/Dockerfile` (Build from Root context).
- [ ] Verify local build: `docker build -f services/dashboard-api/Dockerfile .`

**2. Terraform (Infrastructure)**
- [ ] Create `infra/dashboard_api.tf`.
- [ ] Define Cloud Run Service (`dashboard-api`).
- [ ] Create Service Account (`dashboard-api-sa`).
- [ ] Grant Invoke permission to `weather-provider`.

**3. CI/CD (GitHub Actions)**
- [ ] Create `.github/workflows/deploy-dashboard-api.yml`.
- [ ] Implement "Build & Push" step.
- [ ] Implement "Deploy to Cloud Run" step.

**4. Verification**
- [ ] Deploy to GCP.
- [ ] `curl` the public URL.



### Phase 4: Infrastructure & Deployment
*Goal: Go live.*
- [ ] Create `infra/dashboard_api.tf` (Terraform).
- [ ] Configure IAM (Allow `dashboard-api-sa` to call `weather-provider-sa`).
- [ ] Create `.github/workflows/deploy-dashboard-api.yml`.
- [ ] Verify end-to-end flow from Cloud Shell.

---

## Context Notes
- **Naming:** Service is named `dashboard-api`.
- **Port:** HTTP server listens on 8080 (Cloud Run default).
- **gRPC Port:** Internal services listen on 50051.
- **Docker:** Must run `docker build` from root using `-f services/dashboard-api/Dockerfile`.
