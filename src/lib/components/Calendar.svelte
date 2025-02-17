<script lang="ts">
  import { page } from '$app/stores';
  import SectionHeader from './SectionHeader.svelte';

  $: ({ calendarUrl } = $page.data);
</script>

<div class="calendar-container">
  <SectionHeader title="Calendar" fullscreenPath="/fullscreen/calendar" />

  <iframe title="Family Calendar" src={calendarUrl} class="calendar-frame" scrolling="no" />
</div>

<style>
  /* Large (default) styles */
  .calendar-container {
    padding: 1.5rem;
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .calendar-frame {
    width: 100%;
    flex: 1;
    border: none;
    min-height: 0;
  }

  :global(.fullscreen) .calendar-container {
    height: calc(100vh - 48px - 88px - 3rem); /* Header (48px) + Nav (88px) + Padding (3rem) */
  }

  /* Medium (1360x768) styles */
  @media (max-width: 1360px) and (max-height: 768px) {
    .calendar-container {
      padding: 0.5rem;
    }
  }

  /* Small (mobile) styles */
  @media (max-width: 768px) {
    :global(.fullscreen) .calendar-container {
      height: calc(100vh - 48px - 64px - 2rem); /* Header (48px) + Nav (64px) + Padding (2rem) */
      padding: 1rem;
    }
  }
</style>
