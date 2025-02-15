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
    height: calc(100vh - 80px); /* Account for header and padding */
    max-width: 2000px;
    margin: 0 auto;
    box-sizing: border-box;
    grid-template-rows: 1fr 1fr; /* Two equal rows */
  }

  .weather-section, .word-section, .calendar-section {
    overflow: hidden;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    display: flex;
    flex-direction: column;
    min-height: 0; /* Important for Firefox */
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

  @media (max-width: 1024px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      height: auto;
      min-height: calc(100vh - 80px);
      padding: 1rem;
    }

    .calendar-section {
      grid-column: 1;
      grid-row: auto;
      min-height: 600px;
    }
  }
</style>
