<script lang="ts">
  import { onMount } from 'svelte';
  import wordData from '$lib/data/sat-words.json';
  import { writable } from 'svelte/store';
  import SectionHeader from './SectionHeader.svelte';

  let words = Object.entries(wordData);
  const wordStore = writable<{ word: string; definitions: any[]; date: string } | null>(null);

  function getRandomWord() {
    const randomIndex = Math.floor(Math.random() * words.length);
    const [word, definitions] = words[randomIndex];
    const today = new Date().toLocaleDateString();
    wordStore.set({ word, definitions, date: today });
  }

  onMount(() => {
    // Check if we have a stored word and if it's from today
    const today = new Date().toLocaleDateString();
    const storedWord = localStorage.getItem('satWord');

    if (storedWord) {
      const parsed = JSON.parse(storedWord);
      if (parsed.date === today) {
        wordStore.set(parsed);
        return;
      }
    }

    // If no stored word or it's from a different day, get a new word
    getRandomWord();
  });

  // Subscribe to store changes to save to localStorage
  $: if ($wordStore) {
    localStorage.setItem('satWord', JSON.stringify($wordStore));
  }
</script>

<div class="word-container">
  <SectionHeader 
    title="Word of the Day" 
    fullscreenPath="/fullscreen/sat-word" 
    onRefresh={getRandomWord}
  />

  {#if $wordStore}
    <div class="word-section">
      <div class="word">{$wordStore.word}</div>
    </div>

    {#each $wordStore.definitions as { type, definition, example }, i}
      <div class="definition-block">
        {#if $wordStore.definitions.length > 1}
          <div class="type">({i + 1}. {type})</div>
        {:else}
          <div class="type">({type})</div>
        {/if}
        <div class="definition">{definition}</div>
        <div class="example">"{example}"</div>
      </div>
    {/each}
  {/if}
</div>

<style>
  /* Large (default) styles */
  .word-container {
    padding: 1.5rem;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: auto;
  }

  .word-section {
    flex: 1;
    text-align: center;
    overflow: auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 1rem 0;
  }

  .word {
    font-size: 3rem;
    font-weight: 600;
    color: var(--teal-800);
    margin-bottom: 0.75rem;
    letter-spacing: -0.03em;
    line-height: 1;
  }

  .definition-block {
    margin-bottom: 0.75rem;
    padding: 1rem;
    background: var(--teal-50);
    border-radius: .75rem;
    border: 1px solid var(--teal-100);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
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

  /* Medium (1360x768) styles */
  @media (max-width: 1360px) and (max-height: 768px) {
    .word-container {
      padding: 0.5rem;
      overflow: hidden;
    }

    .word {
      font-size: 2.5rem;
      margin-bottom: 0.5rem;
    }

    .definition-block {
      margin-bottom: 0.5rem;
      padding: 0.5rem;
      display: grid;
      gap: 0.5rem;
    }

    .type {
      margin-bottom: 0.5rem;
      font-size: 0.875rem;
      padding: 0.25rem 0.75rem;
    }

    .definition {
      font-size: 1.125rem;
      margin-bottom: 0.75rem;
      line-height: 1.4;
    }

    .example {
      font-size: 0.875rem;
      padding: 0.75rem;
      line-height: 1.4;
    }
  }

  /* Small (mobile) styles */
  @media (max-width: 768px) {
    .word-container {
      padding: 1rem;
    }

    .word {
      font-size: 3rem;
      margin-bottom: 1rem;
    }

    .definition-block {
      padding: 1rem;
      margin-bottom: 1rem;
    }

    .definition {
      font-size: 1.25rem;
    }

    .example {
      font-size: 1rem;
    }
  }
</style>
