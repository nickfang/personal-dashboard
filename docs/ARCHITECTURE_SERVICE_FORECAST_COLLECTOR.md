# Forecast Collector Service Architecture

## 1. Overview
The **Forecast Collector** (`services/forecast-collector`) is a background worker that fetches the hourly weather forecast from the Google Weather API, detects pressure-drop alerts, and writes both to Firestore. It is the forward-looking counterpart to the **Weather Collector**, which records observations as they happen.

It is also the only service that *creates* `Alert` records. The **Weather Provider** reads them back out of the cache; the Dashboard API and clients only ever display them. See [Issue #62](https://github.com/nickfang/personal-dashboard/issues/62).

For platform-level details (Deployment, Terraform, Identity), see **[ARCHITECTURE_INFRASTRUCTURE.md](./ARCHITECTURE_INFRASTRUCTURE.md)**.

## 2. System Architecture

### Components
1.  **Forecast Collector (`services/forecast-collector`)**
    *   **Role**: Background Worker (Writer).
    *   **Runtime**: Cloud Run Job.
    *   **Trigger**: Cloud Scheduler (every 6 hours).
    *   **Architecture:** Layered (Client → Service → Repository), matching the collector pattern from [Issue #31](https://github.com/nickfang/personal-dashboard/issues/31).
    *   **Folder Structure:**
        ```text
        services/forecast-collector/
        ├── cmd/
        │   ├── main.go                  # Wiring + config + orchestration loop
        │   └── main_test.go             # collectAll() tests (partial failure, all-fail, empty) + env parsing
        ├── internal/
        │   ├── api/
        │   │   ├── api.go               # Fetcher interface + Client (HTTP + pagination + retry)
        │   │   ├── api_test.go          # Pagination, retry/non-retryable, API-key-not-leaked tests
        │   │   ├── types.go             # Google Weather API forecast response types
        │   │   └── testdata/            # Recorded forecast_hours responses (2 pages)
        │   ├── service/
        │   │   ├── collector.go         # CollectorService orchestration + alert delivery
        │   │   ├── collector_test.go    # Orchestration, alert-wiring, delivery-gate, failure-path tests
        │   │   ├── convert.go           # CtoF(), MapToForecastPoint(), MapRun()
        │   │   ├── convert_test.go      # Mapping + invalid-hour rejection tests
        │   │   ├── detect.go            # AlertConfig + DetectPressureAlerts()
        │   │   └── detect_test.go       # Window sliding, episode coalescing, severity, tolerance
        │   ├── repository/
        │   │   ├── writer.go            # Writer interface + MergeFunc + MarkNotified + Firestore implementation
        │   │   ├── writer_test.go       # buildCacheDoc() + applyNotifiedAt() tests
        │   │   └── types.go             # ForecastPoint, ForecastRun, ForecastCacheDoc
        │   └── testutil/
        │       └── mocks.go             # MockFetcher, MockWriter, MockSender
        ├── Dockerfile
        └── go.mod
        ```
    *   **Responsibility**:
        *   Fetch the hourly forecast (default 72h horizon) from the Google Weather API.
        *   Detect pressure-drop alerts over the forecast.
        *   Perform "Dual-Write" to Firestore (Archive + Cache), merging alerts transactionally.
        *   Deliver undelivered active alerts by email and record the delivery (section 5).

2.  **Weather Provider (`services/weather-provider`)**
    *   **Role**: API Service (Reader).
    *   **Responsibility**: Read `forecast_cache` and serve forecast points plus alerts to the Dashboard API via `ForecastService`. Alerts ride inside the forecast response rather than a dedicated RPC, because they live in the same cache document — see `services/protos/weather-provider/v1/forecast.proto`.

## 3. Data Strategy (Firestore)

Same **Dual-Write** strategy as the Weather Collector, in the same `weather-log` database.

### Collection 1: `forecast_raw` (The Archive)
*   **Purpose**: Append-only audit of every forecast run. Lets a past prediction be compared against what actually happened.
*   **Write Frequency**: 1 document per location per run (every 6 hours).
*   **Schema** (`ForecastRun`):
    ```json
    {
      "location": "string",
      "issued_at": "timestamp",
      "points": [
        {
          "valid_time": "timestamp",
          "humidity_pct": "int",
          "precipitation_pct": "int",
          "uv_index": "int",
          "pressure_mb": "float64",
          "wind_dir_deg": "int",
          "temp_c": "float64",
          "temp_feel_c": "float64",
          "dewpoint_c": "float64",
          "wind_speed_kph": "float64",
          "wind_gust_kph": "float64",
          "temp_f": "float64",
          "temp_feel_f": "float64",
          "dewpoint_f": "float64"
        }
      ]
    }
    ```

### Collection 2: `forecast_cache` (The Latest View)
*   **Purpose**: Read-optimized latest forecast plus the live alert set for the dashboard.
*   **Structure**: 1 Document per Location ID (e.g., `house-nick`), fully replaced on each run.
*   **Schema** (`ForecastCacheDoc`):
    ```json
    {
      "location": "string",
      "issued_at": "timestamp",
      "points": ["// Same ForecastPoint shape as forecast_raw"],
      "alerts": ["// Merged Alert records — see section 5"]
    }
    ```

## 4. Pressure-Drop Detection

Detection is inline in the collector — the analog of the Weather Collector's pressure-delta analysis, but run over predicted rather than observed pressure.

*   **Window sliding**: A `WindowHours` window slides across the forecast. For each hour, the point closest to `valid_time + WindowHours` is located within a **±30 minute tolerance** (`windowTolerance`), so a missing or shifted forecast hour doesn't drop the window.
*   **Episode coalescing**: Overlapping qualifying windows are collapsed into **one alert per continuous drop episode**, rather than emitting one alert per window position. The alert's `Value` is the steepest single-window delta in the episode; its window spans the whole episode.
*   **Severity**: `warning` at or beyond `DropThresholdMb`, upgraded to `severe` at or beyond `SevereThresholdMb`.
*   **Message**: Anchored on the hour where the steepest drop begins, e.g. `Thu 2 PM  -6.2 mb/3h  -8.1/6h`. The extended figure covers twice the window and is omitted when the horizon doesn't reach that far. Rendered in `America/Chicago` — Cloud Run runs in UTC, so the zone is loaded explicitly and requires tzdata in the image.

### Configuration

Thresholds are environment-tunable so they can be adjusted without a code change. Defaults come from `DefaultAlertConfig()` and are set explicitly in Terraform (`infra/staging/main.tf`, `infra/prod/main.tf`); an unset or invalid value falls back to the default with a warning.

| Env Var | Default | Meaning |
|---------|---------|---------|
| `FORECAST_HORIZON_HOURS` | `72` | How many forecast hours to request |
| `PRESSURE_DROP_MB` | `5` | Drop per window that raises a `warning` |
| `PRESSURE_SEVERE_MB` | `10` | Drop per window that raises a `severe` alert |
| `PRESSURE_WINDOW_HOURS` | `3` | Detection window length; also names the rule (`pressure-drop-3h`) |

Required (no default): `GOOGLE_MAPS_API_KEY`, `GCP_PROJECT_ID`.

Alert delivery (section 5) is configured separately. `NOTIFY_ENABLED` is a plain env var so delivery can be killed per environment without touching secrets; the password is the only value that goes through Secret Manager, since anything in `env_vars` lands in Terraform state and the Cloud Run config in plaintext. A disabled switch, or any of the three values empty, yields a no-op sender rather than an error — the job has to keep collecting when delivery is unconfigured.

| Env Var | Source | Default | Meaning |
|---------|--------|---------|---------|
| `NOTIFY_ENABLED` | `env_vars` | `false` | Master switch; delivery runs only when set to `true` |
| `NOTIFY_SMTP_USER` | `env_vars` | *(unset)* | Gmail address; also the `From` |
| `NOTIFY_SMTP_PASSWORD` | Secret Manager (`notify-smtp-password`) | *(unset)* | Google app password — requires 2-Step Verification on the account |
| `NOTIFY_EMAIL_TO` | `env_vars` | *(unset)* | Recipient |

Host and port are constants, not configuration: there is no second SMTP server to point at, and a wrong value would fail at delivery rather than at startup. Delivery is enabled in **both** environments — staging is separated by a `+staging` tagged recipient rather than by being switched off, so the delivery path is exercised before production.

## 5. Alert Lifecycle

The `Alert` contract lives in `services/shared/alerts.go` and is deliberately **source-agnostic** — a future pollen-spike detector reuses the struct unchanged.

*   **Stable identity**: `ComputeID()` hashes `location | rule | window start truncated to the hour`. Merging preserves the original ID across runs even as the episode's window shifts, which is what makes "one alert per episode" hold across a re-forecast.
*   **Transactional merge**: Detection is pure and runs once in the service layer; the merge against stored alerts runs inside the repository's Firestore transaction via a `MergeFunc` closure, so it sees prior alert state atomically with the write that replaces it. This mirrors the Weather Collector's `AnalyzeFunc`. The merge *logic* is unit-tested in `services/shared/alerts_test.go`; the transaction wrapper itself is not covered, as it requires a live Firestore.
*   **`MergeAlerts` rules**:
    *   Alerts whose window has fully passed are pruned.
    *   A detected alert overlapping a stored one keeps the stored ID and delivery record, and updates its numbers. Matching picks the **greatest window overlap**.
    *   A stored `resolved` alert that is detected again re-activates, keeping its delivery record.
    *   Stored alerts no longer detected become `resolved`, keeping their delivery record.
    *   Brand-new alerts come through as `active`.

### Status and delivery are two separate facts

`Status` says whether the condition is in the forecast. `NotifiedAt` says whether the user has been told. Collapsing them into a single field would make a delivered alert vanish from the dashboard while the drop is still hours away, and would lose the delivery record whenever forecast noise briefly resolved an alert that was still building.

| Field | Values | Meaning |
|-------|--------|---------|
| `status` | `active` | Condition is in the current forecast — clients render these |
| | `resolved` | Previously detected, no longer in the forecast — not rendered |
| `notified_at` | timestamp | When the alert was delivered |
| | zero | Never delivered |

**Delivery gate:** `Status == active && NotifiedAt.IsZero()`.

**Escalate-if-worse** (`mergedNotifiedAt`): a delivered alert has its `NotifiedAt` **cleared** — re-arming delivery through that same single gate — when the predicted drop worsens by at least `AlertEscalationStepMb` (1.0 mb) or its severity upgrades to `severe`. Otherwise the stored `NotifiedAt` carries forward, so re-forecasting the same episode does not re-notify.

This keys off the timestamp rather than a status value on purpose. An alert can flap — delivered, resolved by one noisy run near the threshold, then detected again. Keying escalation off the status string would miss an escalation that follows a flap, silently dropping a warning that had since upgraded to `severe`. `TestMergeAlerts_FlapDoesNotRedeliver` and `TestMergeAlerts_EscalationAfterFlapRearmsDelivery` pin the pair.

### Delivery

Alerts are delivered **inline from this collector**, immediately after the cache write commits — no Pub/Sub, no separate notifier service. Issue [#68](https://github.com/nickfang/personal-dashboard/issues/68) originally specified a Pub/Sub → notifier design; for a 6-hour cron over 3 locations delivering to one person, a new service, module, and CI pipeline (written twice across staging and prod) was not worth the decoupling. The seam that would allow it later is `notify.Sender`.

*   **Channel**: email over Gmail SMTP (`smtp.gmail.com:587`), authenticating with a Google app password held in Secret Manager. Chosen because it needs no new vendor account and `net/smtp` is stdlib, so `services/shared/go.mod` stays dependency-free.
*   **Package**: `services/shared/notify` — not `forecast-collector/internal/`, because Go's `internal` rule would block a pollen detector ([#69](https://github.com/nickfang/personal-dashboard/issues/69)) from reusing it. `Sender` is a one-method interface; adding SMS for `severe` alerts later is a new file plus a routing decision.
*   **Message shape**: the subject *is* the notification — a phone lock screen shows little else — so it carries severity, location, and the formatted delta: `Pressure drop (severe) - house-nick: Thu 2 PM  -6.2 mb/3h  -8.1/6h`. The body carries the full record. The subject is kept ASCII; non-ASCII would require RFC 2047 encoded-word wrapping.
*   **One email per alert.** A run can emit several (3 locations × possibly multiple episodes). Batching would be quieter but makes "which IDs delivered" ambiguous on partial failure, which is exactly what marking needs.
*   **Not `smtp.SendMail`**: `net/smtp` is frozen and context-unaware — `SendMail` sets no deadline and dials bare, so a hung connection would block the job until Cloud Run's task timeout. The client is built explicitly with the connection deadline derived from `ctx`, and the dialer is a struct field so tests can substitute a fake listener.

**Ordering: deliver, then mark.** `UpdateCache` returns the merged alert set it committed (returned explicitly rather than captured through the `MergeFunc` closure, because Firestore retries transactions and the closure can run more than once). `Collect` sends every alert matching the gate, then calls `MarkNotified` with the IDs that delivered. If marking fails, the alert simply re-delivers next run; the inverse ordering would risk recording a delivery that never happened, which is the worse failure for an alerting system.

`MarkNotified` runs its own transaction, matches alerts by ID so a concurrent run cannot be clobbered, and updates only the `alerts` field — `points` carries 72 forecast hours and there is no reason to rewrite it. It never touches `Status`.

**No in-process retry.** A failed send leaves `NotifiedAt` zero, so the 6-hour cron *is* the retry — and unlike an in-process loop it survives the job being killed. A delivery failure for one location is logged at `slog.Error` and does not fail the run.

> **Accepted limitation:** a persistently unreachable SMTP endpoint degrades silently — the job keeps reporting success while no alert arrives. There is no alerting on the alerter. Deliberate for a personal project.

> **No quiet hours.** Nothing defers an overnight alert; a 3 AM `severe` drop sends a 3 AM email. Phone-level Do Not Disturb is the mitigation.

## 6. External API

Google Weather API `forecast/hours:lookup`.

*   **Pagination**: Fetched in pages of 24 hours, following `nextPageToken` until the horizon is covered or no further pages exist.
*   **Retry**: Three retries with 1s/2s/4s backoff. `429` and `5xx` are retried; other `4xx` and JSON decode failures are wrapped as non-retryable and fail immediately.
*   **Auth**: API key passed via the `X-Goog-Api-Key` header, never in the query string.

## 7. Failure Handling

The orchestration loop is **partial-failure tolerant**: each location is collected independently, a failure is logged and skipped, and the run only exits non-zero when *every* location fails. A location that fetches successfully but maps to zero valid points is treated as a failure for that location.

## 8. Monitored Locations

Shared with the other collectors via `shared.Locations`.

| Location ID | Latitude | Longitude |
|------------|----------|-----------|
| `house-nick` | 30.2605 | -97.6677 |
| `house-nita` | 30.2942 | -97.6959 |
| `distribution-hall` | 30.2619 | -97.7282 |
