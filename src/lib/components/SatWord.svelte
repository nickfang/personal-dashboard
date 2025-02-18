<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { quintOut } from 'svelte/easing';
  import wordData from '$lib/data/sat-words.json';
  import { writable } from 'svelte/store';
  import SectionHeader from './SectionHeader.svelte';

  let words = Object.entries(wordData);
  const wordStore = writable<{ word: string; definitions: any[]; date: string } | null>(null);
  let currentDefinitionIndex = 0;
  let intervalId: NodeJS.Timeout;
  let progress = 0;
  let progressInterval: NodeJS.Timeout;

  function setupInterval() {
    // Clear any existing interval
    if (intervalId) clearInterval(intervalId);
    if (progressInterval) clearInterval(progressInterval);
    
    // Reset index and progress
    currentDefinitionIndex = 0;
    progress = 0;

    // Start new interval if multiple definitions
    if ($wordStore && $wordStore.definitions.length > 1) {
      intervalId = setInterval(cycleDefinitions, 8000);
      // Update progress every 80ms (100 steps over 8 seconds)
      progressInterval = setInterval(() => {
        progress = Math.min(100, progress + 1);
      }, 80);
    }
  }

  function getRandomWord() {
    const randomIndex = Math.floor(Math.random() * words.length);
    const [word, definitions] = words[randomIndex];
    const today = new Date().toLocaleDateString();
    wordStore.set({ word, definitions, date: today });
    setupInterval();
  }

  function cycleDefinitions() {
    if ($wordStore && $wordStore.definitions.length > 1) {
      currentDefinitionIndex = (currentDefinitionIndex + 1) % $wordStore.definitions.length;
      progress = 0;
    }
  }

  onMount(() => {
    // Check if we have a stored word and if it's from today
    const today = new Date().toLocaleDateString();
    const storedWord = localStorage.getItem('satWord');

    if (storedWord) {
      const parsed = JSON.parse(storedWord);
      if (parsed.date === today) {
        wordStore.set(parsed);
        setupInterval();
        return;
      }
    }

    // If no stored word or it's from a different day, get a new word
    getRandomWord();

    return () => {
      if (intervalId) clearInterval(intervalId);
      if (progressInterval) clearInterval(progressInterval);
    };
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
      {#if $wordStore.definitions.length > 1}
        <div class="definition-counter">
          {currentDefinitionIndex + 1} / {$wordStore.definitions.length}
          <div class="progress-container">
            <div class="progress-bar" style="width: {progress}%"></div>
          </div>
        </div>
      {/if}
    </div>

    {#key currentDefinitionIndex}
      {#if $wordStore && $wordStore.definitions[currentDefinitionIndex]}
        {@const currentDef = $wordStore.definitions[currentDefinitionIndex]}
        <div class="definition-block"
          in:fly={{ x: 300, duration: 600, easing: quintOut }}
          out:fly={{ x: -300, duration: 600, easing: quintOut }}
        >
          <div class="type">({currentDef.type})</div>
          <div class="definition">{currentDef.definition}</div>
          <div class="example">"{currentDef.example}"</div>
        </div>
      {/if}
    {/key}
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

  .definition-counter {
    font-size: 0.875rem;
    color: var(--teal-600);
    margin-top: -0.5rem;
    margin-bottom: 0.5rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
  }

  .progress-container {
    width: 100px;
    height: 1px;
    background: var(--teal-100);
    overflow: hidden;
  }

  .progress-bar {
    height: 100%;
    background: var(--teal-600);
    transition: width 80ms linear;
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
      font-size: 0.75rem;
      padding: 0.25rem 0.75rem;
    }

    .definition {
      font-size: 1.25rem;
      margin-bottom: 0.5rem;
      line-height: 1.5;
    }

    .example {
      font-size: 1rem;
      padding: 0.75rem;
      line-height: 1.4;
    }

    .definition-counter {
      font-size: 0.65rem;
      margin-top: -0.25rem;
      margin-bottom: 0.25rem;
    }

    .progress-container {
      width: 60px;
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
