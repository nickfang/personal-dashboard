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
    min-height: calc(100vh - 80px);
    max-width: 2000px;
    margin: 0 auto;
    box-sizing: border-box;
    grid-template-rows: 1fr 1fr;
  }

  .weather-section, .word-section, .calendar-section {
    overflow: auto;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    display: flex;
    flex-direction: column;
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

  @media (max-width: 768px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      min-height: calc(100vh - 80px);
      padding: 1rem;
      gap: 1rem;
      overflow: visible;
      grid-template-rows: auto;
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
