<script lang="ts">
  import { onMount } from 'svelte';
  import { fullscreenStore } from '$lib/stores/fullscreen';
  import { Maximize2, Minimize2, RefreshCw } from 'lucide-svelte';
  import wordData from '$lib/data/sat-words.json';

  let words = Object.entries(wordData);
  let word: string;
  let definitions: any[];

  function getRandomWord() {
    const randomIndex = Math.floor(Math.random() * words.length);
    [word, definitions] = words[randomIndex];
  }

  // Initialize with a random word
  getRandomWord();

  function toggleFullscreen() {
    $fullscreenStore = $fullscreenStore === 'satword' ? false : 'satword';
  }
</script>

<style>
  .word-container {
    padding: 2rem;
    max-width: 800px;
    margin: 0 auto;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 3rem;
    border-bottom: 2px solid var(--teal-100);
    padding-bottom: 1rem;
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

  .toggle-btn, .refresh-btn {
    padding: 0.625rem;
    border: none;
    background: none;
    border-radius: 9999px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .toggle-btn:hover, .refresh-btn:hover {
    background-color: var(--teal-50);
    transform: translateY(-1px);
  }

  .refresh-btn:active {
    transform: rotate(180deg);
  }

  .word-section {
    text-align: center;
    margin-bottom: 3rem;
  }

  .word {
    font-size: 4rem;
    font-weight: 600;
    color: var(--teal-800);
    margin-bottom: 0.5rem;
    letter-spacing: -0.03em;
    line-height: 1;
  }

  .definition-block {
    margin-bottom: 2rem;
    padding: 2rem;
    background: var(--teal-50);
    border-radius: 1rem;
    border: 1px solid var(--teal-100);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }

  .definition-block:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 12px -2px rgba(0, 0, 0, 0.1);
  }

  .definition-block:last-child {
    margin-bottom: 0;
  }

  .type {
    display: inline-block;
    color: var(--teal-600);
    font-style: italic;
    margin-bottom: 1.25rem;
    font-size: 1.1rem;
    padding: 0.25rem 1rem;
    background: white;
    border-radius: 9999px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .definition {
    color: var(--gray-800);
    margin-bottom: 1.75rem;
    line-height: 1.6;
    font-size: 1.5rem;
    font-weight: 500;
    letter-spacing: -0.01em;
  }

  .example {
    color: var(--teal-600);
    font-style: italic;
    line-height: 1.6;
    padding: 1.25rem;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
    font-size: 1.1rem;
  }

  @media (max-width: 640px) {
    .word-container {
      padding: 1.5rem;
    }

    .word {
      font-size: 3rem;
    }

    .definition {
      font-size: 1.25rem;
    }
  }
</style>

<div class="word-container">
  <div class="header">
    <h2>Word of the Day</h2>
    <div class="header-buttons">
      <button class="refresh-btn" on:click={getRandomWord}>
        <RefreshCw size={20} color="var(--teal-600)" />
      </button>
      <button class="toggle-btn" on:click={toggleFullscreen}>
        {#if $fullscreenStore === 'satword'}
          <Minimize2 size={20} color="var(--teal-600)" />
        {:else}
          <Maximize2 size={20} color="var(--teal-600)" />
        {/if}
      </button>
    </div>
  </div>

  <div class="word-section">
    <div class="word">{word}</div>
  </div>
  
  {#each definitions as { type, definition, example }, i}
    <div class="definition-block">
      {#if definitions.length > 1}
        <div class="type">({i + 1}. {type})</div>
      {:else}
        <div class="type">({type})</div>
      {/if}
      <div class="definition">{definition}</div>
      <div class="example">"{example}"</div>
    </div>
  {/each}
</div> 