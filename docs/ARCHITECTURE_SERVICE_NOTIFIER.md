# Notifier Service Architecture

## 1. Overview

The **Notifier** (`services/notifier`) is an hourly background job that reads the weather and forecast caches and **records what it sees**. It sends nothing.

That is deliberate, and it is the whole point of the service. Alert *delivery* lives in the **Forecast Collector**, which runs every 6 hours and mails an alert on first detection — so an episode forecast for Thursday is mailed on Monday. Fixing that needs two decisions: *when* an alert is close enough to be worth sending ([#79](https://github.com/nickfang/personal-dashboard/issues/79)), and *what* the send threshold should measure ([#80](https://github.com/nickfang/personal-dashboard/issues/80)). Both were designed once and the design did not survive review — the proposed predicate inverted in its own motivating case, and its threshold compared a 3-hour detection figure against an excursion spanning 15+ hours.

Rather than ship a gate that could not be defended, this job ships first and gathers the evidence those decisions need. It holds no credentials, so sending is not merely switched off but structurally unavailable.

For platform-level details (Deployment, Terraform, Identity), see **[ARCHITECTURE_INFRASTRUCTURE.md](./ARCHITECTURE_INFRASTRUCTURE.md)**.

## 2. System Architecture

*   **Role**: Background Worker (Reader).
*   **Runtime**: Cloud Run Job.
*   **Trigger**: Cloud Scheduler, `15 * * * *` — hourly, offset past the Weather Collector's `0 * * * *` so the observation it reads is fresh.
*   **Architecture**: Layered (Service → Repository), matching the collectors minus the API layer.

```text
services/notifier/
├── cmd/
│   ├── main.go                 # Wiring + observeAll() loop
│   └── main_test.go            # Partial failure, all-fail, empty, pinned-now
├── internal/
│   ├── service/
│   │   ├── observe.go          # Observation + BuildObservation() (pure)
│   │   ├── observe_test.go
│   │   ├── notifier.go         # NotifierService orchestration + logging
│   │   └── notifier_test.go
│   ├── repository/
│   │   ├── store.go            # Store interface (no writes) + FirestoreStore
│   │   └── types.go            # Mirror structs for the two cache documents
│   └── testutil/
│       └── mocks.go            # MockStore
├── Dockerfile
└── go.mod
```

**There is no `internal/api/` package.** Every other collector has one; this job makes no external calls, which is a design property rather than an omission. It also carries no `tzdata` in its image for the same reason — nothing here renders local time.

## 3. What it reads

Both collections live in the `weather-log` database, so a single Firestore client serves both.

| Collection | Owner | Read |
|---|---|---|
| `weather_cache/{locationID}` | Weather Collector | `current.pressure_mb`, `current.timestamp` |
| `forecast_cache/{locationID}` | Forecast Collector | `issued_at`, `points[]`, `alerts[]` |

`internal/repository/types.go` declares **mirror structs** carrying only these fields — Firestore ignores document fields absent from the destination struct. The Weather Provider does the same thing for the same reason. The cost is that an upstream shape change fails at decode time rather than compile time, which is why the mirrors are kept minimal.

> **The read-only property is enforced in code, not in IAM.** The `Store` interface exposes no writes, but the shared `cloud-run-job` module grants every job service account project-wide `roles/datastore.user`, which includes write. Nothing at the infrastructure layer stops this job from modifying documents owned by the collectors. Narrowing it to `roles/datastore.viewer` needs a variable on that shared module.

A **missing observation is recorded, not fatal**: the Weather Collector may simply not have run for that location, and knowing that is itself a finding. A missing *forecast* is an error, since there is nothing to observe against. A read *failure* is distinct from an absent document and always fails the location.

## 4. What it records

One `observation` log line per location per hour, plus one `alert seen` line per alert in the cache.

```
observation  location=house-nick observed_mb=1015.4 observed_age_min=12
             forecast_issued_at=... forecast_age_min=214
             forecast_at_observed_mb=1017.9 error_mb=-2.5
             fwd_03h=-5.8 fwd_03h_matched=... fwd_06h=-8.1 fwd_12h=-8.4 fwd_24h=-4.2

alert seen   location=house-nick alert=abc123 rule=pressure-drop-3h
             severity=warning status=active value=-6.2 notified=false
             hours_to_window=11.75
```

Five choices worth knowing:

*   **Deltas anchor on the observed reading**, not on the forecast's own first point. The existing display paths (`dashboard-api/internal/handlers/format.go`, the CLI TUI) compute forecast-to-forecast deltas; these are observed-to-forecast, which is the quantity a person actually experiences.
*   **`error_mb` is the headline number.** It is the observed pressure minus what the forecast predicted **for the moment the barometer was read** — the forecast error accumulated since issue. Sampling the forecast at *now* instead would be wrong whenever the Weather Collector has skipped a location, since `weather_cache` retains the previous document and the gap's real pressure change would be folded into a field labelled forecast error. Whether that error is routinely large is precisely what decides if an observed-anchored gate is worth building at all (#80).
*   **Forward deltas anchor their target on `now`**, not on the observation — the question is what happens over the next N hours from here. A stale observation therefore widens the true interval, which is what `observed_age_min` is for.
*   **`fwd_NNh_matched` records which point was used.** Tolerance matching accepts a point up to 45 minutes from the requested offset, so a delta labelled "+3h" may be measured over 2h15m. Nothing downstream could recover that.
*   **It records facts, not verdicts.** There is deliberately no `would_send` field. Computing one would bake in the predicate this service exists to defer.

The forward search uses `shared.NearestIndex`, a generic tolerance search added with this service. It is **not** yet used by the four existing implementations of the same search — see [#80](https://github.com/nickfang/personal-dashboard/issues/80), since `weather-collector`'s version scans descending and resolves ties in the opposite direction.

## 5. Configuration

| Env Var | Default | Meaning |
|---------|---------|---------|
| `GCP_PROJECT_ID` | *(required)* | Firestore project |
| `DEBUG` | `false` | Debug-level logging via `shared.InitLogging()` |

No API key, no SMTP credentials, no thresholds. Every offset and tolerance is a constant in `observe.go` — there is nothing to tune until there is a decision to tune it against.

## 6. Failure Handling

`observeAll` is **partial-failure tolerant**, matching the collectors: a location that cannot be read is logged and skipped, and the run exits non-zero only when *every* location fails.

The evaluation time is **pinned once in `main`** and threaded through every location, so a run that straddles an hour boundary cannot split across two evaluations.

> **No monitoring yet.** There are no Cloud Monitoring resources in this project, so a job that stops firing is noticed by its logs going quiet. An absence alert on `run.googleapis.com/job/completed_task_attempt_count` is the natural next step and would cover all four jobs, not just this one.

## 7. What this service will become

Once #79 and #80 are settled, the natural shape is for delivery to move here: `NotifierService.Observe` already returns the `Observation` it builds rather than discarding it, precisely so a gate can be layered on without restructuring. That move must also relocate `MarkNotified` and remove `deliver()` from the Forecast Collector **in a single change** — any ordering with both live sends two emails per alert, and the reverse leaves a delivery gap.

Whether that move happens at all is an open question. `ARCHITECTURE_SERVICE_FORECAST_COLLECTOR.md` §5 records a deliberate decision *against* a separate delivery service for this workload, and that reasoning has not been rebutted — only deferred.
