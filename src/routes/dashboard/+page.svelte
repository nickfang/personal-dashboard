<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import Calendar from '$lib/components/Calendar.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
</script>

<div class="dashboard-grid">
  <div class="weather-section">
    <Weather />
  </div>
  <div class="calendar-section">
    <Calendar />
  </div>
  <div class="word-section">
    <SatWord />
  </div>
</div>

<style>
  .dashboard-grid {
    display: grid;
    gap: 2rem;
    padding: 1rem;
    grid-template-columns: 1fr 2fr;
    height: calc(100vh - 80px);
    margin: 0 auto;
    box-sizing: border-box;
    grid-template-rows: 1fr 1fr;
    width: min(calc(100vh - 80px) * 16/9, 100vw - 2rem);
    height: min(calc((100vw - 2rem) * 9/16), calc(100vh - 80px));
  }

  .weather-section,
  .word-section,
  .calendar-section {
    overflow: hidden;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .weather-section {
    grid-column: 1;
    grid-row: 1;
  }

  .calendar-section {
    grid-column: 2;
    grid-row: 1 / span 2;
  }

  .word-section {
    grid-column: 1;
    grid-row: 2;
  }

  .weather-section :global(.weather-container),
  .word-section :global(.word-container) {
    transform: scale(var(--scale, 1));
    transform-origin: top center;
  }

  @media (max-width: 768px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      height: auto;
      padding: 1rem;
      gap: 1rem;
      grid-template-rows: auto;
      width: 100%;
    }

    .weather-section {
      grid-row: auto;
    }

    .calendar-section {
      grid-column: 1;
      grid-row: auto;
      min-height: 600px;
    }

    .word-section {
      grid-column: 1;
      grid-row: auto;
    }
  }
</style>
