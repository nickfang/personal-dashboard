<script lang="ts">
  import { page } from '$app/stores';
  import SectionHeader from './SectionHeader.svelte';

  $: ({ calendarUrl } = $page.data);
</script>

<div class="calendar-container">
  <SectionHeader title="Calendar" fullscreenPath="/fullscreen/calendar" />

  <iframe
    src={calendarUrl +
      '&showDate=false&showPrint=false&showNav=false&showTabs=true&showCalendars=false'}
    style="border-width:0"
    width="100%"
    height="100%"
    frameborder="0"
    scrolling="no"
    title="Google Calendar"
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

  .calendar-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 300px;
    color: var(--teal-600, #006666);
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--teal-100, #a3dfdf);
    border-top: 3px solid var(--teal-600, #006666);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }

  @keyframes spin {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
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
