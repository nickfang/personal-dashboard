#!/usr/bin/env python3
"""Replay pressure-drop detection over historical forecasts and verify it.

Read-only. Answers, from data already in weather_raw and forecast_raw:

  1. When would alerts have fired, with what lead time?  -> episodes.csv
  2. How close was the forecast to what actually happened, by lead time?
     -> verification.csv
  3. How often do alerts fire, and do the three locations triplicate them?
  4. What would other (window, threshold) rules have fired?  -> sweep table

!! KEEP IN SYNC WITH THE GO DETECTOR !!

The detector is reimplemented here to match
services/forecast-collector/internal/service/detect.go, including its
30-minute window tolerance (the display paths use 45; detection does not).
Episodes are then merged across successive forecast runs the way
shared.MergeAlerts does, by window overlap, so "first detected" is the run
that would actually have sent the alert.

That duplication is deliberate — this has to run over historical data outside
the service — but nothing enforces it. If detect.go changes, this silently
reports what the OLD detector would have done. Issue #80 will add rise
detection; update `detect()` and the constants below in the same PR. Nothing
in CI covers this file.

A note on verification: at 0-1h lead the observation and the forecast are the
same Google analysis, so agreement there is ~0 by construction. It becomes a
real skill measure at longer leads, where the forecast predates the
observations that went into the verifying analysis.

Usage:
    python3 scripts/pressure_analysis.py [project] [start] [end] [options]

    python3 scripts/pressure_analysis.py
    python3 scripts/pressure_analysis.py fang-gcp-staging 2026-01-01 2026-03-31
    python3 scripts/pressure_analysis.py fang-gcp 2026-01-01 2026-03-31 --out-dir /tmp/winter

Options:
    --no-verify      skip verification.csv (the row-per-forecast-point file)
    --out-dir DIR    where CSVs are written (default: current directory)
    --refresh        ignore the cache and refetch

Completed date blocks are cached under $XDG_CACHE_HOME/pressure-analysis
(default ~/.cache/pressure-analysis), since the raw collections are
append-only and never change once written. Only the block containing the
present is refetched. The cache is disposable — delete the directory to reset.

Defaults to staging. Pass fang-gcp explicitly to read production.
Needs only `gcloud` on PATH. No Python packages.
"""
import bisect
import csv
import json
import os
import subprocess
import sys
from collections import defaultdict
from datetime import datetime, timedelta, timezone
from statistics import median

# --- mirrors services/forecast-collector/internal/service/detect.go ---
WINDOW_HOURS = 3
DROP_THRESHOLD_MB = 5.0
SEVERE_THRESHOLD_MB = 10.0
WINDOW_TOLERANCE = timedelta(minutes=30)   # detect.go:13
OBS_TOLERANCE = timedelta(minutes=45)      # matching an observation to a time

args, flags = [], {}
it = iter(sys.argv[1:])
for a in it:
    if a == "--out-dir":
        flags["out_dir"] = next(it, ".")
    elif a.startswith("--"):
        flags[a] = True
    else:
        args.append(a)

if "--help" in flags or "-h" in flags:
    print(__doc__)
    sys.exit(0)
for k in flags:
    if k not in ("--no-verify", "--refresh", "out_dir"):
        sys.exit(f"Unknown option: {k}\nTry --help")

PROJECT = args[0] if args else "fang-gcp-staging"
DB = "weather-log"
BASE = f"https://firestore.googleapis.com/v1/projects/{PROJECT}/databases/{DB}/documents"
OUT_DIR = flags.get("out_dir", ".")
SKIP_VERIFY = "--no-verify" in flags
REFRESH = "--refresh" in flags
# XDG Base Directory spec: $XDG_CACHE_HOME, else ~/.cache.
CACHE_DIR = os.path.join(
    os.environ.get("XDG_CACHE_HOME") or os.path.expanduser("~/.cache"),
    "pressure-analysis")
# Blocks are aligned to a fixed epoch, not to the requested window, so
# overlapping date ranges reuse the same cache entries.
CACHE_EPOCH = datetime(2026, 1, 1, tzinfo=timezone.utc)


def parse_day(s, end=False):
    d = datetime.strptime(s, "%Y-%m-%d").replace(tzinfo=timezone.utc)
    return d + timedelta(days=1) if end else d


END = parse_day(args[2], end=True) if len(args) > 2 else datetime.now(timezone.utc)
START = parse_day(args[1]) if len(args) > 1 else END - timedelta(days=45)


