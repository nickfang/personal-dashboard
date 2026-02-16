# Implementation Progress: Dashboard API

This document tracks the lifecycle of the Dashboard API service implementation.

## Status Summary
- **Target Platform:** Google Cloud Run (Service)
- **Framework:** Go 1.25 + Chi + gRPC
- **Current Phase:** Phase 4: Infrastructure & Deployment

---

## Roadmap & Progress

### Phase 1: Shared Library Refactor (The "Enabler")
*Goal: Centralize gRPC code in `services/gen/go` using Buf (v2).*
- [x] **TDD:** Create `services/gen/go/verification_test.go`.
- [x] **Configure Buf:** Setup `buf.yaml` and `buf.gen.yaml`.
- [x] **Execute:** Generated code into `services/gen/go`.
- [x] **Initialize:** Created `go.mod` for the shared library.

### Phase 2: Dashboard API Scaffolding
*Goal: Create the service skeleton.*
- [x] **Create Directory Structure:** `internal/app`, `internal/handlers`, `internal/middleware`, `internal/clients`.
- [x] **Initialize Go Module:** `go mod init github.com/nickfang/personal-dashboard/services/dashboard-api`.
- [x] **Add to Workspace:** `go work use ./services/dashboard-api`.

### Phase 3: Core Implementation
*Goal: Aggregation and gRPC logic.*
- [x] **Weather Client:** Implemented gRPC client using `services/gen/go` (initially).
- [x] **Dashboard Handler:** Implemented `GetDashboard` with aggregation logic.
- [x] **Router Setup:** Created `internal/app/router.go` with `chi` and `slog` middleware.
- [x] **Wiring:** Updated `cmd/server/main.go` to connect all components.

### Phase 3.5: Refactor - Distributed Generation
*Goal: Decouple services by moving code generation into each service directory.*

**1. Buf Configuration**
- [x] Create `services/weather-provider/buf.gen.yaml`.
- [x] Create `services/dashboard-api/buf.gen.yaml`.
- [x] Delete `services/buf.gen.yaml`.

**2. Code Generation**
- [x] Update `Makefile` to run generation for each service.
- [x] Run `make proto`.
- [x] Delete `services/gen/go`.

**3. Code Updates (Imports)**
- [x] Update `weather-provider` imports (`main.go`, `handler.go`).
- [x] Update `dashboard-api` imports (`handler.go`, `client.go`, tests).
- [x] Remove `replace` from `dashboard-api/go.mod` (using local module resolution).

**4. Infrastructure Fix**
- [x] Update `weather-provider/Dockerfile` (Self-contained, remove root COPY).
- [x] Update `dashboard-api/Dockerfile` (Self-contained, remove root COPY).

### Phase 4: Infrastructure & Deployment
*Goal: Go live.*

**1. Dockerize**
- [x] Create `services/dashboard-api/Dockerfile`.
- [ ] Verify local build: `docker build -t dashboard-api services/dashboard-api`.

**2. Terraform (Infrastructure)**
- [ ] Create `infra/dashboard_api.tf`.
- [ ] Define Cloud Run Service (`dashboard-api`).
- [ ] Create Service Account (`dashboard-api-sa`).
- [ ] Grant Invoke permission to `weather-provider`.

**3. CI/CD (GitHub Actions)**
- [ ] Create `.github/workflows/deploy-dashboard-api.yml`.

**4. Verification**
- [ ] Deploy to GCP.
- [ ] `curl` the public URL.

---

## Context Notes
- **Naming:** Service is named `dashboard-api`.
- **Strategy:** Distributed Generation (each service owns its code).
- **Docker:** Must run `docker build services/dashboard-api` (isolated context).
