<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { onMount } from 'svelte';
  import PressureGraph from './PressureGraph.svelte';
  import SectionHeader from './SectionHeader.svelte';

  let weather: any = null;
  let forecast: any = null;
  let loading = true;
  let error: string | null = null;

  const dispatch = createEventDispatcher();

  async function fetchWeather() {
    try {
      const response = await fetch('/api/weather');
      const data = await response.json();
      weather = data.current;
      forecast = data.forecast;
      loading = false;
      dispatch('weatherData', { forecast: data.forecast });
    } catch (e) {
      error = 'Failed to load weather data';
      loading = false;
    }
  }

  $: forecastDays = forecast?.forecastday || [];

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
      <div class="current-weather">
        <img src={weather.condition.icon} alt={weather.condition.text} width="48" height="48" />
        <div class="current-main">
          <div class="location">Austin, TX</div>
          <div class="temp-condition">
            <div class="temperature">{weather.temp_f}°F</div>
            <div class="condition">{weather.condition.text}</div>
          </div>
        </div>
        <div class="weather-details">
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
              <path d="M12 3v18M7 6l10 0M7 12l10 0M7 18l10 0" />
            </svg>
            {weather.humidity}%
          </span>
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
            {weather.precip_in}" rain
          </span>
        </div>
      </div>

      <div class="forecast-grid">
        {#each forecastDays as day}
          <div class="forecast-card">
            <div class="day-row">
              <div class="forecast-day">
                {new Date(day.date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' })}
              </div>
              <img
                src={day.day.condition.icon}
                alt={day.day.condition.text}
                width="32"
                height="32"
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

      {#if forecast}
        <PressureGraph {forecast} />
      {/if}
    </div>
  {/if}
</div>

<style>
  .weather-container {
    padding: 1.5rem;
    height: 100%;
    max-width: 800px;
    margin-left: auto;
    margin-right: auto;
    display: flex;
    flex-direction: column;
    overflow: auto;
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
    gap: 0.75rem;
    min-height: 0;
    overflow: auto;
  }

  .current-weather {
    text-align: center;
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 2rem;
    align-items: center;
    padding: 1rem 3rem;
    background: var(--teal-50);
    border-radius: 0.75rem;
    margin-bottom: 0.75rem;
  }

  .current-main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    text-align: left;
  }

  .temp-condition {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
  }

  .temperature {
    font-size: 2.5rem;
    font-weight: 300;
    color: var(--teal-800);
  }

  .condition {
    color: var(--teal-600);
    font-size: 0.875rem;
    margin-top: 0.25rem;
  }

  .forecast-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
    text-align: center;
  }

  .forecast-card {
    border: 1px solid var(--teal-100);
    border-radius: 0.5rem;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.5);
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .day-row {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 0.5rem;
  }

  .forecast-day {
    color: var(--teal-600);
    font-size: 0.875rem;
    margin-right: 0.5rem;
  }

  .temp-range {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .forecast-details {
    font-size: 0.75rem;
    color: var(--teal-600);
    display: grid;
    gap: 0.25rem;
  }

  .forecast-details span {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .forecast-details svg {
    width: 12px;
    height: 12px;
  }

  .max-temp {
    color: var(--teal-800);
    font-weight: 500;
  }

  .min-temp {
    color: var(--teal-600);
    font-weight: 400;
  }

  .weather-details {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.5rem;
    color: var(--teal-600);
    font-size: 0.875rem;
  }

  .location {
    color: var(--teal-600);
    font-size: 1rem;
    margin-bottom: 0.25rem;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 768px) {
    .weather-container {
      padding: 1rem;
    }

    .current-weather {
      grid-template-columns: 1fr;
      gap: 1rem;
      padding: 1rem;
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .current-main {
      align-items: center;
      text-align: center;
      order: 2;
    }

    .current-weather img {
      width: 64px;
      height: 64px;
      order: 1;
      margin-bottom: 0.5rem;
    }

    .weather-details {
      align-items: center;
      flex-direction: row;
      justify-content: center;
      gap: 2rem;
      order: 3;
      margin: 0.5rem 0;
    }

    .forecast-grid {
      grid-template-columns: 1fr;
    }

    .graph-container {
      margin: 0 0.5rem;
    }
  }
</style>
