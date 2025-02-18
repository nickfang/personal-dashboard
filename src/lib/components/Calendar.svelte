<script lang="ts">
  import { page } from '$app/stores';
  import SectionHeader from './SectionHeader.svelte';

  $: ({ calendarUrl } = $page.data);
</script>

<div class="calendar-container">
  <SectionHeader title="Calendar" fullscreenPath="/fullscreen/calendar" />

  <iframe
    src={calendarUrl + "&showDate=false&showPrint=false&showNav=false&showTabs=true&showCalendars=false"}
    style="border-width:0"
    width="100%"
    height="100%"
    frameborder="0"
    scrolling="no"
  />
</div>

<style>
  /* Large (default) styles */
  .calendar-container {
    padding: 1.5rem;
    height: 100%;
    display: flex;
    flex-direction: column;
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
