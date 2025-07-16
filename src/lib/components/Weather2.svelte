<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import PressureGraph from './PressureGraph.svelte';
  import SectionHeader from './SectionHeader.svelte';

  let location = 'Austin, TX';
  let weather: any = null;
  let forecast: any = null;
  let loading: boolean = true;
  let error: string | null = null;
  let lastUpdated: Date | null = null;

  const dispatch = createEventDispatcher();

  async function fetchWeather() {
    try {
      const response = await fetch('/api/weather');
      const data = await response.json();
      weather = data.current;
      forecast = data.forecast;
      location = data.location?.name
        ? `${data.location.name}, ${data.location.region || data.location.country || ''}`
        : location;
      loading = false;
      lastUpdated = new Date();
      dispatch('weatherData', { forecast: data.forecast });
    } catch (e) {
      error = 'Failed to load weather data';
      loading = false;
    }
  }

  $: forecastDays = forecast?.forecastday?.slice(0, 3) || [];
  $: lastUpdatedTime = lastUpdated
    ? lastUpdated.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true,
      })
    : '';

  onMount(fetchWeather);
</script>

<div class="weather2-container">
  <SectionHeader title="Weather" fullscreenPath="/fullscreen/weather" />
  {#if loading}
    <div class="loading"><div class="spinner"></div></div>
  {:else if error}
    <div class="error">{error}</div>
  {:else}
    <div class="weather2-grid">
      <div class="current">
        <img class="icon" src={weather.condition.icon} alt={weather.condition.text} />
        <div class="main">
          <div class="location">{location}</div>
          {#if lastUpdatedTime}
            <div class="last-updated">Updated {lastUpdatedTime}</div>
          {/if}
          <div class="temp">{weather.temp_f}°F</div>
          <div class="condition">{weather.condition.text}</div>
        </div>
        <div class="details">
          <span title="Humidity">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"><path d="M12 3v18M7 6l10 0M7 12l10 0M7 18l10 0" /></svg
            >
            {weather.humidity}%
          </span>
          <span title="Precipitation">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" /><path
                d="M12 12v9"
              /><path d="m8 17 4 4 4-4" /></svg
            >
            {weather.precip_in}" rain
          </span>
        </div>
      </div>
      <div class="forecast">
        {#each forecastDays as day}
          <div class="forecast-card">
            <div class="date">
              {new Date(day.date + 'T00:00:00').toLocaleDateString('en-US', { weekday: 'short' })}
            </div>
            <img class="icon" src={day.day.condition.icon} alt={day.day.condition.text} />
            <div class="temps">
              <span class="max">{day.day.maxtemp_f}°</span>
              <span class="min">{day.day.mintemp_f}°</span>
            </div>
            <div class="precip">
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
                ><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" /><path
                  d="M12 12v9"
                /><path d="m8 17 4 4 4-4" /></svg
              >
              {day.day.totalprecip_in}" rain
            </div>
          </div>
        {/each}
      </div>
      <div class="graph">
        {#if forecast}
          <PressureGraph {forecast} />
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .weather2-container {
    box-sizing: border-box;
    width: 100%;
    height: 100%;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    font-size: clamp(0.65rem, 1vw, 1.05rem);
    padding: clamp(0.05rem, 0.5vw, 0.4rem);
  }
  .weather2-grid {
    flex: 1 1 auto;
    display: grid;
    grid-template-areas:
      'current forecast'
      'graph graph';
    grid-template-columns: 1fr 1fr;
    grid-template-rows: minmax(0, 0.48fr) minmax(0, 0.52fr);
    gap: clamp(0.15rem, 0.5vw, 0.5rem);
    min-width: 0;
    min-height: 0;
    height: 100%;
    width: 100%;
  }
  .current {
    grid-area: current;
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    background: var(--teal-50, #f0fdfa);
    border-radius: 0.55rem;
    padding: clamp(0.15rem, 0.5vw, 0.5rem);
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
    height: 100%;
  }
  .current .icon {
    width: clamp(2rem, 4vw, 3rem);
    height: clamp(2rem, 4vw, 3rem);
    margin-right: clamp(0.2rem, 1vw, 0.5rem);
    flex-shrink: 0;
  }
  .main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    min-width: 0;
    min-height: 0;
    flex: 1 1 0;
    gap: 0.1em;
    width: 100%;
    overflow: hidden;
    word-break: break-word;
  }
  .main > * {
    min-width: 0;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: normal;
    word-break: break-word;
  }
  @media (max-width: 900px) {
    .current {
      flex-direction: column;
      align-items: center;
      gap: clamp(0.2rem, 1vw, 0.5rem);
    }
    .main {
      align-items: center;
      text-align: center;
    }
  }
  .details {
    display: flex;
    flex-direction: column;
    gap: clamp(0.1rem, 0.5vw, 0.3rem);
    color: var(--teal-700, #0f766e);
    font-size: clamp(0.7rem, 0.9vw, 1rem);
    margin-left: clamp(0.2rem, 1vw, 0.4rem);
    flex-shrink: 0;
  }
  .forecast {
    grid-area: forecast;
    display: flex;
    flex-direction: column;
    gap: clamp(0.1rem, 0.5vw, 0.3rem);
    min-width: 0;
    min-height: 0;
    align-items: stretch;
    justify-content: flex-start;
    height: 100%;
  }
  .forecast-card {
    background: rgba(255, 255, 255, 0.7);
    border-radius: 0.45rem;
    border: 1px solid var(--teal-100, #ccfbf1);
    padding: clamp(0.1rem, 0.5vw, 0.3rem);
    display: flex;
    flex-direction: row;
    align-items: center;
    min-width: 0;
    min-height: 0;
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
    flex: 1 1 0;
    height: auto;
    justify-content: flex-start;
    gap: clamp(0.1rem, 0.5vw, 0.3rem);
  }
  .forecast-card .date {
    color: var(--teal-600, #0891b2);
    font-size: clamp(0.7rem, 0.9vw, 1rem);
    margin-right: clamp(0.1rem, 0.5vw, 0.2rem);
    min-width: 2.5em;
  }
  .forecast-card .icon {
    width: clamp(1.1rem, 2vw, 1.7rem);
    height: clamp(1.1rem, 2vw, 1.7rem);
    margin-right: clamp(0.1rem, 0.5vw, 0.2rem);
  }
  .temps {
    display: flex;
    gap: 0.2em;
    font-size: clamp(0.8rem, 0.9vw, 1rem);
    margin-right: clamp(0.1rem, 0.5vw, 0.2rem);
  }
  .max {
    color: var(--teal-900, #134e4a);
    font-weight: 500;
  }
  .min {
    color: var(--teal-600, #0891b2);
    font-weight: 400;
  }
  .precip {
    display: flex;
    align-items: center;
    gap: 0.2em;
    color: var(--teal-700, #0f766e);
    font-size: clamp(0.7rem, 0.9vw, 1rem);
  }
  .graph {
    grid-area: graph;
    background: #fff;
    border-radius: 0.55rem;
    border: 1px solid var(--teal-100, #ccfbf1);
    min-width: 0;
    min-height: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    align-items: stretch;
    justify-content: stretch;
    padding: clamp(0.08rem, 0.5vw, 0.3rem);
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
  }
  .graph > * {
    width: 100% !important;
    height: 100% !important;
    min-width: 0 !important;
    min-height: 0 !important;
    max-width: 100% !important;
    max-height: 100% !important;
    display: block;
  }
  .current {
    grid-area: current;
    display: flex;
    flex-direction: column;
    align-items: center;
    background: var(--teal-50, #f0fdfa);
    border-radius: 0.75rem;
    padding: clamp(0.3rem, 1vw, 1.2rem);
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
  }
  .current .icon {
    width: clamp(2.5rem, 7vw, 4.5rem);
    height: clamp(2.5rem, 7vw, 4.5rem);
    margin-bottom: clamp(0.2rem, 1vw, 0.5rem);
  }
  .main {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 0;
    min-height: 0;
  }
  .location {
    color: var(--teal-700, #0f766e);
    font-size: clamp(0.8rem, 1vw, 1.1rem);
    font-weight: 500;
    margin-bottom: clamp(0.1rem, 0.5vw, 0.3rem);
  }
  .last-updated {
    color: var(--teal-600, #0891b2);
    font-size: clamp(0.7rem, 0.8vw, 1rem);
    opacity: 0.7;
    margin-bottom: clamp(0.1rem, 0.5vw, 0.3rem);
  }
  .temp {
    font-size: clamp(1.5rem, 4vw, 2.5rem);
    font-weight: 300;
    color: var(--teal-900, #134e4a);
    margin-bottom: clamp(0.1rem, 0.5vw, 0.3rem);
  }
  .condition {
    color: var(--teal-600, #0891b2);
    font-size: clamp(0.8rem, 1vw, 1.1rem);
    margin-bottom: clamp(0.1rem, 0.5vw, 0.3rem);
  }
  .details {
    display: flex;
    gap: clamp(0.5rem, 2vw, 1.5rem);
    color: var(--teal-700, #0f766e);
    font-size: clamp(0.8rem, 1vw, 1.1rem);
    margin-top: clamp(0.2rem, 1vw, 0.5rem);
  }
  .details span {
    display: flex;
    align-items: center;
    gap: 0.3em;
  }
  .forecast {
    grid-area: forecast;
    display: flex;
    flex-direction: column;
    gap: clamp(0.2rem, 1vw, 0.7rem);
    min-width: 0;
    min-height: 0;
    align-items: stretch;
    justify-content: space-between;
  }
  .forecast-card {
    background: rgba(255, 255, 255, 0.7);
    border-radius: 0.75rem;
    border: 1px solid var(--teal-100, #ccfbf1);
    padding: clamp(0.2rem, 1vw, 0.7rem);
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 0;
    min-height: 0;
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
  }
  .forecast-card .date {
    color: var(--teal-600, #0891b2);
    font-size: clamp(0.8rem, 1vw, 1.1rem);
    margin-bottom: clamp(0.1rem, 0.5vw, 0.2rem);
  }
  .forecast-card .icon {
    width: clamp(1.5rem, 5vw, 2.5rem);
    height: clamp(1.5rem, 5vw, 2.5rem);
    margin-bottom: clamp(0.1rem, 0.5vw, 0.2rem);
  }
  .temps {
    display: flex;
    gap: 0.5em;
    font-size: clamp(0.9rem, 1vw, 1.2rem);
    margin-bottom: clamp(0.1rem, 0.5vw, 0.2rem);
  }
  .max {
    color: var(--teal-900, #134e4a);
    font-weight: 500;
  }
  .min {
    color: var(--teal-600, #0891b2);
    font-weight: 400;
  }
  .precip {
    display: flex;
    align-items: center;
    gap: 0.3em;
    color: var(--teal-700, #0f766e);
    font-size: clamp(0.8rem, 1vw, 1.1rem);
  }
  .graph {
    grid-area: graph;
    background: #fff;
    border-radius: 0.75rem;
    border: 1px solid var(--teal-100, #ccfbf1);
    min-width: 0;
    min-height: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    display: flex;
    align-items: stretch;
    justify-content: stretch;
    padding: clamp(0.2rem, 1vw, 1rem);
    box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
  }
  .graph > * {
    width: 100% !important;
    height: 100% !important;
    min-width: 0 !important;
    min-height: 0 !important;
    max-width: 100% !important;
    max-height: 100% !important;
    display: block;
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
    border: 2px solid var(--teal-600, #0891b2);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  .error {
    color: red;
    text-align: center;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  @media (max-width: 900px) {
    .weather2-grid {
      grid-template-areas:
        'current'
        'forecast'
        'graph';
      grid-template-columns: 1fr;
      grid-template-rows: minmax(0, 0.28fr) minmax(0, 0.22fr) minmax(0, 0.5fr);
      gap: clamp(0.1rem, 0.5vw, 0.3rem);
    }
    .current {
      flex-direction: column;
      align-items: center;
      justify-content: flex-start;
      padding: clamp(0.1rem, 1vw, 0.3rem);
    }
    .current .icon {
      margin-right: 0;
      margin-bottom: clamp(0.1rem, 1vw, 0.3rem);
    }
    .main {
      align-items: center;
      text-align: center;
    }
    .details {
      flex-direction: row;
      margin-left: 0;
      margin-top: clamp(0.1rem, 1vw, 0.3rem);
      justify-content: center;
      width: 100%;
    }
    .forecast {
      flex-direction: column;
      gap: clamp(0.1rem, 0.5vw, 0.3rem);
      justify-content: flex-start;
      align-items: stretch;
      height: 100%;
    }
    .forecast-card {
      flex: 1 1 0;
      min-width: 0;
      height: auto;
      padding: clamp(0.08rem, 1vw, 0.2rem);
    }
  }
  @media (max-width: 600px) {
    .weather2-container {
      padding: clamp(0.03rem, 1vw, 0.2rem);
      font-size: clamp(0.55rem, 1.2vw, 0.9rem);
    }
    .weather2-grid {
      gap: clamp(0.05rem, 0.5vw, 0.2rem);
      grid-template-rows: minmax(0, 0.22fr) minmax(0, 0.18fr) minmax(0, 0.6fr);
    }
    .current,
    .forecast-card,
    .graph {
      padding: clamp(0.05rem, 1vw, 0.2rem);
    }
    .main,
    .details {
      font-size: clamp(0.6rem, 1.2vw, 0.8rem);
    }
    .forecast {
      flex-direction: column;
      gap: clamp(0.08rem, 0.5vw, 0.15rem);
      height: 100%;
    }
    .forecast-card {
      height: auto;
      min-height: 0;
    }
  }
</style>
