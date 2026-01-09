<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { onMount } from 'svelte';
  import PressureGraph from './PressureGraph.svelte';
  import SectionHeader from './SectionHeader.svelte';

  let weather: any = null;
  let forecast: any = null;
  let loading = true;
  let error: string | null = null;
  let refreshInterval: NodeJS.Timeout;
  let lastUpdated: Date | null = null;

  const dispatch = createEventDispatcher();

  async function fetchWeather() {
    try {
      const response = await fetch('/api/weather');
      const data = await response.json();
      weather = data.current;
      forecast = data.forecast;
      loading = false;
      lastUpdated = new Date();
      dispatch('weatherData', { forecast: data.forecast });
    } catch (e) {
      error = 'Failed to load weather data';
      loading = false;
    }
  }

  $: forecastDays = forecast?.forecastday || [];
  $: lastUpdatedTime = lastUpdated
    ? lastUpdated.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true,
      })
    : '';

  onMount(fetchWeather);
</script>

<div class="weather-container">
  <SectionHeader title="Weather" fullscreenPath="/fullscreen/weather" />

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if error}
    <div class="error">{error}</div>
  {:else}
    <div class="weather-grid">
      <div class="top-row">
        <div class="current-weather">
          <img src={weather.condition.icon} alt={weather.condition.text} width="64" height="64" />
          <div class="current-main">
            <div class="location">Austin, TX</div>
            {#if lastUpdatedTime}
              <div class="last-updated">Updated {lastUpdatedTime}</div>
            {/if}
            <div class="temp-condition">
              <div class="temperature">{weather.temp_f}°F</div>
              <div class="condition">{weather.condition.text}</div>
            </div>
          </div>
          <div class="weather-details">
            <span>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M12 3v18M7 6l10 0M7 12l10 0M7 18l10 0" />
              </svg>
              {weather.humidity}%
            </span>
            <span>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" />
                <path d="M12 12v9" />
                <path d="m8 17 4 4 4-4" />
              </svg>
              {weather.precip_in}" rain
            </span>
          </div>
        </div>

        <div class="forecast-grid">
          {#each forecastDays as day}
            <div class="forecast-card">
              <div class="day-row">
                <div class="forecast-day">
                  {new Date(day.date + 'T00:00:00').toLocaleDateString('en-US', {
                    weekday: 'short',
                  })}
                </div>
                <img
                  src={day.day.condition.icon}
                  alt={day.day.condition.text}
                  width="40"
                  height="40"
                />
              </div>
              <div class="temp-range">
                <span class="max-temp">{day.day.maxtemp_f}°</span>
                <span class="min-temp">{day.day.mintemp_f}°</span>
              </div>
              <div class="forecast-details">
                <span>
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" />
                    <path d="M12 12v9" />
                    <path d="m8 17 4 4 4-4" />
                  </svg>
                  {day.day.totalprecip_in}" rain
                </span>
              </div>
            </div>
          {/each}
        </div>
      </div>

      {#if forecast}
        <div class="graph-container">
          <PressureGraph {forecast} />
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .weather-container {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-sizing: border-box;
  }

  .loading {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 12rem;
  }

  .spinner {
    width: 2rem;
    height: 2rem;
    border: 2px solid var(--teal-600);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  .error {
    color: red;
    text-align: center;
  }

  .weather-grid {
    flex: 1;
    display: grid;
    grid-template-rows: 1fr 200px; /* Fixed heights: flexible top row, 200px chart */
    gap: 1rem;
    overflow: hidden;
  }

  .top-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    min-height: 0;
    overflow: hidden;
  }

  .current-weather {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.75rem;
    align-items: center;
    padding: 1rem;
    background: var(--teal-50);
    border-radius: 0.75rem;
    min-height: 0;
  }

  .current-main {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
  }

  .temp-condition {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .temperature {
    font-size: 2.5rem;
    font-weight: 300;
    color: var(--teal-800);
  }

  .condition {
    color: var(--teal-600);
    font-size: 1rem;
    margin-top: 0.25rem;
  }

  .location {
    color: var(--teal-600);
    font-size: 1.1rem;
    margin-bottom: 0.25rem;
    font-weight: 500;
  }

  .last-updated {
    color: var(--teal-600);
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
    opacity: 0.8;
  }

  .weather-details {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.75rem;
    color: var(--teal-600);
    font-size: 1rem;
  }

  .forecast-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
    min-height: 0;
  }

  .forecast-card {
    border: 1px solid var(--teal-100);
    border-radius: 0.75rem;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.5);
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    min-height: 120px;
  }

  .day-row {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1rem;
    gap: 0.5rem;
  }

  .forecast-day {
    color: var(--teal-600);
    font-size: 1rem;
    font-weight: 500;
  }

  .temp-range {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
    margin: 0.5rem 0;
  }

  .max-temp {
    color: var(--teal-800);
    font-weight: 600;
    font-size: 1.1rem;
  }

  .min-temp {
    color: var(--teal-600);
    font-weight: 400;
    font-size: 1.1rem;
  }

  .forecast-details {
    font-size: 0.85rem;
    color: var(--teal-600);
    margin-top: auto;
  }

  .forecast-details span {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .forecast-details svg {
    width: 14px;
    height: 14px;
  }

  .graph-container {
    background: white;
    padding: 1rem;
    border-radius: 0.75rem;
    border: 1px solid var(--teal-100);
    height: 100%; /* Fill the allocated grid space */
    max-height: 200px; /* Consistent maximum height */
    grid-column: 1 / -1; /* Full width across both columns */
    overflow: hidden; /* Prevent content overflow */
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  /*
   * CONTAINER QUERY BREAKPOINTS:
   * - 400px height: Reduce chart height, compact spacing
   * - 300px height: Hide chart entirely
   * - 250px height: Ultra-compact forecast layout
   * - 400px width: Stack to single column
   */

  /* Medium container height - reduce chart, tighten spacing */
  @container (max-height: 400px) {
    .weather-container {
      padding: var(--space-sm);
    }

    .weather-grid {
      gap: var(--space-sm);
      grid-template-rows: 1fr 120px;
    }

    .top-row {
      gap: var(--space-sm);
    }

    .current-weather {
      padding: var(--space-sm);
      gap: var(--space-sm);
    }

    .temperature {
      font-size: var(--font-2xl);
    }

    .location {
      font-size: var(--font-base);
      margin-bottom: var(--space-xs);
    }

    .last-updated {
      font-size: var(--font-xs);
      margin-bottom: var(--space-xs);
    }

    .condition {
      font-size: var(--font-sm);
    }

    .weather-details {
      gap: var(--space-sm);
      font-size: var(--font-sm);
    }

    .forecast-grid {
      gap: var(--space-sm);
    }

    .forecast-card {
      padding: var(--space-sm);
      min-height: 80px;
      gap: var(--space-xs);
    }

    .day-row {
      margin-bottom: var(--space-sm);
      gap: var(--space-xs);
    }

    .day-row img {
      width: 32px;
      height: 32px;
    }

    .forecast-day {
      font-size: var(--font-sm);
    }

    .temp-range {
      margin: var(--space-xs) 0;
      gap: 0.125rem;
    }

    .max-temp,
    .min-temp {
      font-size: var(--font-base);
    }

    .forecast-details {
      font-size: var(--font-xs);
    }

    .forecast-details svg {
      width: 12px;
      height: 12px;
    }

    .graph-container {
      padding: var(--space-sm);
      max-height: 120px;
    }
  }

  /* Small container height - hide chart */
  @container (max-height: 300px) {
    .weather-grid {
      grid-template-rows: 1fr;
    }

    .graph-container {
      display: none;
    }

    .forecast-card {
      min-height: 70px;
    }
  }

  /* Very small container height - ultra-compact */
  @container (max-height: 250px) {
    .weather-container {
      padding: var(--space-xs);
    }

    .weather-grid {
      gap: var(--space-xs);
    }

    .top-row {
      gap: var(--space-xs);
    }

    .current-weather {
      padding: var(--space-xs);
    }

    .temperature {
      font-size: var(--font-xl);
    }

    .forecast-card {
      padding: var(--space-xs);
      min-height: 60px;
    }

    .day-row img {
      width: 24px;
      height: 24px;
    }
  }

  /* Narrow container width - stack layout */
  @container (max-width: 500px) {
    .top-row {
      grid-template-columns: 1fr;
      gap: var(--space-sm);
    }

    .current-weather {
      display: flex;
      flex-direction: row;
      align-items: center;
      gap: var(--space-md);
      padding: var(--space-md);
      text-align: left;
    }

    .current-main {
      flex: 1;
      order: 2;
    }

    .current-weather img {
      order: 1;
      width: 56px;
      height: 56px;
      flex-shrink: 0;
    }

    .weather-details {
      order: 3;
      flex-direction: column;
      align-items: flex-end;
      gap: var(--space-sm);
      flex-shrink: 0;
    }

    .temp-condition {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--space-xs);
    }

    .temperature {
      font-size: 1.75rem;
    }

    .condition {
      font-size: var(--font-sm);
    }

    .location {
      font-size: var(--font-base);
    }

    .forecast-grid {
      grid-template-columns: 1fr;
      gap: var(--space-sm);
    }

    .forecast-card {
      display: flex;
      flex-direction: row;
      align-items: center;
      gap: var(--space-sm);
      padding: var(--space-sm);
      min-height: 60px;
    }

    .day-row {
      flex-direction: row;
      align-items: center;
      gap: var(--space-sm);
      margin-bottom: 0;
      flex-shrink: 0;
    }

    .day-row img {
      width: 32px;
      height: 32px;
    }

    .forecast-day {
      font-size: var(--font-xs);
      font-weight: 500;
      min-width: 45px;
    }

    .temp-range {
      flex-direction: row;
      gap: var(--space-xs);
      margin: 0;
      flex-shrink: 0;
    }

    .max-temp,
    .min-temp {
      font-size: var(--font-sm);
    }

    .max-temp::after {
      content: '/';
      color: var(--teal-400);
      margin-left: 2px;
    }

    .forecast-details {
      font-size: 0.7rem;
      margin-top: 0;
      margin-left: auto;
    }

    .graph-container {
      display: block; /* Override height-based hiding in scrollable mode */
      padding: var(--space-sm);
      max-height: 160px; /* Reasonable size for narrow scrollable layout */
    }

    /*
     * In narrow scrollable mode, remove height constraints.
     * The page can scroll, so show all content naturally.
     */
    .weather-grid {
      grid-template-rows: auto auto; /* Let rows size naturally */
      overflow: visible;
    }
  }
</style>
