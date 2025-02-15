<script lang="ts">
  import { page } from '$app/stores';
  import SectionHeader from './SectionHeader.svelte';

  $: ({ calendarUrl } = $page.data);
</script>

<style>
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
</style>

<div class="calendar-container">
  <SectionHeader 
    title="Calendar" 
    fullscreenPath="/fullscreen/calendar" 
  />

  <iframe
    title="Family Calendar"
    src={calendarUrl}
    class="calendar-frame"
    scrolling="no"
  />
</div>  
