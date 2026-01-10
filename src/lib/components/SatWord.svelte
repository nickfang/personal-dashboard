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
  let isAutoCycling = true;
  let userInteractionTimer: NodeJS.Timeout;

  // Container dimensions for responsive behavior (bound to DOM)
  let containerHeight = 0;
  let containerWidth = 0;

  // Reactive: determine display mode based on container size
  $: definitionCount = $wordStore?.definitions?.length ?? 0;
  $: showAllDefinitions = computeShowAllDefinitions(
    containerHeight,
    containerWidth,
    definitionCount
  );

  function computeShowAllDefinitions(height: number, width: number, count: number): boolean {
    if (count <= 1) return true;
    // Container height thresholds (matching CSS container queries)
    if (height < 350) return false; // Compact: always cycle
    if (height < 450) return count <= 2; // Medium: show up to 2
    return count <= 3; // Large: show up to 3
  }

  // Reactive: manage cycling when display mode changes
  $: if ($wordStore) {
    manageCycling(showAllDefinitions, isAutoCycling);
  }

  function manageCycling(showAll: boolean, autoCycle: boolean) {
    // Clear existing intervals
    if (intervalId) clearInterval(intervalId);
    if (progressInterval) clearInterval(progressInterval);

    // Reset state
    currentDefinitionIndex = 0;
    progress = 0;

    // Start cycling only when not showing all and auto-cycling is enabled
    if (!showAll && $wordStore && $wordStore.definitions.length > 1 && autoCycle) {
      intervalId = setInterval(cycleDefinitions, 10000);
      progressInterval = setInterval(() => {
        if (isAutoCycling) {
          progress = Math.min(100, progress + 1);
        }
      }, 100);
    }
  }

  function getRandomWord() {
    const randomIndex = Math.floor(Math.random() * words.length);
    const [word, definitions] = words[randomIndex];
    const today = new Date().toLocaleDateString();
    wordStore.set({ word, definitions, date: today });
    // Cycling is managed reactively via $: manageCycling()
  }

  function cycleDefinitions() {
    if (!showAllDefinitions && $wordStore && $wordStore.definitions.length > 1 && isAutoCycling) {
      currentDefinitionIndex = (currentDefinitionIndex + 1) % $wordStore.definitions.length;
      progress = 0;
    }
  }

  function previousDefinition() {
    if (!showAllDefinitions && $wordStore && $wordStore.definitions.length > 1) {
      pauseAutoCycling();
      currentDefinitionIndex =
        currentDefinitionIndex === 0
          ? $wordStore.definitions.length - 1
          : currentDefinitionIndex - 1;
    }
  }

  function nextDefinition() {
    if (!showAllDefinitions && $wordStore && $wordStore.definitions.length > 1) {
      pauseAutoCycling();
      currentDefinitionIndex = (currentDefinitionIndex + 1) % $wordStore.definitions.length;
    }
  }

  function pauseAutoCycling() {
    isAutoCycling = false;
    progress = 0;
    if (intervalId) clearInterval(intervalId);
    if (progressInterval) clearInterval(progressInterval);
    if (userInteractionTimer) clearTimeout(userInteractionTimer);

    // Resume auto-cycling after 15 seconds of no interaction
    userInteractionTimer = setTimeout(() => {
      isAutoCycling = true;
      // The reactive $: manageCycling() will restart the intervals
    }, 15000);
  }

  function toggleAutoCycling() {
    isAutoCycling = !isAutoCycling;
    if (!isAutoCycling) {
      if (intervalId) clearInterval(intervalId);
      if (progressInterval) clearInterval(progressInterval);
      progress = 0;
    }
    // When turning on, the reactive $: manageCycling() handles it
  }

  onMount(() => {
    // Check if we have a stored word from today
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

    // Cleanup on unmount
    return () => {
      if (intervalId) clearInterval(intervalId);
      if (progressInterval) clearInterval(progressInterval);
      if (userInteractionTimer) clearTimeout(userInteractionTimer);
    };
  });

  // Subscribe to store changes to save to localStorage
  $: if ($wordStore) {
    localStorage.setItem('satWord', JSON.stringify($wordStore));
  }
</script>

