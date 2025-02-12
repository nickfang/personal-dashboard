<script lang="ts">
  import { onMount } from 'svelte';
  import { fullscreenStore } from '$lib/stores/fullscreen';
  import { Maximize2, Minimize2 } from 'lucide-svelte';

  function toggleFullscreen() {
    $fullscreenStore = $fullscreenStore === 'calendar' ? false : 'calendar';
  }
</script>

<div class="p-6">
  <div class="flex justify-between items-center mb-6">
    <h2 class="text-2xl font-semibold text-teal-800">Calendar</h2>
    <button 
      class="p-2 hover:bg-teal-50 rounded-full transition-colors"
      on:click={toggleFullscreen}
    >
      {#if $fullscreenStore === 'calendar'}
        <Minimize2 class="w-5 h-5 text-teal-600" />
      {:else}
        <Maximize2 class="w-5 h-5 text-teal-600" />
      {/if}
    </button>
  </div>

  <!-- Calendar content placeholder -->
  <div class="h-64 flex items-center justify-center text-teal-600">
    Calendar Component (Coming Soon)
  </div>
</div>  
