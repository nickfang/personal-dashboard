# Pollen Provider — Implementation Guide

This document is the step-by-step implementation plan for the Pollen subsystem. It was produced from an architectural review session where every decision was explicitly made. A junior engineer should be able to follow this guide with no ambiguity.

For platform-level context (Terraform patterns, CI/CD, Firestore), see [ARCHITECTURE_INFRASTRUCTURE.md](./ARCHITECTURE_INFRASTRUCTURE.md).
For the original high-level design, see [ARCHITECTURE_SERVICE_POLLEN.md](./ARCHITECTURE_SERVICE_POLLEN.md).

---

## Decision Log (Quick Reference)

| # | Area | Decision |
|---|---|---|
| 1 | External API | Google Pollen API (reuses `GOOGLE_MAPS_API_KEY`) |
| 2 | Data depth | 3 pollen types + all plant-level UPI values, no health recs |
| 3 | Overall summary | Computed at write time by collector |
| 4 | Firestore strategy | Dual-write: `pollen_raw` + `pollen_cache` |
| 5 | Database | Same `weather-log` database |
| 6 | Cache history | 14-day rolling window (28 entries at 2x/day) |
| 7 | Service architecture | Standalone `pollen-provider` gRPC (port 50052) |
| 8 | Proto contract | `PollenService` with `GetAllPollenReports` + `GetPollenReport` |
| 9 | Locations | All 3 locations fetched separately |
| 10 | Schedule | 2x/day at 6:00 AM + 2:00 PM Central |
| 11 | Dashboard API | Separate `"pollen"` key alongside `"pressure"` |
| 12 | Infrastructure | Separate Cloud Run Job for pollen-collector |
| 13 | Shared code | Local `services/shared/` Go module (locations, constants, logging) |

---

## Phase 0: Shared Module

**Goal:** Create the `services/shared/` Go module that all services will import for locations, constants, and logging setup. This phase is done first because subsequent phases depend on these imports.

### Step 0.1: Initialize the Shared Module

```bash
mkdir -p services/shared
cd services/shared
go mod init github.com/nickfang/personal-dashboard/services/shared
```

### Step 0.2: Create `locations.go`

Single source of truth for all monitored locations. Both collectors import this instead of defining their own location slices.

```go
package shared

// Location represents a monitored geographic point.
type Location struct {
	ID   string
	Lat  float64
	Long float64
}

// Locations is the canonical list used by all collector services.
var Locations = []Location{
	{ID: "house-nick", Lat: 30.260543381977474, Long: -97.66768538740229},
	{ID: "house-nita", Lat: 30.29420179895202, Long: -97.6958691874014},
	{ID: "distribution-hall", Lat: 30.261932944618565, Long: -97.72816923158192},
}
```

### Step 0.3: Create `constants.go`

Database ID and cache collection names shared between collectors (writers) and providers (readers).

```go
package shared

const (
	// DatabaseID is the Firestore database used by all services.
	DatabaseID = "weather-log"

	// Cache collection names — shared between each collector/provider pair.
	WeatherCacheCollection = "weather_cache"
	PollenCacheCollection  = "pollen_cache"
)
```

**Why collection names are here:** The `_cache` collection name is a contract between each collector and its provider. A typo or drift in either side causes a silent failure (writer writes to one name, reader reads from another). The `_raw` collection names stay local to each collector since no other service references them.

### Step 0.4: Create `logging.go`

Standardized structured logging setup. Ensures every service uses JSON output to stdout (required for GCP Cloud Run) and supports the `DEBUG` env var toggle.

```go
package shared

import (
	"log/slog"
	"os"
)

// InitLogging configures the global slog logger with JSON output.
// Set DEBUG=true env var for debug-level logging.
func InitLogging() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}
```

### Step 0.5: Update `go.work`

```
go 1.25.6

use (
    ./services/dashboard-api
    ./services/pollen-collector
    ./services/pollen-provider
    ./services/shared
    ./services/weather-collector
    ./services/weather-provider
)
```

### Step 0.6: Verify

```bash
# From repo root — go.work resolves the shared module automatically
cd services/shared && go build ./...
```

---

## Phase 1: Proto Contract

**Goal:** Define the gRPC contract in `services/protos`, then generate code into the pollen-provider and dashboard-api services.

### Step 1.1: Create the Proto File

Create `services/protos/pollen-provider/v1/pollen_provider.proto`:

```protobuf
syntax = "proto3";

package pollen_provider.v1;

import "google/protobuf/timestamp.proto";

option go_package = "pollen_provider/v1";

service PollenService {
  rpc GetAllPollenReports(GetAllPollenReportsRequest) returns (GetAllPollenReportsResponse);
  rpc GetPollenReport(GetPollenReportRequest) returns (GetPollenReportResponse);
}

message PollenReport {
  string location_id = 1;
  google.protobuf.Timestamp collected_at = 2;

  // Overall summary (computed by collector at write time).
  // Represents the highest UPI across the 3 pollen types.
  int32 overall_index = 3;
  string overall_category = 4;
  string dominant_type = 5;

  repeated PollenType types = 6;
  repeated PollenPlant plants = 7;
}

message PollenType {
  string code = 1;
  int32 index = 2;
  string category = 3;
  bool in_season = 4;
}

message PollenPlant {
  string code = 1;
  string display_name = 2;
  int32 index = 3;
  string category = 4;
  bool in_season = 5;
}

message GetAllPollenReportsRequest {}

message GetAllPollenReportsResponse {
  repeated PollenReport reports = 1;
}

message GetPollenReportRequest {
  string location_id = 1;
}

message GetPollenReportResponse {
  PollenReport report = 1;
}
```

**Why these choices:**
*   `int32` for index (not `double`) — the Universal Pollen Index is a discrete 0–5 integer.
*   `overall_*` fields precomputed by the collector — the dashboard API reads one field instead of scanning arrays.
*   `PollenType` and `PollenPlant` are separate message types so the frontend can render them in different components.

### Step 1.2: Create buf.gen.yaml for Pollen Provider

Create `services/pollen-provider/buf.gen.yaml`:

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package
      value: github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1
plugins:
  - local: protoc-gen-go
    out: internal/gen/go
    opt: paths=source_relative
  - local: protoc-gen-go-grpc
    out: internal/gen/go
    opt: paths=source_relative
```

### Step 1.3: Update buf.gen.yaml for Dashboard API

The dashboard-api also needs to generate pollen stubs. Update `services/dashboard-api/buf.gen.yaml`:

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package
      # Weather stubs
      path: weather-provider/v1/weather_provider.proto
      value: github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1
    - file_option: go_package
      # Pollen stubs
      path: pollen-provider/v1/pollen_provider.proto
      value: github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1
plugins:
  - local: protoc-gen-go
    out: internal/gen/go
    opt: paths=source_relative
  - local: protoc-gen-go-grpc
    out: internal/gen/go
    opt: paths=source_relative
```

**Note:** The dashboard-api uses explicit `path:` matchers on its overrides because it generates stubs for multiple proto packages, and each needs a different `go_package`. Single-service configs (weather-provider, pollen-provider) don't need `path:` matchers since they only generate one proto package.

### Step 1.4: Generate Code

