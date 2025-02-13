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
    max-width: 2000px;
    margin: 0 auto;
    box-sizing: border-box;
    overflow: hidden;
  }

  .weather-section, .word-section, .calendar-section {
    overflow: hidden;
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
      padding: 1rem;
      height: auto;
    }

    .calendar-section {
      grid-column: 1;
      grid-row: auto;
      height: 600px;
    }
  }
</style>
