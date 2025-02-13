<script>
  import { ArrowLeft } from 'lucide-svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';

  $: isFullscreen = $page.url.pathname.includes('fullscreen');
</script>

<style>
  header {
    background-color: white;
    border-bottom: 1px solid var(--teal-100);
    padding: 0.75rem 1.5rem;
  }

  .header-content {
    display: flex;
    align-items: center;
    max-width: 2000px;
    margin: 0 auto;
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--teal-800);
    margin: 0;
    letter-spacing: -0.02em;
  }

  .back-btn {
    padding: 0.5rem;
    margin-right: 1rem;
    border: none;
    background: none;
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
  }

  .back-btn:hover {
    background-color: var(--teal-50);
  }
</style>

<header>
  <div class="header-content">
    {#if isFullscreen}
      <button class="back-btn" on:click={() => goto('/dashboard')}>
        <ArrowLeft size={20} color="var(--teal-600)" />
      </button>
    {/if}
    <h1>Family Dashboard</h1>
  </div>
</header>