Proto generation is scoped per service using the `--path` flag. This prevents Buf from generating stubs for *all* protos in the `protos/` module into every service. The `--path` flag filters which proto subdirectory is generated, while the `buf.gen.yaml` in each service controls the output location and `go_package` mapping.

```bash
# From repository root (or use `make proto`)
cd services/weather-provider && buf generate ../protos --path ../protos/weather-provider
cd services/pollen-provider && buf generate ../protos --path ../protos/pollen-provider
cd services/dashboard-api && buf generate ../protos
```

**Why `--path` instead of `inputs` in `buf.gen.yaml`?** The `protos/` directory is a single Buf module (defined by `protos/buf.yaml`). Buf does not allow subdirectories of a module to be used as standalone `inputs`. The `--path` flag filters within the module at generation time. For a future service that needs a subset of protos, add multiple `--path` flags (e.g., `--path ../protos/pollen-provider --path ../protos/air-quality-provider`).

Update the `Makefile` proto target (see Phase 6).

### Step 1.5: Commit Generated Code

Per the Developer Guide, we commit generated code to Git so the project builds without requiring `buf`.

---

## Phase 2: Pollen Collector (Cloud Run Job)

**Goal:** Create a Go binary that fetches pollen data from the Google Pollen API and writes to Firestore.

### Directory Structure

```text
services/pollen-collector/
├── main.go          # Single-file implementation (matches weather-collector pattern)
├── main_test.go     # Unit tests
├── go.mod
├── Dockerfile
└── .env.example
```

**Why single-file?** The weather-collector is a run-once job with no transport layer. Layered architecture (transport/service/repository) is unnecessary for a batch job that runs and exits. Keeping it in one file follows the established pattern.

### Step 2.1: Initialize Go Module

```bash
cd services/pollen-collector
go mod init github.com/nickfang/personal-dashboard/services/pollen-collector
```

Add the shared module dependency to `go.mod`:

```
require github.com/nickfang/personal-dashboard/services/shared v0.0.0

replace github.com/nickfang/personal-dashboard/services/shared => ../shared
```

The `replace` directive tells Go to resolve the shared module from the local filesystem instead of a module proxy. This is required for Docker builds, which don't use `go.work`. During local development, `go.work` provides its own module resolution (via the `use` block), so `replace` is technically redundant locally — but it must stay in `go.mod` for container builds to work.

### Step 2.2: Create .env.example

```env
GCP_PROJECT_ID=your-project-id
GOOGLE_MAPS_API_KEY=your-api-key
DEBUG=true
```

### Step 2.3: Implement main.go

The collector has these responsibilities:
1.  Load config and initialize Firestore.
2.  Loop through all 3 locations.
3.  Fetch pollen data from Google Pollen API (with retry).
4.  Compute the overall summary (highest UPI).
5.  Dual-write: append to `pollen_raw`, update `pollen_cache`.

#### Constants & Types

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "time"

    "cloud.google.com/go/firestore"
    "github.com/joho/godotenv"
    "github.com/nickfang/personal-dashboard/services/shared"
)

const (
    MAX_HISTORY_POINTS    = 28 // 14 days × 2 readings/day
    POLLEN_RAW_COLLECTION = "pollen_raw"
)

// Locations come from shared.Locations (see Phase 0).
// Database ID comes from shared.DatabaseID.
// Cache collection name comes from shared.PollenCacheCollection.
// POLLEN_RAW_COLLECTION stays local — no other service reads from it.
```

#### Google Pollen API Response Types

These structs map directly to the Google Pollen API `forecast:lookup` JSON response. Only the fields we need are included (Go's `json` decoder ignores unmatched fields).

```go
type PollenAPIResponse struct {
    DailyInfo []DailyInfo `json:"dailyInfo"`
}

type DailyInfo struct {
    Date           APIDate          `json:"date"`
    PollenTypeInfo []PollenTypeInfo `json:"pollenTypeInfo"`
    PlantInfo      []PlantInfo      `json:"plantInfo"`
}

type APIDate struct {
    Year  int `json:"year"`
    Month int `json:"month"`
    Day   int `json:"day"`
}

type PollenTypeInfo struct {
    Code        string    `json:"code"`
    DisplayName string    `json:"displayName"`
    InSeason    bool      `json:"inSeason"`
    IndexInfo   IndexInfo `json:"indexInfo"`
}

type PlantInfo struct {
    Code        string    `json:"code"`
    DisplayName string    `json:"displayName"`
    InSeason    bool      `json:"inSeason"`
    IndexInfo   IndexInfo `json:"indexInfo"`
}

type IndexInfo struct {
    Value    int    `json:"value"`
    Category string `json:"category"`
}
```

#### Firestore Models

These are the shapes written to Firestore. They are independent of the API response types (separation of external API vs internal storage).

```go
type StoredPollenType struct {
    Code     string `firestore:"code"`
    Index    int    `firestore:"index"`
    Category string `firestore:"category"`
    InSeason bool   `firestore:"in_season"`
}

type StoredPollenPlant struct {
    Code        string `firestore:"code"`
    DisplayName string `firestore:"display_name"`
    Index       int    `firestore:"index"`
    Category    string `firestore:"category"`
    InSeason    bool   `firestore:"in_season"`
}

// PollenSnapshot is a single reading. Used in both pollen_raw docs and the
// pollen_cache history array.
type PollenSnapshot struct {
    LocationID      string              `firestore:"location_id"`
    CollectedAt     time.Time           `firestore:"collected_at"`
    OverallIndex    int                 `firestore:"overall_index"`
    OverallCategory string              `firestore:"overall_category"`
    DominantType    string              `firestore:"dominant_type"`
    Types           []StoredPollenType  `firestore:"types"`
    Plants          []StoredPollenPlant `firestore:"plants"`
}

// PollenCacheDoc is the shape of a pollen_cache document (one per location).
type PollenCacheDoc struct {
    LastUpdated time.Time        `firestore:"last_updated"`
    Current     PollenSnapshot   `firestore:"current"`
    History     []PollenSnapshot `firestore:"history"`
}
```

#### API Fetch with Retry

Matches the weather-collector retry pattern (3 attempts, exponential backoff).

```go
func fetchPollenWithRetry(apiKey string, loc shared.Location) (*PollenAPIResponse, error) {
    var lastErr error
    backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

    for i := 0; i <= len(backoffs); i++ {
        data, err := fetchPollen(apiKey, loc)
        if err == nil {
            return data, nil
        }
        lastErr = err
        if i < len(backoffs) {
            time.Sleep(backoffs[i])
        }
    }
    return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

func fetchPollen(apiKey string, loc shared.Location) (*PollenAPIResponse, error) {
    url := fmt.Sprintf(
        "https://pollen.googleapis.com/v1/forecast:lookup?key=%s&location.latitude=%f&location.longitude=%f&days=1",
        apiKey, loc.Lat, loc.Long,
    )

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Pollen API returned status: %s", resp.Status)
    }

    var data PollenAPIResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("failed to decode pollen JSON: %w", err)
    }

    if len(data.DailyInfo) == 0 {
        return nil, fmt.Errorf("no daily info returned for %s", loc.ID)
    }

    return &data, nil
}
```

#### Mapping API Response to Firestore Models

This function converts the external API shape to our internal storage shape and computes the overall summary.

```go
func mapToSnapshot(locationID string, apiResp *PollenAPIResponse) PollenSnapshot {
    today := apiResp.DailyInfo[0]

    snapshot := PollenSnapshot{
        LocationID:  locationID,
        CollectedAt: time.Now(),
    }

    // Map pollen types
    for _, t := range today.PollenTypeInfo {
        snapshot.Types = append(snapshot.Types, StoredPollenType{
            Code:     t.Code,
            Index:    t.IndexInfo.Value,
            Category: t.IndexInfo.Category,
            InSeason: t.InSeason,
        })
    }

    // Map plants
    for _, p := range today.PlantInfo {
        snapshot.Plants = append(snapshot.Plants, StoredPollenPlant{
            Code:        p.Code,
            DisplayName: p.DisplayName,
            Index:       p.IndexInfo.Value,
            Category:    p.IndexInfo.Category,
            InSeason:    p.InSeason,
        })
    }

    // Compute overall summary: find the highest UPI across the 3 types
    for _, t := range snapshot.Types {
        if t.Index > snapshot.OverallIndex {
            snapshot.OverallIndex = t.Index
            snapshot.OverallCategory = t.Category
            snapshot.DominantType = t.Code
        }
    }

    return snapshot
}
```

#### Dual-Write Logic

```go
func saveRawPollenData(ctx context.Context, client *firestore.Client, snapshot PollenSnapshot) error {
    _, _, err := client.Collection(POLLEN_RAW_COLLECTION).Add(ctx, snapshot)
    return err
}

