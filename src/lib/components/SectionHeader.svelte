<script lang="ts">
  import { Maximize2 } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';

  export let title: string;
  export let fullscreenPath: string;

  $: isFullscreen = $page.url.pathname.includes('fullscreen');
</script>

<div class="header">
  <h2>{title}</h2>
  {#if !isFullscreen}
    <button class="toggle-btn" on:click={() => goto(fullscreenPath)}>
      <Maximize2 size={20} color="var(--teal-600)" />
    </button>
  {/if}
</div>

<style>
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    border-bottom: 2px solid var(--teal-100);
    padding-bottom: 1rem;
    height: 3.5rem; /* approximately 56px */
  }

  h2 {
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--teal-800);
    margin: 0;
    letter-spacing: -0.02em;
  }

  .toggle-btn {
    padding: 0.625rem;
    border: none;
    background: none;
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .toggle-btn:hover {
    background-color: var(--teal-50);
    transform: translateY(-1px);
  }
</style>
