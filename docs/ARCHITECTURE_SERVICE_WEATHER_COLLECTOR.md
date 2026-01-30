# Weather Service Architecture & Implementation Plan

## 1. Overview
The **Weather Service** is a subsystem of the Personal Dashboard. It separates data collection (background job) from data serving (API).

For platform-level details (Deployment, Terraform, Identity), see **[ARCHITECTURE_INFRASTRUCTURE.md](./ARCHITECTURE_INFRASTRUCTURE.md)**.

## 2. System Architecture

### Components
1.  **Weather Collector (`services/weather-collector`)**
    *   **Role**: Background Worker (Writer).
    *   **Runtime**: Cloud Run Job.
    *   **Trigger**: Cloud Scheduler (Hourly Cron).
    *   **Responsibility**:
        *   Fetch weather data from external API (Google Weather/Maps).
        *   Process and format data.
        *   Perform "Dual-Write" to Firestore (Archive + Cache).

2.  **Weather Provider (`services/weather-provider`)** *[Planned]*
    *   **Role**: API Service (Reader).
    *   **Runtime**: Cloud Run Service (gRPC).
    *   **Responsibility**:
        *   Read pre-aggregated data from Firestore `weather_cache`.
        *   Serve data to the SvelteKit Frontend.

3.  **Frontend (`frontend/`)**
    *   **Framework**: SvelteKit.
    *   **Responsibility**: Display current weather and 24h trends.

## 3. Data Strategy (Firestore)

We utilize a **Dual-Write** strategy to balance historical accuracy with dashboard loading speed.

### Collection 1: `weather_raw` (The Archive)
*   **Purpose**: Long-term history logging. Flat structure for easy future export to BigQuery/SQL.
*   **Write Frequency**: 1 document per location per hour.
*   **Schema**:
    ```json
    {
      "location": "string",
      "timestamp": "timestamp",
      "temp_c": "float64",
      "temp_feel_c": "float64",
      "humidity": "int",
      "uv_index": "int",
      "pressure_mb": "float64",
      "wind_dir_deg": "int",
      "wind_speed_kph": "float64",
      "wind_gust_kph": "float64",
      "visibility_km": "float64"
    }
    ```

### Collection 2: `weather_cache` (The Dashboard View)
*   **Purpose**: Instant read-optimized view for the frontend.
*   **Structure**: 1 Document per `Location ID` (e.g., `house-nick`).
*   **Schema**:
    ```json
    {
      "last_updated": "timestamp",
      "current": {
        // Snapshot of conditions (same fields as raw)
      },
      "history": [
        // Array of last 48 simplified data points
        {
          "ts": "time",
          "t": 22.5,
          "tf": 21.0,
          "h": 60,
          "uv": 4,
          "p": 1013.2,
          "wd": 180,
          "ws": 14.0,
          "wg": 25.0,
          "v": 16.0
        }
      ]
    }
    ```

## 4. Implementation Details

### Weather Collector Logic (`main.go`)
1.  **Initialize**: Load config (Locations), init Firestore client.
2.  **Iterate**: Loop through target locations.
3.  **Fetch (Robust)**: Call Google Weather API with **Retry Policy**.
    *   Attempts: 3 max.
    *   Backoff: 1s, 2s, 4s (Exponential).
    *   **Failure Handling**: If all retries fail, log a structured error (Severity: ERROR) to trigger **GCP Error Reporting**.
4.  **Calculate Stats**: Compute pressure trends (1h, 3h, 6h, 12h, 24h).
    *   **Algorithm**: Timestamp-based Time-Window Search.
    *   **Logic**: Find historical point `P` where `abs(P.Timestamp - (Now - DeltaHours)) < Tolerance`.
    *   **Tolerance**: +/- 45 minutes.
    *   **Fallback**: If no point found within tolerance, Delta = 0.0 (Stable).
    *   **Trend Definition**: The string "Trend" (Rising/Falling/Stable) is derived **exclusively** from the **3-Hour Delta**. This adheres to the World Meteorological Organization (WMO) standard for "Barometric Tendency". Other deltas (1h, 24h) are provided as raw values for context but are not used to label the official trend.
5.  **Write Raw**: Insert new document into `weather_raw`.
6.  **Update Cache**: Use a Firestore Transaction to:
    *   Read existing cache doc for the location.
    *   Append new point to `history` array.
    *   Truncate `history` to keep only the last 48 entries (24h buffer).
    *   Update `current` fields, `analysis` stats, and `last_updated`.

### Configuration
Locations are currently hardcoded in Go but tracked as:
*   `house-nick` (Lat: 30.2605, Long: -97.6677)
*   `house-nita` (Lat: 30.2942, Long: -97.6959)
*   `distribution-hall` (Lat: 30.2619, Long: -97.7282)

## 6. Reliability & Data Quality

### Input Strategy: Robust Acquisition
To ensure high data availability without excessive costs:
*   **Retry with Backoff**: The collector implements a retry loop (3 attempts, exponential backoff) to handle transient API or network failures.
*   **Observability**: Critical failures (after retries exhausted) are logged with structured payloads to integrate with **Google Cloud Error Reporting**, enabling active alerting on persistent outages.

### Output Strategy: Resilient Calculation
To ensure accurate trends despite potential data gaps or jitter:
*   **Time-Window Search**: Delta calculations (e.g., "Change over 3 hours") do not rely on fixed array indices. Instead, they search the history buffer for the data point with a timestamp closest to the target time (e.g., `Now - 3h`).
*   **Tolerance**: A search window (e.g., +/- 45 mins) allows for job scheduling jitter or missed cycles while maintaining statistical relevance.

## 7. Next Steps
1.  **Protocol Definitions**: Create `protos/weather/v1/weather.proto`.
2.  **Provider Service**: Scaffold `services/weather-provider`.
3.  **Frontend Integration**: Connect SvelteKit to the backend.