func updatePollenCache(ctx context.Context, client *firestore.Client, locationID string, snapshot PollenSnapshot) error {
    cacheRef := client.Collection(shared.PollenCacheCollection).Doc(locationID)

    return client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
        doc, err := tx.Get(cacheRef)
        var cache PollenCacheDoc
        if err == nil {
            if err := doc.DataTo(&cache); err != nil {
                return err
            }
        } else {
            cache = PollenCacheDoc{History: []PollenSnapshot{}}
        }

        cache.History = append(cache.History, snapshot)
        if len(cache.History) > MAX_HISTORY_POINTS {
            cache.History = cache.History[len(cache.History)-MAX_HISTORY_POINTS:]
        }

        cache.LastUpdated = snapshot.CollectedAt
        cache.Current = snapshot

        return tx.Set(cacheRef, cache)
    })
}
```

#### Main Function

```go
func main() {
    shared.InitLogging()

    if err := godotenv.Load(); err != nil {
        slog.Debug("No .env file found, using system environment variables", "error", err)
    }

    ctx := context.Background()
    apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
    projectID := os.Getenv("GCP_PROJECT_ID")

    if apiKey == "" || projectID == "" {
        slog.Error("Missing required env vars", "vars", "GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
        os.Exit(1)
    }

    client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.DatabaseID)
    if err != nil {
        slog.Error("Failed to create Firestore client", "error", err)
        os.Exit(1)
    }
    defer client.Close()

    for _, loc := range shared.Locations {
        apiResp, err := fetchPollenWithRetry(apiKey, loc)
        if err != nil {
            slog.Error("Failed to fetch pollen after retries",
                "location", loc.ID,
                "error", err,
            )
            continue
        }

        snapshot := mapToSnapshot(loc.ID, apiResp)

        if err := saveRawPollenData(ctx, client, snapshot); err != nil {
            slog.Error("Error saving raw pollen data", "location", loc.ID, "error", err)
            continue
        }

        if err := updatePollenCache(ctx, client, loc.ID, snapshot); err != nil {
            slog.Error("Error updating pollen cache", "location", loc.ID, "error", err)
            continue
        }

        slog.Info("Processed pollen", "location", loc.ID, "overall_index", snapshot.OverallIndex, "dominant", snapshot.DominantType)
    }
}
```

### Step 2.4: Create Dockerfile

The Docker build context is `services/` (not `services/pollen-collector/`) so the shared module is available for `go mod tidy`.

```dockerfile
# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY pollen-collector/ ./pollen-collector/

WORKDIR /app/pollen-collector
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/pollen-collector/bin main.go

# Stage 2: Final image
FROM alpine:3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/pollen-collector/bin ./pollen-collector

CMD ["./pollen-collector"]
```

**Note:** The Dockerfile path is still `services/pollen-collector/Dockerfile`, but it's built with `-f` from the `services/` context (see Phase 6 for docker-compose and Makefile changes).

### Step 2.5: Unit Tests (main_test.go)

Test the pure functions (mapping, overall computation) without Firestore.

```go
package main

import "testing"