def token():
    r = subprocess.run(["gcloud", "auth", "print-access-token"], capture_output=True, text=True)
    if r.returncode != 0:
        sys.exit("Not authenticated. Run: gcloud auth login")
    return r.stdout.strip()


TOKEN = token()


def query_range(collection, field, lo, hi):
    body = {"structuredQuery": {
        "from": [{"collectionId": collection}],
        "where": {"compositeFilter": {"op": "AND", "filters": [
            {"fieldFilter": {"field": {"fieldPath": field}, "op": "GREATER_THAN_OR_EQUAL",
                             "value": {"timestampValue": lo.strftime("%Y-%m-%dT%H:%M:%SZ")}}},
            {"fieldFilter": {"field": {"fieldPath": field}, "op": "LESS_THAN",
                             "value": {"timestampValue": hi.strftime("%Y-%m-%dT%H:%M:%SZ")}}},
        ]}},
        "orderBy": [{"field": {"fieldPath": field}, "direction": "ASCENDING"}],
    }}
    r = subprocess.run(
        ["curl", "-s", "-X", "POST", f"{BASE}:runQuery",
         "-H", f"Authorization: Bearer {TOKEN}",
         "-H", "Content-Type: application/json", "-d", json.dumps(body)],
        capture_output=True, text=True)
    try:
        rows = json.loads(r.stdout)
    except json.JSONDecodeError:
        sys.exit(f"Unexpected response for {collection}:\n{r.stdout[:400]}")
    if isinstance(rows, dict) and "error" in rows:
        sys.exit(f"Firestore error on {collection}: {rows['error'].get('message')}")
    return [x["document"] for x in rows if "document" in x]


def ts(v):
    return datetime.fromisoformat(v.replace("Z", "+00:00"))


