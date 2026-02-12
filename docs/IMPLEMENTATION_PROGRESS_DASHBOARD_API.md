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
- [x] **Configure Generation:** Create `services/buf.gen.yaml` (v2).

**3. Execute Generation**
- [x] **Run Buf:** From the `services/` directory: `buf generate protos`.
- [x] **Verify:** Check `.pb.go` files in `services/gen/go`.

**4. Initialize the Go Module**
- [x] **Create go.mod:** Initialized with `github.com/nickfang/personal-dashboard/services/gen/go`.
- [x] **Verify TDD:** Run `go test ./services/gen/go`. Passes!

**5. Integrate with Existing Services**
- [x] **Refactor Makefile:** Update the `proto` target to use Buf.
- [x] **Update Weather Provider:** Imports now point to shared library.
- [x] **Update Docker:** `weather-provider/Dockerfile` now builds from root context.
- [x] **Update go.work:** Included `services/gen/go`.

### Phase 2: Dashboard API Scaffolding
*Goal: Create the service skeleton.*
- [ ] Create `services/dashboard-api` directory.
- [ ] Initialize `go.mod`.
- [ ] Create `cmd/server/main.go` with basic Chi router.
- [ ] Create basic health check endpoint (`GET /health`).

### Phase 3: Core Implementation
*Goal: Aggregation and gRPC logic.*
- [ ] Implement `WeatherClient` in `internal/clients/`.
- [ ] Implement parallel fetch logic using `errgroup`.
- [ ] Implement `GET /api/v1/dashboard` handler.
- [ ] Implement Cognito JWT validation middleware.
- [ ] Handle CORS for Svelte frontend.

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