func TestMapToSnapshot_OverallSummary(t *testing.T) {
    apiResp := &PollenAPIResponse{
        DailyInfo: []DailyInfo{{
            PollenTypeInfo: []PollenTypeInfo{
                {Code: "GRASS", InSeason: true, IndexInfo: IndexInfo{Value: 2, Category: "Low"}},
                {Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
                {Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
            },
            PlantInfo: []PlantInfo{
                {Code: "JUNIPER", DisplayName: "Juniper", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
                {Code: "OAK", DisplayName: "Oak", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
            },
        }},
    }

    snapshot := mapToSnapshot("house-nick", apiResp)

    if snapshot.OverallIndex != 4 {
        t.Errorf("expected OverallIndex 4, got %d", snapshot.OverallIndex)
    }
    if snapshot.OverallCategory != "High" {
        t.Errorf("expected OverallCategory 'High', got %s", snapshot.OverallCategory)
    }
    if snapshot.DominantType != "TREE" {
        t.Errorf("expected DominantType 'TREE', got %s", snapshot.DominantType)
    }
    if len(snapshot.Types) != 3 {
        t.Errorf("expected 3 types, got %d", len(snapshot.Types))
    }
    if len(snapshot.Plants) != 2 {
        t.Errorf("expected 2 plants, got %d", len(snapshot.Plants))
    }
}

func TestMapToSnapshot_AllZero(t *testing.T) {
    apiResp := &PollenAPIResponse{
        DailyInfo: []DailyInfo{{
            PollenTypeInfo: []PollenTypeInfo{
                {Code: "GRASS", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
                {Code: "TREE", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
                {Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
            },
        }},
    }

    snapshot := mapToSnapshot("house-nick", apiResp)

    if snapshot.OverallIndex != 0 {
        t.Errorf("expected OverallIndex 0, got %d", snapshot.OverallIndex)
    }
}
```

---

## Phase 3: Pollen Provider (gRPC Service)

**Goal:** Create a read-only gRPC service that serves pollen data from `pollen_cache`.

### Directory Structure

```text
services/pollen-provider/
├── cmd/
│   └── server/
│       └── main.go          # Entry point (wires layers, graceful shutdown)
├── internal/
│   ├── gen/go/               # Generated gRPC stubs (from Phase 1)
│   ├── repository/
│   │   ├── firestore.go      # Firestore implementation
│   │   └── reader.go         # Interface definition
│   ├── service/
│   │   ├── pollen.go         # Business logic
│   │   └── pollen_test.go    # Unit tests with mock repo
│   └── transport/
│       ├── handler.go        # gRPC handler (maps domain → proto)
│       └── handler_test.go   # Handler tests
├── buf.gen.yaml              # (Created in Phase 1)
├── go.mod
├── Dockerfile
└── .env.example
```

### Step 3.1: Initialize Go Module

```bash
cd services/pollen-provider
go mod init github.com/nickfang/personal-dashboard/services/pollen-provider
```

Add the shared module dependency to `go.mod`:

```
require github.com/nickfang/personal-dashboard/services/shared v0.0.0

replace github.com/nickfang/personal-dashboard/services/shared => ../shared
```

### Step 3.2: Create .env.example

```env
GCP_PROJECT_ID=your-project-id
PORT=50052
DEBUG=true
```

### Step 3.3: Repository Layer

**`internal/repository/reader.go`** — Interface definition:

```go
package repository

import "context"

type PollenReader interface {
    GetAll(ctx context.Context) ([]CacheDoc, error)
    GetByID(ctx context.Context, id string) (*CacheDoc, error)
}
```

**`internal/repository/firestore.go`** — Implementation:

```go
package repository

import (
    "context"
    "log/slog"
    "time"

    "cloud.google.com/go/firestore"
    "github.com/nickfang/personal-dashboard/services/shared"
    "google.golang.org/api/iterator"
)

// Firestore models — match the shapes written by pollen-collector.
type StoredPollenType struct {
    Code     string `firestore:"code"`
    Index    int    `firestore:"index"`
    Category string `firestore:"category"`
    InSeason bool   `firestore:"in_season"`
}

type StoredPollenPlant struct {
    Code        string `firestore:"code"`
    DisplayName string `firestore:"display_name"`
    Index       int    `firestore:"index"`
    Category    string `firestore:"category"`
    InSeason    bool   `firestore:"in_season"`
}

type PollenSnapshot struct {
    LocationID      string              `firestore:"location_id"`
    CollectedAt     time.Time           `firestore:"collected_at"`
    OverallIndex    int                 `firestore:"overall_index"`
    OverallCategory string              `firestore:"overall_category"`
    DominantType    string              `firestore:"dominant_type"`
    Types           []StoredPollenType  `firestore:"types"`
    Plants          []StoredPollenPlant `firestore:"plants"`
}

type CacheDoc struct {
    LocationID  string         `firestore:"-"`
    LastUpdated time.Time      `firestore:"last_updated"`
    Current     PollenSnapshot `firestore:"current"`
}

type FirestoreRepository struct {
    client *firestore.Client
}

func NewFirestoreRepository(ctx context.Context, projectID string) (*FirestoreRepository, error) {
    client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.DatabaseID)
    if err != nil {
        return nil, err
    }
    return &FirestoreRepository{client: client}, nil
}

func (r *FirestoreRepository) Close() error {
    return r.client.Close()
}

func (r *FirestoreRepository) GetAll(ctx context.Context) ([]CacheDoc, error) {
    var results []CacheDoc
    iter := r.client.Collection(shared.PollenCacheCollection).Limit(100).Documents(ctx)
    defer iter.Stop()

    for {
        doc, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, err
        }

        var cache CacheDoc
        if err := doc.DataTo(&cache); err != nil {
            slog.Warn("Skipping invalid document in GetAll", "doc_id", doc.Ref.ID, "error", err)
            continue
        }
        cache.LocationID = doc.Ref.ID
        results = append(results, cache)
    }

    return results, nil
}

func (r *FirestoreRepository) GetByID(ctx context.Context, id string) (*CacheDoc, error) {
    doc, err := r.client.Collection(shared.PollenCacheCollection).Doc(id).Get(ctx)
    if err != nil {
        return nil, err
    }

    var cache CacheDoc
    if err := doc.DataTo(&cache); err != nil {
        return nil, err
    }
    cache.LocationID = doc.Ref.ID
    return &cache, nil
}
```

**Why CacheDoc only reads `current` (not `history`):** The provider serves the *latest* pollen report. The history array exists in Firestore for potential future use (trend display), but the gRPC contract doesn't expose history — so we don't load it into memory.

### Step 3.4: Service Layer

**`internal/service/pollen.go`**:

```go
package service

import (
    "context"

    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
)

type PollenService struct {
    repo repository.PollenReader
}

func NewPollenService(repo repository.PollenReader) *PollenService {
    return &PollenService{repo: repo}
}

func (s *PollenService) GetAllReports(ctx context.Context) ([]repository.CacheDoc, error) {
    return s.repo.GetAll(ctx)
}

func (s *PollenService) GetReportByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
    return s.repo.GetByID(ctx, id)
}
```

**`internal/service/pollen_test.go`**:

```go
package service

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
)

type MockRepository struct {
    GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
    GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
}

func (m *MockRepository) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
    return m.GetAllFunc(ctx)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
    return m.GetByIDFunc(ctx, id)
}

func TestGetReportByID_Success(t *testing.T) {
    mockRepo := &MockRepository{
        GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
            return &repository.CacheDoc{
                LocationID:  id,
                LastUpdated: time.Now(),
            }, nil
        },
    }
    svc := NewPollenService(mockRepo)

    res, err := svc.GetReportByID(context.Background(), "house-nick")

    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
    if res.LocationID != "house-nick" {
        t.Errorf("expected location house-nick, got %s", res.LocationID)
    }
}

func TestGetReportByID_Error(t *testing.T) {
    mockRepo := &MockRepository{
        GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
            return nil, errors.New("db error")
        },
    }
    svc := NewPollenService(mockRepo)

    _, err := svc.GetReportByID(context.Background(), "house-nick")

    if err == nil {
        t.Error("expected error, got nil")
    }
}
```

### Step 3.5: Transport Layer

**`internal/transport/handler.go`**:

```go
package transport

import (
    "context"
    "log/slog"

    pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
    pb.UnimplementedPollenServiceServer
    svc *service.PollenService
}

func NewGrpcHandler(svc *service.PollenService) *GrpcHandler {
    return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) GetAllPollenReports(ctx context.Context, req *pb.GetAllPollenReportsRequest) (*pb.GetAllPollenReportsResponse, error) {
    docs, err := h.svc.GetAllReports(ctx)
    if err != nil {
        slog.Error("Failed to retrieve pollen data", "error", err)
        return nil, status.Errorf(codes.Unknown, "Failed to retrieve pollen data: %v", err)
    }

    var reports []*pb.PollenReport
    for i := range docs {
        reports = append(reports, mapToProto(&docs[i]))
    }

    return &pb.GetAllPollenReportsResponse{Reports: reports}, nil
}

func (h *GrpcHandler) GetPollenReport(ctx context.Context, req *pb.GetPollenReportRequest) (*pb.GetPollenReportResponse, error) {
    doc, err := h.svc.GetReportByID(ctx, req.LocationId)
    if err != nil {
        slog.Error("Failed to retrieve pollen data", "error", err)
        return nil, status.Errorf(codes.Unknown, "Failed to retrieve pollen data: %v", err)
    }

    return &pb.GetPollenReportResponse{Report: mapToProto(doc)}, nil
}