<div class="word-container" bind:clientHeight={containerHeight} bind:clientWidth={containerWidth}>
  <SectionHeader
    title="Word of the Day"
    fullscreenPath="/fullscreen/sat-word"
    onRefresh={getRandomWord}
  />

  {#if $wordStore}
    <div class="word-section">
      <div class="word">{$wordStore.word}</div>
      {#if $wordStore.definitions.length > 1 && !showAllDefinitions}
        <div class="definition-counter">
          {currentDefinitionIndex + 1} / {$wordStore.definitions.length}
          <div class="progress-container">
            <div class="progress-bar" style:width="{progress}%"></div>
          </div>
        </div>
      {:else if $wordStore.definitions.length > 2}
        <div class="definition-counter-static">
          {$wordStore.definitions.length} definitions
        </div>
      {/if}
    </div>

    {#if showAllDefinitions}
      <!-- Show all definitions at once -->
      <div class="all-definitions" class:single-definition={$wordStore.definitions.length === 1}>
        {#each $wordStore.definitions as definition, index}
          <div class="definition-block" class:multiple={$wordStore.definitions.length > 1}>
            <div class="type">({definition.type})</div>
            <div class="definition">{definition.definition}</div>
            <div class="example">"{definition.example}"</div>
          </div>
        {/each}
      </div>
    {:else}
      <!-- Cycling view for limited space -->
      {#key currentDefinitionIndex}
        {#if $wordStore && $wordStore.definitions[currentDefinitionIndex]}
          {@const currentDef = $wordStore.definitions[currentDefinitionIndex]}
          <div
            class="definition-block cycling"
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
  {/if}
</div>

<style>
  /* Large (default) styles */
  .word-container {
    padding: 1rem;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden; /* No internal scrollbar - let container queries adapt content */
  }

  .word-section {
    text-align: center;
    overflow: hidden;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 1rem 0;
  }

  .word {
    font-size: 2.5rem;
    font-weight: 600;
    color: var(--teal-800);
    margin-bottom: 0.5rem;
    letter-spacing: -0.03em;
    line-height: 1;
  }

  .definition-block {
    margin-bottom: 0.75rem;
    padding: 1rem;
    background: var(--teal-50);
    border-radius: 0.75rem;
    border: 1px solid var(--teal-100);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
  }

  .definition-block.multiple {
    margin-bottom: 0;
    padding: 0.75rem;
  }

  .definition-block.multiple .type {
    margin-bottom: 0.75rem;
    font-size: 0.875rem;
  }

  .definition-block.multiple .definition {
    margin-bottom: 1rem;
    font-size: 1.1rem;
  }

  .definition-block.multiple .example {
    padding: 0.75rem;
    font-size: 0.95rem;
  }

  .definition-block:last-child {
    margin-bottom: 0;
  }

  .type {
    display: inline-block;
    color: var(--teal-600);
    font-style: italic;
    margin-bottom: 1rem;
    font-size: 0.875rem;
    padding: 0.375rem 0.875rem;
    background: white;
    border-radius: 9999px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    max-width: fit-content;
    white-space: nowrap;
    min-width: 3.5rem;
    text-align: center;
  }

  .definition {
    color: var(--gray-800);
    margin-bottom: 1.25rem;
    line-height: 1.6;
    font-size: 1.2rem;
    font-weight: 500;
    letter-spacing: -0.01em;
  }

  .example {
    color: var(--teal-600);
    font-style: italic;
    line-height: 1.6;
    padding: 1rem;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
    font-size: 1rem;
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

  .definition-counter-static {
    font-size: 0.875rem;
    color: var(--teal-600);
    margin-top: -0.5rem;
    margin-bottom: 0.5rem;
    text-align: center;
    opacity: 0.8;
  }

  .all-definitions {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1rem;
    flex: 1;
    overflow-y: auto;
    justify-content: center;
  }

  /* Force single column when on small screens, but keep half-width for single definition on desktop */
  .all-definitions.single-column {
    grid-template-columns: 1fr;
  }

  /* For single definition, use more width but not full width */
  .all-definitions.single-definition {
    grid-template-columns: minmax(300px, 0.75fr);
    justify-content: center;
  }

  .definition-block.multiple {
    margin-bottom: 0;
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

  /*
   * CONTAINER QUERY BREAKPOINTS:
   * - 450px+ height: Full layout, all definitions visible
   * - 350-450px height: Medium, up to 2 definitions
   * - <350px height: Compact, cycling mode
   * - <500px width: Single column layout
   */

  /* Medium container height - compact spacing */
  @container (max-height: 450px) {
    .word-container {
      padding: var(--space-sm);
    }

    .word {
      font-size: var(--font-2xl);
      margin-bottom: var(--space-sm);
    }

    .word-section {
      padding: var(--space-sm) 0;
    }

    .all-definitions {
      gap: var(--space-sm);
    }

    .definition-block {
      padding: var(--space-sm);
      margin-bottom: var(--space-sm);
    }

    .definition-block.multiple {
      padding: var(--space-sm);
    }

    .definition-block.multiple .type {
      margin-bottom: var(--space-sm);
      font-size: var(--font-sm);
    }

    .definition-block.multiple .definition {
      margin-bottom: var(--space-sm);
      font-size: var(--font-base);
    }

    .definition-block.multiple .example {
      padding: var(--space-sm);
      font-size: var(--font-sm);
    }

    .type {
      margin-bottom: var(--space-sm);
      font-size: var(--font-sm);
    }

    .definition {
      font-size: var(--font-lg);
      margin-bottom: var(--space-sm);
      line-height: 1.5;
    }

    .example {
      font-size: var(--font-base);
      padding: var(--space-sm);
      line-height: 1.4;
    }

    .definition-counter,
    .definition-counter-static {
      font-size: var(--font-xs);
      margin-top: calc(-1 * var(--space-xs));
      margin-bottom: var(--space-xs);
    }

    .progress-container {
      width: 70px;
    }
  }

  /* Compact container height - very tight spacing */
  @container (max-height: 350px) {
    .word-container {
      padding: var(--space-xs);
    }

    .word {
      font-size: var(--font-xl);
      margin-bottom: var(--space-xs);
    }

    .word-section {
      padding: var(--space-xs) 0;
    }

    .definition-block {
      padding: var(--space-xs);
      margin-bottom: var(--space-xs);
    }

    .definition-block.cycling {
      padding: var(--space-sm);
    }

    .type {
      margin-bottom: var(--space-xs);
      font-size: var(--font-xs);
      padding: var(--space-xs) var(--space-sm);
    }

    .definition {
      font-size: var(--font-base);
      margin-bottom: var(--space-xs);
      line-height: 1.4;
    }

    .example {
      font-size: var(--font-sm);
      padding: var(--space-xs);
      line-height: 1.3;
    }

    .definition-counter,
    .definition-counter-static {
      font-size: 0.6rem;
      margin-top: 0;
      margin-bottom: var(--space-xs);
    }

    .progress-container {
      width: 50px;
    }
  }

  /* Very compact - minimal padding */
  @container (max-height: 250px) {
    .word {
      font-size: var(--font-lg);
      margin-bottom: 0;
    }

    .word-section {
      padding: 0;
    }

    .definition-block.cycling {
      padding: var(--space-xs);
    }

    .type {
      font-size: 0.65rem;
      padding: 0.125rem var(--space-xs);
      margin-bottom: var(--space-xs);
    }

    .definition {
      font-size: var(--font-sm);
      margin-bottom: var(--space-xs);
    }

    .example {
      font-size: var(--font-xs);
      padding: var(--space-xs);
    }
  }

  /* Narrow container - single column */
  @container (max-width: 500px) {
    .all-definitions {
      grid-template-columns: 1fr;
      gap: var(--space-sm);
    }

    .definition-block {
      padding: var(--space-md);
    }

    .definition-block.multiple {
      padding: var(--space-sm);
    }

    .definition-block.multiple .definition {
      font-size: var(--font-base);
    }

    .definition-block.multiple .example {
      font-size: var(--font-sm);
    }

    .word {
      font-size: var(--font-2xl);
    }
  }

  /* Large container height - generous spacing */
  @container (min-height: 500px) {
    .word-container {
      padding: var(--space-lg);
    }

    .word {
      font-size: 3rem;
      margin-bottom: var(--space-md);
    }

    .all-definitions {
      gap: var(--space-md);
    }

    .definition-block {
      padding: var(--space-lg);
      margin-bottom: var(--space-md);
    }

    .definition-block.multiple {
      padding: var(--space-md);
    }

    .definition {
      font-size: 1.3rem;
      margin-bottom: var(--space-md);
      line-height: 1.6;
    }

    .example {
      font-size: var(--font-lg);
      padding: var(--space-md);
      line-height: 1.5;
    }
  }
</style>