def blocks(start, end, slice_len):
    """Fixed-boundary blocks covering [start, end), aligned to CACHE_EPOCH."""
    b = CACHE_EPOCH + ((start - CACHE_EPOCH) // slice_len) * slice_len
    while b < end:
        yield b, b + slice_len
        b += slice_len


def fetch(collection, field, slice_len):
    """Fetch a window, caching blocks that are entirely in the past.

    The raw collections are append-only, so a completed block never changes.
    Only the block containing 'now' is refetched each run.
    """
    now = datetime.now(timezone.utc)
    out, hits, misses = [], 0, 0
    for b0, b1 in blocks(START, END, slice_len):
        path = os.path.join(CACHE_DIR, PROJECT, collection,
                            f"{b0:%Y%m%d}-{slice_len.days}d.json")
        settled = b1 <= now
        if settled and not REFRESH and os.path.exists(path):
            with open(path) as fh:
                docs = json.load(fh)
            hits += 1
        else:
            docs = query_range(collection, field, b0, b1)
            misses += 1
            if settled:
                os.makedirs(os.path.dirname(path), exist_ok=True)
                with open(path, "w") as fh:
                    json.dump(docs, fh)
        out += docs
        print(f"\r  {collection}: {len(out)} docs through {b1:%Y-%m-%d}",
              end="", file=sys.stderr)
    # Blocks overhang the requested window; trim to it.
    out = [d for d in out
           if START <= ts(d["fields"][field]["timestampValue"]) < END]
    print(f"\r  {collection}: {len(out)} docs  "
          f"({hits} cached, {misses} fetched)          ", file=sys.stderr)
    return out


def num(f):
    if "doubleValue" in f:
        return float(f["doubleValue"])
    if "integerValue" in f:
        return float(f["integerValue"])
    return None


def pct(v, p):
    return v[min(int(len(v) * p), len(v) - 1)]


banner = "PRODUCTION" if PROJECT == "fang-gcp" else "staging"
print(f"project {PROJECT} [{banner}], database {DB}", file=sys.stderr)
print(f"window  {START:%Y-%m-%d} .. {END:%Y-%m-%d}  ({(END-START).days} days)", file=sys.stderr)
obs_docs = fetch("weather_raw", "timestamp", timedelta(days=10))
fc_docs = fetch("forecast_raw", "issued_at", timedelta(days=5))
if not obs_docs:
    sys.exit("No observations in that window. Check the dates and the project.")
if not fc_docs:
    # forecast-collector shipped after weather-collector, so windows before it
    # existed have observations only. The threshold question is answerable from
    # those alone; the episode replay is not.
    print("\n*** No forecast runs in this window — observation-only mode. ***\n"
          "    Episode replay and verification need forecast_raw.", file=sys.stderr)

# --- observations ---
obs = defaultdict(list)
for d in obs_docs:
    f = d["fields"]
    p = num(f.get("pressure_mb", {}))
    if p:
        obs[f["location"]["stringValue"]].append((ts(f["timestamp"]["timestampValue"]), p))
for loc in obs:
    obs[loc].sort()
keys = {loc: [t for t, _ in s] for loc, s in obs.items()}


def nearest(loc, target, tol=OBS_TOLERANCE):
    if loc not in obs:
        return None
    series, ks = obs[loc], keys[loc]
    i = bisect.bisect_left(ks, target)
    best, diff = None, tol
    for j in (i - 1, i):
        if 0 <= j < len(series):
            d = abs(series[j][0] - target)
            if d <= diff:
                best, diff = series[j][1], d
    return best


def steepest_observed(loc, start, end, hours=WINDOW_HOURS):
    """Most negative observed change over any `hours` window inside [start,end]."""
    if loc not in obs:
        return None
    worst = None
    for t, p in obs[loc]:
        if t < start - OBS_TOLERANCE or t > end:
            continue
        p2 = nearest(loc, t + timedelta(hours=hours))
        if p2 is None:
            continue
        d = p2 - p
        if worst is None or d < worst:
            worst = d
    return worst


# --- detector replay, faithful to detect.go ---
def point_near_offset(points, i, offset, tol=WINDOW_TOLERANCE):
    target = points[i][0] + offset
    best, mindiff = -1, tol + timedelta(seconds=1)
    for j in range(i + 1, len(points)):
        diff = abs(points[j][0] - target)
        if diff <= tol and diff < mindiff:
            mindiff, best = diff, j
        if points[j][0] - target > tol:
            break
    return best


def detect(points, window_hours=WINDOW_HOURS, threshold=DROP_THRESHOLD_MB,
           severe=SEVERE_THRESHOLD_MB, tol=WINDOW_TOLERANCE, direction="drop"):
    """points: [(valid_time, pressure_mb)] ascending. -> [(start,end,value,severity)]

    Faithful to detect.go when called with the defaults, which include
    direction="drop" — the shipped detector is drop-only. Parameterized so the
    sweep can ask what other windows, thresholds, and directions would have
    produced. Values keep their sign, so a rise episode reports positive.
    """
    if len(points) < 2:
        return []
    wins = []
    for i in range(len(points)):
        j = point_near_offset(points, i, timedelta(hours=window_hours), tol)
        if j < 0:
            continue
        delta = points[j][1] - points[i][1]
        if (delta <= -threshold) if direction == "drop" else (delta >= threshold):
            wins.append((i, j, delta))
    if not wins:
        return []

    out, episode = [], [wins[0]]

    def build(ep):
        steep = (min if direction == "drop" else max)(ep, key=lambda w: w[2])
        sev = "severe" if abs(steep[2]) >= severe else "warning"
        return (points[ep[0][0]][0], points[ep[-1][1]][0], steep[2], sev)

    for w in wins[1:]:
        last = episode[-1]
        if not points[w[0]][0] > points[last[1]][0]:
            episode.append(w)
        else:
            out.append(build(episode))
            episode = [w]
    out.append(build(episode))
    return out


# --- replay every run, merging episodes across runs by overlap ---
runs = []
for d in fc_docs:
    f = d["fields"]
    pts = []
    for pt in f.get("points", {}).get("arrayValue", {}).get("values", []):
        pf = pt["mapValue"]["fields"]
        p = num(pf.get("pressure_mb", {}))
        if p:
            pts.append((ts(pf["valid_time"]["timestampValue"]), p))
    pts.sort()
    runs.append((ts(f["issued_at"]["timestampValue"]), f["location"]["stringValue"], pts))
runs.sort()

live = defaultdict(list)   # location -> [episode dict]
episodes = []
for issued, loc, pts in runs:
    live[loc] = [e for e in live[loc] if e["window_end"] > issued]
    for start, end, value, sev in detect(pts):
        match = None
        for e in live[loc]:
            if start < e["window_end"] and e["window_start"] < end:
                match = e
                break
        if match:
            match["window_start"], match["window_end"] = start, end
            match["last_seen"] = issued
            match["runs"] += 1
            if value < match["worst_value"]:
                match["worst_value"], match["worst_severity"] = value, sev
        else:
            e = {"location": loc, "first_detected": issued, "last_seen": issued,
                 "window_start": start, "window_end": end, "runs": 1,
                 "first_value": value, "worst_value": value,
                 "first_severity": sev, "worst_severity": sev}
            live[loc].append(e)
            episodes.append(e)

# --- verify each episode against what was observed ---
now = datetime.now(timezone.utc)
for e in episodes:
    e["lead_hours"] = round((e["window_start"] - e["first_detected"]).total_seconds() / 3600, 1)
    if e["window_end"] > now:
        e["observed_value"], e["verdict"] = None, "pending"
        continue
    o = steepest_observed(e["location"], e["window_start"], e["window_end"])
    e["observed_value"] = None if o is None else round(o, 2)
    if o is None:
        e["verdict"] = "no-data"
    elif o <= -DROP_THRESHOLD_MB:
        e["verdict"] = "verified"
    elif o <= -DROP_THRESHOLD_MB / 2:
        e["verdict"] = "partial"
    else:
        e["verdict"] = "bust"

os.makedirs(OUT_DIR, exist_ok=True)
ep_path = os.path.join(OUT_DIR, "episodes.csv")
with open(ep_path, "w", newline="") as fh:
    w = csv.writer(fh)
    w.writerow(["location", "first_detected", "last_seen", "window_start", "window_end",
                "lead_hours", "runs_seen", "first_predicted_mb", "worst_predicted_mb",
                "severity", "observed_steepest_mb", "verdict"])
    for e in sorted(episodes, key=lambda x: x["first_detected"]):
        w.writerow([e["location"], e["first_detected"].isoformat(), e["last_seen"].isoformat(),
                    e["window_start"].isoformat(), e["window_end"].isoformat(),
                    e["lead_hours"], e["runs"], round(e["first_value"], 2),
                    round(e["worst_value"], 2), e["worst_severity"],
                    e["observed_value"], e["verdict"]])

# --- verification.csv: one row per forecast point that has an observation ---
ver_rows = 0
skill = defaultdict(list)
if not SKIP_VERIFY:
    v_path = os.path.join(OUT_DIR, "verification.csv")
    with open(v_path, "w", newline="") as fh:
        w = csv.writer(fh)
        w.writerow(["location", "issued_at", "valid_time", "lead_hours",
                    "predicted_mb", "observed_mb", "error_mb"])
        for issued, loc, pts in runs:
            for vt, pred in pts:
                o = nearest(loc, vt)
                if o is None:
                    continue
                lead = (vt - issued).total_seconds() / 3600
                w.writerow([loc, issued.isoformat(), vt.isoformat(), round(lead, 1),
                            round(pred, 2), round(o, 2), round(o - pred, 3)])
                ver_rows += 1
                skill[int(lead // 6) * 6].append(abs(o - pred))

# ============================ terminal summary ============================
lo = min(s[0][0] for s in obs.values() if s)
hi = max(s[-1][0] for s in obs.values() if s)
days = max((hi - lo).total_seconds() / 86400, 1)
print(f"\nObservations {lo:%Y-%m-%d} .. {hi:%Y-%m-%d}  ({len(obs_docs)} readings, "
      f"{len(runs)} forecast runs)")

print("\n=== How fast pressure actually moves (observed) ===")
rate_totals = {}
for hours in (1, 3, 6, 12):
    absolute = []
    for loc, series in obs.items():
        for t, p in series:
            p2 = nearest(loc, t + timedelta(hours=hours))
            if p2 is not None:
                absolute.append(abs(p2 - p))
    if absolute:
        absolute.sort()
        rate_totals[hours] = absolute
        print(f"  over {hours:2d}h   n={len(absolute):6d}   median {median(absolute):5.2f}   "
              f"p90 {pct(absolute,0.9):5.2f}   p99 {pct(absolute,0.99):5.2f}   "
              f"max {absolute[-1]:5.2f}  mb total")

if 3 in rate_totals:
    a3 = rate_totals[3]
    print("\n  Observed 3-hour windows crossing each threshold:")
    for name, mb in (("warning   5 mb", 5.0), ("severe   10 mb", 10.0)):
        n = sum(1 for x in a3 if x >= mb)
        print(f"    {name}   {n:6d} / {len(a3)}  ({100*n/len(a3):6.3f}%)")
    print(f"    largest observed 3h swing: {a3[-1]:.2f} mb")
    print("    (this needs no forecast data, so it works over the full weather_raw history)")

print(f"\n=== Alerts that would have fired ===")
by_sev = defaultdict(int)
for e in episodes:
    by_sev[e["worst_severity"]] += 1
if not fc_docs:
    print("  (skipped — no forecast runs in this window)")
print(f"  {len(episodes)} episodes over {days:.0f} days "
      f"= {len(episodes)/days*7:.1f} per week")
for s in ("warning", "severe"):
    print(f"    {s:8s} {by_sev[s]:4d}")
if episodes:
    leads = sorted(e["lead_hours"] for e in episodes)
    print(f"  lead time at first detection: median {median(leads):.1f}h  "
          f"min {leads[0]:.1f}h  max {leads[-1]:.1f}h")
    print(f"  (NOTIFY_LEAD_HOURS below the min would never fire; above the max "
          f"changes nothing)")
    v = defaultdict(int)
    for e in episodes:
        v[e["verdict"]] += 1
    print("  verdicts: " + "  ".join(f"{k}={n}" for k, n in sorted(v.items())))

# --- threshold/window sweep over observations ---
SWEEP_WINDOWS = (3, 6, 12, 24)
SWEEP_THRESHOLDS = (3, 4, 5, 6, 8, 10)


def distinct_events(window_hours, threshold, direction="drop"):
    """Episodes across all locations, collapsed where they overlap in time.

    The three locations sit within 0.03 mb of each other, so an episode at one
    is the same weather as an episode at another. Collapsing overlaps counts
    what you would actually be told about, assuming any dedup at all.
    """
    spans = []
    for loc, series in obs.items():
        for start, end, _, _ in detect(series, window_hours, threshold,
                                       severe=threshold * 2, tol=OBS_TOLERANCE,
                                       direction=direction):
            spans.append((start, end))
    if not spans:
        return 0
    spans.sort()
    merged, cur_e = 0, spans[0][1]
    for s_, e_ in spans[1:]:
        if s_ <= cur_e:
            cur_e = max(cur_e, e_)
        else:
            merged += 1
            cur_e = e_
    return merged + 1


months = days / 30.44
print(f"\n=== What each rule would fire: distinct events per month ===")
print("    Run over OBSERVATIONS — what the weather actually did, not what was")
print("    forecast. Locations collapsed where episodes overlap in time.")
print(f"    Sample: {days:.0f} days = {months:.1f} months")

for direction in ("drop", "rise"):
    label = "DROPS (what the detector sees today)" if direction == "drop" \
        else "RISES (invisible to the detector today)"
    print(f"\n  {label}")
    header = "  window  " + "".join(f"{t:>7} mb" for t in SWEEP_THRESHOLDS)
    print(header)
    print("  " + "-" * (len(header) - 2))
    for wh in SWEEP_WINDOWS:
        row = f"  {wh:>4}h   "
        for th in SWEEP_THRESHOLDS:
            n = distinct_events(wh, th, direction)
            row += f"{n/months:>7.1f}   " if n else f"{'.':>7}   "
        print(row)

print("\n  '.' means it never fired. Current setting is 3h / 5 mb, drops only.")
print("  Compare the two tables: if rises are systematically larger, the")
print("  thresholds should differ by direction rather than mirror each other.")

print("\n=== Do the three locations duplicate each other? ===")
locs = sorted(obs)
for i in range(len(locs)):
    for j in range(i + 1, len(locs)):
        a, b = locs[i], locs[j]
        pairs = [(p, nearest(b, t)) for t, p in obs[a]]
        pairs = [(x, y) for x, y in pairs if y is not None]
        if not pairs:
            continue
        diffs = sorted(abs(x - y) for x, y in pairs)
        ident = sum(1 for d in diffs if d == 0) / len(diffs) * 100
        print(f"  {a:18s} vs {b:18s}  median |diff| {median(diffs):.2f} mb   "
              f"identical {ident:5.1f}%")

if skill:
    print("\n=== Forecast skill: |observed - predicted| by lead ===")
    print("    (0-6h is near-tautological; longer leads are the real signal)")
    for bucket in sorted(skill):
        v = sorted(skill[bucket])
        print(f"  {bucket:3d}-{bucket+6:<3d}h  n={len(v):6d}  median {median(v):5.2f}  "
              f"p90 {pct(v,0.9):5.2f}  max {v[-1]:5.2f}  mb")

print(f"\nWrote {ep_path} ({len(episodes)} rows)")
if not SKIP_VERIFY:
    print(f"Wrote {os.path.join(OUT_DIR, 'verification.csv')} ({ver_rows} rows)")