func mapToProto(doc *repository.CacheDoc) *pb.PollenReport {
    report := &pb.PollenReport{
        LocationId:      doc.LocationID,
        CollectedAt:     timestamppb.New(doc.Current.CollectedAt),
        OverallIndex:    int32(doc.Current.OverallIndex),
        OverallCategory: doc.Current.OverallCategory,
        DominantType:    doc.Current.DominantType,
    }

    for _, t := range doc.Current.Types {
        report.Types = append(report.Types, &pb.PollenType{
            Code:     t.Code,
            Index:    int32(t.Index),
            Category: t.Category,
            InSeason: t.InSeason,
        })
    }

    for _, p := range doc.Current.Plants {
        report.Plants = append(report.Plants, &pb.PollenPlant{
            Code:        p.Code,
            DisplayName: p.DisplayName,
            Index:       int32(p.Index),
            Category:    p.Category,
            InSeason:    p.InSeason,
        })
    }

    return report
}
```

**`internal/transport/handler_test.go`**:

```go
package transport

import (
    "context"
    "testing"
    "time"

    pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
)

type MockReader struct {
    GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
    GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
}

func (m *MockReader) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
    return m.GetAllFunc(ctx)
}

func (m *MockReader) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
    return m.GetByIDFunc(ctx, id)
}

func TestGetPollenReport_Mapping(t *testing.T) {
    now := time.Now()
    mockRepo := &MockReader{
        GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
            return &repository.CacheDoc{
                LocationID:  id,
                LastUpdated: now,
                Current: repository.PollenSnapshot{
                    CollectedAt:     now,
                    OverallIndex:    4,
                    OverallCategory: "High",
                    DominantType:    "TREE",
                    Types: []repository.StoredPollenType{
                        {Code: "TREE", Index: 4, Category: "High", InSeason: true},
                        {Code: "GRASS", Index: 1, Category: "Very Low", InSeason: false},
                    },
                    Plants: []repository.StoredPollenPlant{
                        {Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
                    },
                },
            }, nil
        },
    }

    svc := service.NewPollenService(mockRepo)
    handler := NewGrpcHandler(svc)

    req := &pb.GetPollenReportRequest{LocationId: "house-nick"}
    resp, err := handler.GetPollenReport(context.Background(), req)

    if err != nil {
        t.Fatalf("handler returned error: %v", err)
    }
    if resp.Report.OverallIndex != 4 {
        t.Errorf("expected OverallIndex 4, got %d", resp.Report.OverallIndex)
    }
    if resp.Report.DominantType != "TREE" {
        t.Errorf("expected DominantType TREE, got %s", resp.Report.DominantType)
    }
    if len(resp.Report.Types) != 2 {
        t.Errorf("expected 2 types, got %d", len(resp.Report.Types))
    }
    if len(resp.Report.Plants) != 1 {
        t.Errorf("expected 1 plant, got %d", len(resp.Report.Plants))
    }
    if resp.Report.Plants[0].Code != "JUNIPER" {
        t.Errorf("expected plant JUNIPER, got %s", resp.Report.Plants[0].Code)
    }
}

func TestGetAllPollenReports(t *testing.T) {
    now := time.Now()
    mockRepo := &MockReader{
        GetAllFunc: func(ctx context.Context) ([]repository.CacheDoc, error) {
            return []repository.CacheDoc{
                {LocationID: "house-nick", LastUpdated: now},
                {LocationID: "house-nita", LastUpdated: now},
            }, nil
        },
    }

    svc := service.NewPollenService(mockRepo)
    handler := NewGrpcHandler(svc)

    resp, err := handler.GetAllPollenReports(context.Background(), &pb.GetAllPollenReportsRequest{})

    if err != nil {
        t.Fatalf("handler returned error: %v", err)
    }
    if len(resp.Reports) != 2 {
        t.Errorf("expected 2 reports, got %d", len(resp.Reports))
    }
}
```

### Step 3.6: Main Entry Point

**`cmd/server/main.go`**:

```go
package main

import (
    "context"
    "log/slog"
    "net"
    "os"
    "os/signal"
    "syscall"

    "github.com/joho/godotenv"
    "github.com/nickfang/personal-dashboard/services/shared"
    pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
    "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/transport"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
)

func main() {
    // 1. Setup Logging
    shared.InitLogging()

    slog.Info("Pollen Provider starting", "version", "1.0.0", "debug", os.Getenv("DEBUG"))

    if err := godotenv.Load(); err != nil {
        slog.Debug("No .env file found, using system environment variables", "error", err)
    }

    // 2. Load Config
    projectID := os.Getenv("GCP_PROJECT_ID")
    port := os.Getenv("PORT")
    if port == "" {
        port = "50052"
    }
    if projectID == "" {
        slog.Error("Missing required env var: GCP_PROJECT_ID")
        os.Exit(1)
    }

    ctx := context.Background()

    // 3. Initialize Layers
    repo, err := repository.NewFirestoreRepository(ctx, projectID)
    if err != nil {
        slog.Error("Failed to initialize repository", "error", err)
        os.Exit(1)
    }
    defer repo.Close()

    svc := service.NewPollenService(repo)
    handler := transport.NewGrpcHandler(svc)

    // 4. Start gRPC Server
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        slog.Error("Failed to listen", "port", port, "error", err)
        os.Exit(1)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterPollenServiceServer(grpcServer, handler)

    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
    healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

    if os.Getenv("DEBUG") == "true" {
        reflection.Register(grpcServer)
    }

    // 5. Graceful Shutdown
    go func() {
        slog.Info("Pollen Provider Server listening", "port", port)
        if err := grpcServer.Serve(lis); err != nil {
            slog.Error("Failed to serve gRPC", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    slog.Info("Shutting down server gracefully...")
    healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
    grpcServer.GracefulStop()
    slog.Info("Server stopped")
}
```

### Step 3.7: Dockerfile

Same wider-context pattern as the collector — build context is `services/`.

```dockerfile
# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY pollen-provider/ ./pollen-provider/

WORKDIR /app/pollen-provider
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/pollen-provider/bin cmd/server/main.go

# Stage 2: Final image
FROM alpine:3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/pollen-provider/bin ./pollen-provider

ENV PORT=50052

CMD ["./pollen-provider"]
```

---

## Phase 4: Dashboard API Integration

**Goal:** Add a pollen gRPC client to the dashboard-api and include pollen data in the aggregated response.

### Step 4.1: Create Pollen Client

Create `services/dashboard-api/internal/clients/pollen-client.go`:

```go
package clients

import (
    "context"
    "log/slog"
    "strings"

    pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
    "google.golang.org/api/idtoken"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/credentials/oauth"
)

type PollenClient struct {
    conn   *grpc.ClientConn
    client pb.PollenServiceClient
}

func NewPollenClient(ctx context.Context, address string) (*PollenClient, error) {
    var opts []grpc.DialOption

    if strings.HasSuffix(address, ":443") {
        audience := "https://" + strings.TrimSuffix(address, ":443")
        tokenSource, err := idtoken.NewTokenSource(ctx, audience)
        if err != nil {
            slog.Error("Failed to create token source for pollen client", "error", err, "audience", audience)
            return nil, err
        }
        opts = append(opts,
            grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
            grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: tokenSource}),
        )
        slog.Info("Pollen client using Google ID Token auth", "address", address, "audience", audience)
    } else {
        opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
        slog.Info("Pollen client using insecure credentials", "address", address)
    }

    conn, err := grpc.NewClient(address, opts...)
    if err != nil {
        slog.Error("Failed to create pollen gRPC client", "error", err, "address", address)
        return nil, err
    }
    client := pb.NewPollenServiceClient(conn)

    return &PollenClient{conn: conn, client: client}, nil
}

func (c *PollenClient) Close() error {
    return c.conn.Close()
}

func (c *PollenClient) GetPollenReports(ctx context.Context) ([]*pb.PollenReport, error) {
    resp, err := c.client.GetAllPollenReports(ctx, &pb.GetAllPollenReportsRequest{})
    if err != nil {
        slog.Error("Failed to get pollen reports", "error", err)
        return nil, err
    }
    return resp.Reports, nil
}

func (c *PollenClient) GetPollenReport(ctx context.Context, locationID string) (*pb.PollenReport, error) {
    resp, err := c.client.GetPollenReport(ctx, &pb.GetPollenReportRequest{LocationId: locationID})
    if err != nil {
        slog.Error("Failed to get pollen report", "error", err)
        return nil, err
    }
    return resp.Report, nil
}
```

**Note about DRY:** The TLS/auth logic in `NewPollenClient` is nearly identical to `NewWeatherClient`. This duplication is intentional — the shared module (`services/shared/`) is stdlib-only by design, and gRPC client auth pulls in heavy dependencies (`google.golang.org/grpc`, `google.golang.org/api/idtoken`). If a third gRPC client is added, extracting a shared `grpcDialOptions(address)` helper into its own module (or into `shared/` with an accepted dependency cost) would be appropriate.

### Step 4.2: Update Handler Interface & Aggregation

Update `services/dashboard-api/internal/handlers/handler.go` to add the `PollenFetcher` interface and parallel fetch:

```go
package handlers

import (
    "context"
    "encoding/json"
    "net/http"

    pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
    weatherPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
    "golang.org/x/sync/errgroup"
    "google.golang.org/protobuf/encoding/protojson"
)

type WeatherFetcher interface {
    GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error)
    GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error)
}

