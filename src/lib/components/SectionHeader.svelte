<script lang="ts">
  import { Maximize2 } from 'lucide-svelte';
  import { RefreshCw } from 'lucide-svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';

  export let title: string;
  export let fullscreenPath: string;
  export let onRefresh: (() => void) | undefined = undefined;

  $: isFullscreen = $page.url.pathname.includes('fullscreen');
</script>

<div class="header">
  <h2>{title}</h2>
  <div class="header-buttons">
    {#if onRefresh}
      <button class="refresh-btn" on:click={onRefresh}>
        <RefreshCw size={20} color="var(--teal-600)" />
      </button>
    {/if}
    {#if !isFullscreen}
      <button class="toggle-btn" on:click={() => goto(fullscreenPath)}>
        <Maximize2 size={20} color="var(--teal-600)" />
      </button>
    {/if}
  </div>
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

  .header-buttons {
    display: flex;
    gap: 0.75rem;
  }

  .refresh-btn,
  .toggle-btn {
    padding: 0.625rem;
    border: none;
    background: none;
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .refresh-btn:hover,
  .toggle-btn:hover {
    background-color: var(--teal-50);
    transform: translateY(-1px);
  }

  .refresh-btn:active {
    transform: rotate(180deg);
  }

  @media (max-width: 1360px) and (max-height: 768px) {
    .header {
      margin-bottom: 0.25rem;
      padding-bottom: 0.25rem;
      height: 2rem;
      border-bottom-width: 1px;
    }
    
    h2 {
      font-size: 1.125rem;  /* 18px - same as other sections */
      font-weight: 500;
    }

    .toggle-btn {
      padding: 0.375rem;
      transform: scale(0.9);
    }
  }
</style>
