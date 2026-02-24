# Weather Collector Service Architecture

## 1. Overview
The **Weather Collector** (`services/weather-collector`) is a background worker that fetches weather data from the Google Weather API and writes to Firestore. It separates data collection from data serving — the **Weather Provider** reads from the cache to serve the Dashboard API.

For platform-level details (Deployment, Terraform, Identity), see **[ARCHITECTURE_INFRASTRUCTURE.md](./ARCHITECTURE_INFRASTRUCTURE.md)**.

## 2. System Architecture

### Components
1.  **Weather Collector (`services/weather-collector`)**
    *   **Role**: Background Worker (Writer).
    *   **Runtime**: Cloud Run Job.
    *   **Trigger**: Cloud Scheduler (Hourly Cron).
    *   **Responsibility**:
        *   Fetch weather data from external API (Google Weather/Maps).
        *   Calculate pressure deltas and barometric trend.
        *   Perform "Dual-Write" to Firestore (Archive + Cache).

2.  **Weather Provider (`services/weather-provider`)**
    *   **Role**: API Service (Reader).
    *   **Runtime**: Cloud Run Service (gRPC, :50051).
    *   **Responsibility**:
        *   Read pre-aggregated pressure data from Firestore `weather_cache`.
        *   Serve pressure stats to the Dashboard API via gRPC.

## 3. Data Strategy (Firestore)

We utilize a **Dual-Write** strategy to balance historical accuracy with dashboard loading speed.

### Collection 1: `weather_raw` (The Archive)
*   **Purpose**: Long-term history logging. Flat structure for easy future export to BigQuery/SQL.
*   **Write Frequency**: 1 document per location per hour.
*   **Schema** (`WeatherPoint`):
    ```json
    {
      "location": "string",
      "timestamp": "timestamp",
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
      "visibility_km": "float64",
      "temp_f": "float64",
      "temp_feel_f": "float64",
      "dewpoint_f": "float64",
      "wind_speed_mph": "float64",
      "wind_gust_mph": "float64",
      "visibility_miles": "float64"
    }
    ```

### Collection 2: `weather_cache` (The Pressure Analysis View)
*   **Purpose**: Read-optimized pressure analysis for the dashboard. Contains current conditions, pressure history, and computed deltas/trend.
*   **Structure**: 1 Document per Location ID (e.g., `house-nick`).
*   **Schema** (`CacheDoc`):
    ```json
    {
      "last_updated": "timestamp",
      "current": {
        "// Full WeatherPoint snapshot (same fields as weather_raw)"
      },
      "analysis": {
        "timestamp": "timestamp",
        "delta_01h": "float64 | null",
        "delta_03h": "float64 | null",
        "delta_06h": "float64 | null",
        "delta_12h": "float64 | null",
        "delta_24h": "float64 | null",
        "trend": "string (Rising | Falling | Stable)"
      },
      "history": [
        {
          "// Last 48 PressurePoint entries (pressure, temp, humidity, dewpoint)"
          "timestamp": "timestamp",
          "pressure_mb": "float64",
          "humidity_pct": "int",
          "temp_c": "float64",
          "temp_feel_c": "float64",
          "dewpoint_c": "float64",
          "temp_f": "float64",
          "temp_feel_f": "float64",
          "dewpoint_f": "float64"
        }
      ]
    }
    ```

## 4. Pressure Analysis

The collector computes barometric pressure deltas and trend on each run.

*   **Deltas**: Change in pressure over 1h, 3h, 6h, 12h, and 24h windows. Uses a timestamp-based search with +/- 45 minute tolerance to handle scheduling jitter. Delta fields are nullable (`null` = insufficient history) to distinguish missing data from a stable 0.0 change.
*   **Trend**: The string label (Rising/Falling/Stable) is derived **exclusively** from the **3-hour delta**, following the WMO standard for "Barometric Tendency". A noise threshold of 0.5 mb filters out insignificant fluctuations.

## 5. Monitored Locations

| Location ID | Latitude | Longitude |
|------------|----------|-----------|
| `house-nick` | 30.2605 | -97.6677 |
| `house-nita` | 30.2942 | -97.6959 |
| `distribution-hall` | 30.2619 | -97.7282 |