type PollenFetcher interface {
    GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error)
    GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error)
}

type DashboardHandler struct {
    weatherClient WeatherFetcher
    pollenClient  PollenFetcher
}

func NewDashboardHandler(wc WeatherFetcher, pc PollenFetcher) *DashboardHandler {
    return &DashboardHandler{
        weatherClient: wc,
        pollenClient:  pc,
    }
}

var protoMarshaler = protojson.MarshalOptions{}

func aggregatePressureStats(stats []*weatherPb.PressureStat) (map[string]json.RawMessage, error) {
    result := make(map[string]json.RawMessage, len(stats))
    for _, stat := range stats {
        data, err := protoMarshaler.Marshal(stat)
        if err != nil {
            return nil, err
        }
        result[stat.LocationId] = data
    }
    return result, nil
}

func aggregatePollenReports(reports []*pollenPb.PollenReport) (map[string]json.RawMessage, error) {
    result := make(map[string]json.RawMessage, len(reports))
    for _, report := range reports {
        data, err := protoMarshaler.Marshal(report)
        if err != nil {
            return nil, err
        }
        result[report.LocationId] = data
    }
    return result, nil
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
    g, ctx := errgroup.WithContext(r.Context())

    var pressureStats []*weatherPb.PressureStat
    var pollenReports []*pollenPb.PollenReport

    g.Go(func() error {
        stats, err := h.weatherClient.GetPressureStats(ctx)
        if err != nil {
            return err
        }
        pressureStats = stats
        return nil
    })

    g.Go(func() error {
        reports, err := h.pollenClient.GetPollenReports(ctx)
        if err != nil {
            return err
        }
        pollenReports = reports
        return nil
    })

    if err := g.Wait(); err != nil {
        RespondWithGrpcError(w, err, "Failed to fetch dashboard data")
        return
    }

    aggregatedPressure, err := aggregatePressureStats(pressureStats)
    if err != nil {
        http.Error(w, "Failed to encode pressure response", http.StatusInternalServerError)
        return
    }

    aggregatedPollen, err := aggregatePollenReports(pollenReports)
    if err != nil {
        http.Error(w, "Failed to encode pollen response", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(map[string]any{
        "pressure": aggregatedPressure,
        "pollen":   aggregatedPollen,
    }); err != nil {
        http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
    }
}
```

### Step 4.3: Update main.go

Update `services/dashboard-api/cmd/server/main.go` to initialize the pollen client:

```go
// Add to imports
// (existing imports remain)

// Add after weatherAddr config
pollenAddr := os.Getenv("POLLEN_PROVIDER_ADDR")
if pollenAddr == "" {
    pollenAddr = "localhost:50052"
}

// Add after weatherClient initialization
pollenClient, err := clients.NewPollenClient(context.Background(), pollenAddr)
if err != nil {
    slog.Error("Failed to initialize pollen client", "error", err)
    os.Exit(1)
}
defer pollenClient.Close()

// Update handler initialization
dashboardHandler := handlers.NewDashboardHandler(weatherClient, pollenClient)
```

### Step 4.4: Update .env.example

Add to `services/dashboard-api/.env.example`:

```env
POLLEN_PROVIDER_ADDR=localhost:50052
```

### Step 4.5: Add errgroup Dependency

```bash
cd services/dashboard-api
go get golang.org/x/sync/errgroup
```

### Step 4.6: Update Handler Tests

The `NewDashboardHandler` signature changed. Update existing tests to pass both a `WeatherFetcher` mock and a `PollenFetcher` mock.

---

## Phase 5: Infrastructure (Terraform)

### Step 5.1: Create `infra/pollen_collector.tf`

```hcl
# Service Account for the Pollen Collector Job
resource "google_service_account" "pollen_collector_sa" {
  account_id   = "pollen-collector-sa"
  display_name = "Service Account for Pollen Collector Job"
}

# Grant permissions to write to Firestore
resource "google_project_iam_member" "pollen_firestore_writer" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Grant permission to invoke Cloud Run jobs (Self-invocation for Scheduler)
resource "google_project_iam_member" "pollen_collector_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Grant the Pollen Collector access to the existing Google Maps API Key secret.
# The secret itself is defined in weather_collector.tf — we only add an IAM binding here.
resource "google_secret_manager_secret_iam_member" "pollen_secret_access" {
  secret_id = google_secret_manager_secret.google_maps_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Bootstrap Docker Image
resource "null_resource" "pollen_collector_bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services \
        --config ../services/pollen-collector/cloudbuild.yaml \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-collector:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Cloud Run Job
resource "google_cloud_run_v2_job" "pollen_collector" {
  name     = "pollen-collector-job"
  location = var.region

  template {
    template {
      service_account = google_service_account.pollen_collector_sa.email
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-collector:latest"
        env {
          name  = "GCP_PROJECT_ID"
          value = var.project_id
        }
        env {
          name = "GOOGLE_MAPS_API_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.google_maps_key.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [google_project_service.run, null_resource.pollen_collector_bootstrap, google_secret_manager_secret_version.google_maps_key_version]
}

# Cloud Scheduler — Runs at 6:00 AM and 2:00 PM Central
resource "google_cloud_scheduler_job" "pollen_cron" {
  name             = "trigger-pollen-collector"
  description      = "Triggers the pollen collector job twice daily"
  schedule         = "0 6,14 * * *"
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.pollen_collector.name}:run"

    oauth_token {
      service_account_email = google_service_account.pollen_collector_sa.email
    }
  }

  depends_on = [google_project_service.scheduler]
}
```

### Step 5.2: Create `infra/pollen_provider.tf`

```hcl
# Service Account for the Pollen Provider Service
resource "google_service_account" "pollen_provider_sa" {
  account_id   = "pollen-provider-sa"
  display_name = "Service Account for Pollen Provider service"
}

# Grant read-only Firestore access
resource "google_project_iam_member" "pollen_firestore_reader" {
  project = var.project_id
  role    = "roles/datastore.viewer"
  member  = "serviceAccount:${google_service_account.pollen_provider_sa.email}"
}

# Bootstrap Docker Image
resource "null_resource" "pollen_provider_bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services \
        --config ../services/pollen-provider/cloudbuild.yaml \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-provider:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Cloud Run Service
resource "google_cloud_run_v2_service" "pollen_provider" {
  name     = "pollen-provider-service"
  location = var.region

  template {
    service_account = google_service_account.pollen_provider_sa.email
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-provider:latest"
      ports {
        container_port = 50052
        name           = "h2c"
      }
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [google_project_service.run, null_resource.pollen_provider_bootstrap]
}
```

### Step 5.3: Update `infra/dashboard_api.tf`

Add the pollen provider address env var and IAM binding:

```hcl
# Allow Dashboard API to call Pollen Provider
resource "google_cloud_run_v2_service_iam_member" "pollen_provider_invoker" {
  name     = google_cloud_run_v2_service.pollen_provider.name
  location = google_cloud_run_v2_service.pollen_provider.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.dashboard_api_sa.email}"
}
```

Add the `POLLEN_PROVIDER_ADDR` env var to the dashboard-api container definition:

```hcl
env {
  name  = "POLLEN_PROVIDER_ADDR"
  value = "${trimprefix(google_cloud_run_v2_service.pollen_provider.uri, "https://")}:443"
}
```

---

## Phase 6: Build & Development Tooling

### Step 6.1: Update `go.work`

```
go 1.25.6

use (
    ./services/dashboard-api
    ./services/pollen-collector
    ./services/pollen-provider
    ./services/shared
    ./services/weather-collector
    ./services/weather-provider
)
```

**Note:** `go.work` and `replace` are two independent mechanisms for local module resolution. `go.work` is for the developer's environment — it tells `go build`, `go test`, and IDEs to resolve modules listed in its `use` block directly from disk. The `replace` directive in each service's `go.mod` serves tooling that is not workspace-aware, such as Docker builds (which copy `go.mod` but not `go.work`).

### Step 6.2: Update `docker-compose.yml`

All services now use `context: ./services` with a `dockerfile:` path so the shared module is available during builds.

```yaml
services:
  weather-provider:
    build:
      context: ./services
      dockerfile: weather-provider/Dockerfile
    ports:
      - "50051:50051"
    env_file:
      - services/weather-provider/.env
    volumes:
      - ~/.config/gcloud:/root/.config/gcloud

  pollen-provider:
    build:
      context: ./services
      dockerfile: pollen-provider/Dockerfile
    ports:
      - "50052:50052"
    env_file:
      - services/pollen-provider/.env
    volumes:
      - ~/.config/gcloud:/root/.config/gcloud

  dashboard-api:
    build:
      context: ./services
      dockerfile: dashboard-api/Dockerfile
    ports:
      - "8080:8080"
    env_file:
      - services/dashboard-api/.env
    environment:
      - WEATHER_PROVIDER_ADDR=weather-provider:50051
      - POLLEN_PROVIDER_ADDR=pollen-provider:50052
    depends_on:
      - weather-provider
      - pollen-provider
```

**Important:** This is a breaking change for the existing weather-provider and dashboard-api Dockerfiles too. They will also need to be updated to use the `COPY shared/ ./shared/` pattern. See the note at the end of this phase.

### Step 6.3: Update `Makefile`

Add the following sections. Note that `docker build` commands now use `-f` to specify the Dockerfile while building from the `services/` context.

```makefile
# Add to .PHONY at top:
#   pc-dev pc-build pc-run pc-test
#   pp-dev pp-build pp-test

# Update proto target (--path scopes generation to each service's proto package):
proto: ## Generate Go code for all services via Buf
	cd services/weather-provider && buf generate ../protos --path ../protos/weather-provider
	cd services/pollen-provider && buf generate ../protos --path ../protos/pollen-provider
	cd services/dashboard-api && buf generate ../protos

proto-clean: ## Remove all generated proto files
	rm -rf services/weather-provider/internal/gen/go/*
	rm -rf services/pollen-provider/internal/gen/go/*
	rm -rf services/dashboard-api/internal/gen/go/*

# ==============================================================================
# Service: Pollen Collector (Job)
# ==============================================================================
##@ Pollen Collector
pc-dev: ## Run Pollen Collector locally (Go)
	-cd services/pollen-collector && go run main.go

pc-build: ## Build Pollen Collector image
	docker build -t pollen-collector -f services/pollen-collector/Dockerfile services

pc-run: pc-build ## Run Pollen Collector container (One-off job)
	docker run --rm -it \
		--env-file services/pollen-collector/.env \
		-v ~/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_APPLICATION_CREDENTIALS=/root/.config/gcloud/application_default_credentials.json \
		pollen-collector

pc-test: ## Run Pollen Collector tests
	cd services/pollen-collector && go test ./...

# ==============================================================================
# Service: Pollen Provider (Server)
# ==============================================================================
##@ Pollen Provider
pp-dev: ## Run Pollen Provider locally (Go)
	-cd services/pollen-provider && go run cmd/server/main.go

pp-build: ## Build Pollen Provider image
	docker build -t pollen-provider -f services/pollen-provider/Dockerfile services

pp-test: ## Run Pollen Provider tests
	cd services/pollen-provider && go test ./...
```

**Existing services affected:** The `wc-build`, `wp-build`, and `da-build` Makefile targets also need updating to the `-f` pattern once their Dockerfiles are updated to use the shared module. This should happen in a prerequisite PR before the pollen implementation begins.

---

## Phase 7: CI/CD (GitHub Actions)

### Step 7.1: Create `.github/workflows/verify-pollen-collector.yml`

```yaml
name: Verify Pollen Collector

on:
  pull_request:
    branches: [ main ]
    paths:
      - 'services/pollen-collector/**'
      - 'services/shared/**'
      - '.github/workflows/verify-pollen-collector.yml'

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Verify Dependencies
        working-directory: ./services/pollen-collector
        run: go mod verify

      - name: Run Unit Tests
        working-directory: ./services/pollen-collector
        run: go test -v ./...

      - name: Verify Build
        working-directory: ./services/pollen-collector
        run: go build -v ./...
```

### Step 7.2: Create `.github/workflows/deploy-pollen-collector.yml`

```yaml
name: Deploy Pollen Collector

on:
  push:
    branches: [ main ]
    paths:
      - 'services/pollen-collector/**'
      - 'services/shared/**'
      - '.github/workflows/deploy-pollen-collector.yml'

env:
  GCP_REGION: us-central1
  JOB_NAME: pollen-collector-job
  IMAGE_NAME: us-central1-docker.pkg.dev/fang-gcp/personal-dashboard/pollen-collector

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: 'read'
      id-token: 'write'

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Google Auth
        id: auth
        uses: 'google-github-actions/auth@v2'
        with:
          project_id: 'fang-gcp'
          workload_identity_provider: 'projects/964768292331/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider'
          service_account: 'github-actions-sa@fang-gcp.iam.gserviceaccount.com'

      - name: 'Set up Cloud SDK'
        uses: 'google-github-actions/setup-gcloud@v2'

      - name: 'Docker Auth'
        run: |
          gcloud auth configure-docker ${{ env.GCP_REGION }}-docker.pkg.dev

      - name: 'Build and Push Image'
        run: |
          IMAGE_TAG="${{ env.IMAGE_NAME }}:${{ github.sha }}"

          docker build -t "$IMAGE_TAG" -f services/pollen-collector/Dockerfile services
          docker push "$IMAGE_TAG"

          docker tag "$IMAGE_TAG" "${{ env.IMAGE_NAME }}:latest"
          docker push "${{ env.IMAGE_NAME }}:latest"

      - name: 'Deploy to Cloud Run'
        run: |
          gcloud run jobs update ${{ env.JOB_NAME }} \
            --image "${{ env.IMAGE_NAME }}:${{ github.sha }}" \
            --region ${{ env.GCP_REGION }}
```

### Step 7.3: Create `.github/workflows/verify-pollen-provider.yml`

```yaml
name: Verify Pollen Provider

on:
  pull_request:
    branches: [ main ]
    paths:
      - 'services/pollen-provider/**'
      - 'services/shared/**'
      - '.github/workflows/verify-pollen-provider.yml'

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Verify Dependencies
        working-directory: ./services/pollen-provider
        run: go mod verify

      - name: Run Unit Tests
        working-directory: ./services/pollen-provider
        run: go test -v ./...

      - name: Verify Build
        working-directory: ./services/pollen-provider
        run: go build -v ./...
```

### Step 7.4: Create `.github/workflows/deploy-pollen-provider.yml`

```yaml
name: Deploy Pollen Provider

on:
  push:
    branches: [ main ]
    paths:
      - 'services/pollen-provider/**'
      - 'services/shared/**'
      - '.github/workflows/deploy-pollen-provider.yml'

env:
  GCP_REGION: us-central1
  SERVICE_NAME: pollen-provider-service
  IMAGE_NAME: us-central1-docker.pkg.dev/fang-gcp/personal-dashboard/pollen-provider

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: 'read'
      id-token: 'write'

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Google Auth
        id: auth
        uses: 'google-github-actions/auth@v2'
        with:
          project_id: 'fang-gcp'
          workload_identity_provider: 'projects/964768292331/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider'
          service_account: 'github-actions-sa@fang-gcp.iam.gserviceaccount.com'

      - name: 'Set up Cloud SDK'
        uses: 'google-github-actions/setup-gcloud@v2'

      - name: 'Docker Auth'
        run: |
          gcloud auth configure-docker ${{ env.GCP_REGION }}-docker.pkg.dev

      - name: 'Build and Push Image'
        run: |
          IMAGE_TAG="${{ env.IMAGE_NAME }}:${{ github.sha }}"

          docker build -t "$IMAGE_TAG" -f services/pollen-provider/Dockerfile services
          docker push "$IMAGE_TAG"

          docker tag "$IMAGE_TAG" "${{ env.IMAGE_NAME }}:latest"
          docker push "${{ env.IMAGE_NAME }}:latest"

      - name: 'Deploy to Cloud Run'
        run: |
          gcloud run services update ${{ env.SERVICE_NAME }} \
            --image "${{ env.IMAGE_NAME }}:${{ github.sha }}" \
            --region ${{ env.GCP_REGION }}
```

---

## Phase 8: Verification

### Step 8.1: Local Testing (All Unit Tests)

```bash
make pc-test   # Pollen Collector tests
make pp-test   # Pollen Provider tests
make da-test   # Dashboard API tests (updated handler signature)
```

### Step 8.2: Local Integration Test

```bash
# 1. Run collector once to populate Firestore
make pc-dev

# 2. Start provider + dashboard-api
make compose-up

# 3. Test pollen provider directly (requires grpcurl)
grpcurl -plaintext -d '{}' localhost:50052 pollen_provider.v1.PollenService/GetAllPollenReports
grpcurl -plaintext -d '{"location_id": "house-nick"}' localhost:50052 pollen_provider.v1.PollenService/GetPollenReport

# 4. Test aggregated dashboard endpoint
curl http://localhost:8080/api/v1/dashboard | jq .
```

Expected dashboard response shape:

```json
{
  "pressure": {
    "house-nick": { "locationId": "house-nick", "lastUpdated": "...", "delta3h": 0.5, "trend": "rising" }
  },
  "pollen": {
    "house-nick": {
      "locationId": "house-nick",
      "collectedAt": "2026-02-19T12:00:00Z",
      "overallIndex": 4,
      "overallCategory": "High",
      "dominantType": "TREE",
      "types": [
        { "code": "TREE", "index": 4, "category": "High", "inSeason": true },
        { "code": "GRASS", "index": 1, "category": "Very Low", "inSeason": false },
        { "code": "WEED", "index": 0, "category": "None", "inSeason": false }
      ],
      "plants": [
        { "code": "JUNIPER", "displayName": "Juniper", "index": 4, "category": "High", "inSeason": true }
      ]
    }
  }
}
```

### Step 8.3: Cloud Verification

```bash
# Proxy the private pollen-provider
gcloud run services proxy pollen-provider-service --port=50052

# Test via grpcurl
grpcurl -plaintext -d '{}' localhost:50052 pollen_provider.v1.PollenService/GetAllPollenReports
```

---

## Implementation Order Summary

| Phase | Service(s) | What | PR(s) |
|---|---|---|---|
| 0 | shared, root | Shared module + go.work + update existing Dockerfiles/Makefile/CI | **Must be first — prerequisite PR** |
| 1 | protos, pollen-provider, dashboard-api | Proto contract + code generation | Can be its own PR |
| 2 | pollen-collector | Collector binary + tests | Can be its own PR |
| 3 | pollen-provider | Provider service (all layers) + tests | Can be its own PR |
| 4 | dashboard-api | Pollen client + handler integration | Can be its own PR |
| 5 | infra | Terraform resources for both services | Can be its own PR |
| 6 | root | docker-compose, Makefile (pollen targets) | Include with Phase 1 or 3 |
| 7 | .github/workflows | CI/CD pipelines (4 files) | Include with Phase 2 and 3 |
| 8 | — | Verification (local + cloud) | Post-deploy |

**Recommended PR sequence:** Phase 0 (shared module + existing service updates) → Phase 1 + 6 → Phase 2 + 7 (collector workflows) → Phase 3 + 7 (provider workflows) → Phase 4 → Phase 5

**Phase 0 scope (prerequisite PR):**
*   Create `services/shared/` module (locations.go, constants.go, logging.go)
*   Update `go.work` to include shared
*   Update existing services (weather-collector, weather-provider, dashboard-api) to import from shared
*   Update all existing Dockerfiles to use `services/` build context
*   Update existing Makefile build targets for `-f` pattern
*   Update existing CI/CD workflows to include `services/shared/**` in path triggers
*   Update `docker-compose.yml` contexts
