<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import Calendar from '$lib/components/Calendar.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
  import { fade } from 'svelte/transition';
  import { fullscreenStore } from '$lib/stores/fullscreen';
</script>

<div class="container">
  <main class="grid">
    {#if !$fullscreenStore || $fullscreenStore === 'weather'}
      <div 
        class="card"
        class:fullscreen={$fullscreenStore === 'weather'}
        transition:fade
      >
        <Weather />
      </div>
    {/if}

    {#if !$fullscreenStore || $fullscreenStore === 'calendar'}
      <div 
        class="card"
        class:fullscreen={$fullscreenStore === 'calendar'}
        transition:fade
      >
        <Calendar />
      </div>
    {/if}

    {#if !$fullscreenStore || $fullscreenStore === 'satword'}
      <div 
        class="card"
        class:fullscreen={$fullscreenStore === 'satword'}
        transition:fade
      >
        <SatWord />
      </div>
    {/if}
  </main>
</div>
