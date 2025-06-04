<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import Calendar from '$lib/components/Calendar.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
</script>

<div class="dashboard-grid">
  <div class="nav">
    Nav
  </div>
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
  /* Large (default) styles */
  .dashboard-grid {
    display: grid;
    gap: 2rem;
    padding: 1rem;
    grid-template-columns: 1fr 2fr;
    height: 100vh;
    box-sizing: border-box;
    grid-template-rows: 72px 1fr 1fr;
    width: 100%;
    position: absolute;
    left: 0;
    right: 0;
    overflow: hidden;
  }

  .weather-section,
  .word-section,
  .calendar-section {
    overflow: hidden;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .nav {
    grid-column: 1;
    grid-row: 1;
  }

  .weather-section {
    grid-column: 1;
    grid-row: 2;
  }

  .calendar-section {
    grid-column: 2;
    grid-row: 2 / span 2;
  }

  .word-section {
    grid-column: 1;
    grid-row: 3;
  }

  .weather-section :global(.weather-container),
  .word-section :global(.word-container) {
    transform: scale(var(--scale, 1));
    transform-origin: top center;
  }

  :global(.fullscreen) {
    overflow: hidden;
  }

  :global(.fullscreen) .dashboard-grid {
    height: 100vh;
    padding: 0.5rem;
    gap: 0.5rem;
    width: 100%;
  }

  /* Medium (1360x768) styles */
  @media (max-width: 1360px) and (max-height: 768px) {
    .dashboard-grid {
      gap: 1rem;
      padding: 0.5rem;
      width: 100%;
      grid-template-rows: auto auto;
      position: relative;
      height: auto;
      min-height: 100vh;
      overflow: visible;
    }
  }

  /* Small (mobile) styles */
  @media (max-width: 768px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      height: auto;
      min-height: 100vh;
      padding: 1rem;
      gap: 1rem;
      grid-template-rows: auto;
      width: 100%;
      position: relative;
      overflow: visible;
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
