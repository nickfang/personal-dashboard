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

<style>
  .weather-container {
    padding: 1.5rem;
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
    display: grid;
    gap: 1.5rem;
    grid-template-columns: 1fr;
  }

  .current-weather {
    text-align: center;
  }

  .temperature {
    font-size: 3.75rem;
    font-weight: 300;
    color: var(--teal-800);
    margin-bottom: 0.5rem;
  }

  .condition {
    color: var(--teal-600);
    margin-bottom: 1rem;
  }

  .forecast-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
    text-align: center;
  }

  .forecast-card {
    border: 1px solid var(--teal-100);
    border-radius: 0.5rem;
    padding: 1rem;
    background: rgba(255, 255, 255, 0.5);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .forecast-day {
    color: var(--teal-600);
    font-size: 0.875rem;
    margin-bottom: 0.5rem;
  }

  .weather-details {
    display: flex;
    justify-content: center;
    gap: 2rem;
    margin: 1rem 0;
    color: var(--teal-600);
    font-size: 0.875rem;
  }

  .weather-details span {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .location {
    color: var(--teal-600);
    font-size: 1.125rem;
    margin-bottom: 1rem;
  }

  .forecast-details {
    font-size: 0.75rem;
    color: var(--teal-600);
    margin-top: 0.5rem;
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
    margin-top: 0.5rem;
  }

  .min-temp {
    color: var(--teal-600);
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>

<div class="weather-container">
  <SectionHeader 
    title="Weather" 
    fullscreenPath="/fullscreen/weather" 
  />

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if error}
    <div class="error">{error}</div>
  {:else}
    <div class="weather-grid">
      <div class="current-weather">
        <div class="location">Austin, TX</div>
        <div class="temperature">{weather.temp_f}°F</div>
        <div class="condition">{weather.condition.text}</div>
        <div class="weather-details">
          <span>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 3v18M7 6l10 0M7 12l10 0M7 18l10 0"/>
            </svg>
            {weather.humidity}%
          </span>
          <span>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/>
              <path d="M12 12v9"/>
              <path d="m8 17 4 4 4-4"/>
            </svg>
            {weather.precip_in}" rain
          </span>
        </div>
        <img 
          src={weather.condition.icon} 
          alt={weather.condition.text}
          width="64"
          height="64"
        />
      </div>

      <div class="forecast-grid">
        {#each forecastDays as day}
          <div class="forecast-card">
            <div class="forecast-day">
              {new Date(day.date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' })}
            </div>
            <img 
              src={day.day.condition.icon} 
              alt={day.day.condition.text}
              width="40"
              height="40"
            />
            <div class="max-temp">{day.day.maxtemp_f}°</div>
            <div class="min-temp">{day.day.mintemp_f}°</div>
            <div class="forecast-details">
              <span>
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/>
                  <path d="M12 12v9"/>
                  <path d="m8 17 4 4 4-4"/>
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