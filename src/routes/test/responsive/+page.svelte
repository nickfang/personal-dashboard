<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
  import Calendar2 from '$lib/components/Calendar2.svelte';

  // URL params control which component and size to test
  import { page } from '$app/stores';

  $: component = $page.url.searchParams.get('component') || 'weather';
  $: height = parseInt($page.url.searchParams.get('height') || '500');
  $: width = parseInt($page.url.searchParams.get('width') || '800');
</script>

<div class="test-wrapper">
  <div class="test-info">
    Testing: {component} | Container: {width}x{height}px
  </div>

  <div
    class="test-container"
    style:width="{width}px"
    style:height="{height}px"
    data-testid="component-container"
  >
    {#if component === 'weather'}
      <Weather />
    {:else if component === 'satword'}
      <SatWord />
    {:else if component === 'calendar'}
      <Calendar2 />
    {:else}
      <p>Unknown component: {component}</p>
    {/if}
  </div>
</div>

<style>
  .test-wrapper {
    padding: 1rem;
    background: #f0f0f0;
    min-height: 100vh;
  }

  .test-info {
    margin-bottom: 1rem;
    padding: 0.5rem;
    background: #333;
    color: white;
    font-family: monospace;
    font-size: 0.875rem;
  }

  .test-container {
    container-type: size;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    overflow: hidden;
  }
</style>
