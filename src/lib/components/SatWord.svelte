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
  let isAutoCycling = true; // Track if auto-cycling is enabled
  let userInteractionTimer: NodeJS.Timeout; // Timer to resume auto-cycling after user interaction
  let showAllDefinitions = true; // Track whether to show all definitions or cycle
  let containerHeight = 0; // Track container height for responsive behavior

  function setupInterval() {
    // Clear any existing intervals
    if (intervalId) clearInterval(intervalId);
    if (progressInterval) clearInterval(progressInterval);

    // Reset index and progress
    currentDefinitionIndex = 0;
    progress = 0;

    // Determine if we should show all definitions or cycle based on space and count
    updateDisplayMode();

    // Start new interval only if cycling and multiple definitions and auto-cycling is enabled
    if (!showAllDefinitions && $wordStore && $wordStore.definitions.length > 1 && isAutoCycling) {
      intervalId = setInterval(cycleDefinitions, 10000); // 10 seconds
      // Update progress every 100ms (100 steps over 10 seconds)
      progressInterval = setInterval(() => {
        if (isAutoCycling) {
          progress = Math.min(100, progress + 1);
        }
      }, 100);
    }
  }

  function updateDisplayMode() {
    if (!$wordStore) return;

    const definitionCount = $wordStore.definitions.length;
    const isSmallScreen = window.innerWidth <= 768;
    const isMediumScreen = window.innerWidth <= 1360 && window.innerHeight <= 768;
    const aspectRatio = window.innerWidth / window.innerHeight;
    
    // Categorize aspect ratios for different display behaviors
    const isSquareish = aspectRatio >= 1.0 && aspectRatio <= 1.35; // 4:3 (1.33), 5:4 (1.25)
    const isStandard = aspectRatio > 1.35 && aspectRatio <= 1.65; // 16:10 (1.6), 1920x1200 (1.6)
    const isWidescreen = aspectRatio > 1.65 && aspectRatio <= 1.8; // 16:9 (1.78)
    const isUltrawide = aspectRatio > 1.8; // 21:9 (2.33), 2:1 (2.0), etc.

    // Show all definitions based on aspect ratio and available space:
    // - Square/4:3: More vertical space, can show more definitions
    // - Standard 16:10: Balanced approach
    // - Widescreen 16:9: Less vertical space, be more conservative
    // - Ultrawide: Much less vertical space relative to width, be most conservative
    if (definitionCount === 1) {
      showAllDefinitions = true;
    } else if (isSmallScreen) {
      showAllDefinitions = definitionCount <= 1;
    } else if (isMediumScreen) {
      showAllDefinitions = definitionCount <= 2;
    } else if (isSquareish) {
      // 4:3, 5:4 - more vertical space available
      showAllDefinitions = definitionCount <= 4;
    } else if (isStandard) {
      // 16:10, 1920x1200 - good balance
      showAllDefinitions = definitionCount <= 3;
    } else if (isWidescreen) {
      // 16:9 - less vertical space
      showAllDefinitions = definitionCount <= 2;
    } else if (isUltrawide) {
      // 21:9, 2:1+ - very wide, limited vertical space
      showAllDefinitions = definitionCount <= 2;
    } else {
      // Fallback for unusual aspect ratios
      showAllDefinitions = definitionCount <= 3;
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
      setupInterval();
    }, 15000);
  }

  function toggleAutoCycling() {
    isAutoCycling = !isAutoCycling;
    if (isAutoCycling) {
      setupInterval();
    } else {
      if (intervalId) clearInterval(intervalId);
      if (progressInterval) clearInterval(progressInterval);
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

    // Add resize listener to update display mode
    const handleResize = () => {
      if ($wordStore) {
        updateDisplayMode();
        setupInterval(); // Restart intervals if needed
      }
    };
    window.addEventListener('resize', handleResize);

    return () => {
      if (intervalId) clearInterval(intervalId);
      if (progressInterval) clearInterval(progressInterval);
      if (userInteractionTimer) clearTimeout(userInteractionTimer);
      window.removeEventListener('resize', handleResize);
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
      {#if $wordStore.definitions.length > 1 && !showAllDefinitions}
        <div class="definition-counter">
          {currentDefinitionIndex + 1} / {$wordStore.definitions.length}
          <div class="progress-container">
            <div class="progress-bar" style="width: {progress}%"></div>
          </div>
        </div>
      {:else if $wordStore.definitions.length > 1}
        <div class="definition-counter-static">
          {$wordStore.definitions.length} definitions
        </div>
      {/if}
    </div>

    {#if showAllDefinitions}
      <!-- Show all definitions at once -->
      <div class="all-definitions">
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
            class="definition-block"
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
    overflow: auto;
  }

  .word-section {
    text-align: center;
    overflow: auto;
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
    font-size: 0.9rem;
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
    font-size: 1rem;
    padding: 0.25rem 1rem;
    background: white;
    border-radius: 9999px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
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
    display: flex;
    flex-direction: column;
    gap: 1rem;
    flex: 1;
    overflow-y: auto;
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

    .all-definitions {
      gap: 0.5rem;
    }

    .definition-block {
      margin-bottom: 0.5rem;
      padding: 0.5rem;
      display: grid;
      gap: 0.5rem;
    }

    .definition-block.multiple {
      padding: 0.4rem;
    }

    .definition-block.multiple .type {
      margin-bottom: 0.4rem;
      font-size: 0.7rem;
    }

    .definition-block.multiple .definition {
      margin-bottom: 0.4rem;
      font-size: 1rem;
    }

    .definition-block.multiple .example {
      padding: 0.5rem;
      font-size: 0.85rem;
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

    .definition-counter,
    .definition-counter-static {
      font-size: 0.65rem;
      margin-top: -0.25rem;
      margin-bottom: 0.25rem;
    }

    .progress-container {
      width: 60px;
    }
  }

  /* Square-ish aspect ratios (4:3, 5:4) - more vertical space */
  @media (min-aspect-ratio: 1/1) and (max-aspect-ratio: 4/3) {
    .word-container {
      padding: 1.25rem;
    }

    .word {
      font-size: 3.5rem;
      margin-bottom: 1.25rem;
    }

    .all-definitions {
      gap: 1rem;
    }

    .definition-block {
      margin-bottom: 1rem;
      padding: 1.25rem;
    }

    .definition-block.multiple {
      padding: 1rem;
    }

    .type {
      margin-bottom: 1rem;
      font-size: 0.9rem;
      padding: 0.4rem 1rem;
    }

    .definition {
      font-size: 1.5rem;
      margin-bottom: 1rem;
      line-height: 1.6;
    }

    .example {
      font-size: 1.2rem;
      padding: 1rem;
      line-height: 1.5;
    }
  }

  /* Standard widescreen (16:10, 1920x1200) - balanced */
  @media (min-aspect-ratio: 4/3) and (max-aspect-ratio: 8/5) {
    .word-container {
      padding: 1rem;
    }

    .word {
      font-size: 3.25rem;
      margin-bottom: 1rem;
    }

    .all-definitions {
      gap: 0.875rem;
    }

    .definition-block {
      margin-bottom: 0.875rem;
      padding: 1rem;
    }

    .definition-block.multiple {
      padding: 0.8rem;
    }

    .type {
      font-size: 0.85rem;
      padding: 0.35rem 0.9rem;
    }

    .definition {
      font-size: 1.4rem;
      line-height: 1.55;
    }

    .example {
      font-size: 1.15rem;
      padding: 0.9rem;
    }
  }

  /* Widescreen 16:9 - less vertical space */
  @media (min-aspect-ratio: 8/5) and (max-aspect-ratio: 16/9) {
    .word-container {
      padding: 0.75rem;
      overflow: hidden;
    }

    .word {
      font-size: 2.75rem;
      margin-bottom: 0.75rem;
    }

    .all-definitions {
      gap: 0.75rem;
    }

    .definition-block {
      margin-bottom: 0.75rem;
      padding: 0.75rem;
    }

    .definition-block.multiple {
      padding: 0.6rem;
    }

    .definition-block.multiple .type {
      margin-bottom: 0.5rem;
      font-size: 0.75rem;
    }

    .definition-block.multiple .definition {
      margin-bottom: 0.5rem;
      font-size: 1.1rem;
    }

    .definition-block.multiple .example {
      padding: 0.6rem;
      font-size: 0.9rem;
    }

    .type {
      margin-bottom: 0.75rem;
      font-size: 0.8rem;
      padding: 0.3rem 0.8rem;
    }

    .definition {
      font-size: 1.3rem;
      margin-bottom: 0.75rem;
      line-height: 1.5;
    }

    .example {
      font-size: 1.05rem;
      padding: 0.8rem;
      line-height: 1.4;
    }

    .definition-counter,
    .definition-counter-static {
      font-size: 0.7rem;
      margin-top: -0.5rem;
      margin-bottom: 0.5rem;
    }

    .progress-container {
      width: 65px;
    }
  }

  /* Ultrawide aspect ratios (21:9, 2:1+) - very limited vertical space */
  @media (min-aspect-ratio: 16/9) {
    .word-container {
      padding: 0.5rem;
      overflow: hidden;
    }

    .word {
      font-size: 2.25rem;
      margin-bottom: 0.5rem;
    }

    .all-definitions {
      gap: 0.5rem;
    }

    .definition-block {
      margin-bottom: 0.5rem;
      padding: 0.5rem;
      display: grid;
      gap: 0.5rem;
    }

    .definition-block.multiple {
      padding: 0.4rem;
    }

    .definition-block.multiple .type {
      margin-bottom: 0.3rem;
      font-size: 0.65rem;
    }

    .definition-block.multiple .definition {
      margin-bottom: 0.3rem;
      font-size: 0.95rem;
    }

    .definition-block.multiple .example {
      padding: 0.4rem;
      font-size: 0.8rem;
    }

    .type {
      margin-bottom: 0.5rem;
      font-size: 0.7rem;
      padding: 0.25rem 0.6rem;
    }

    .definition {
      font-size: 1.1rem;
      margin-bottom: 0.5rem;
      line-height: 1.4;
    }

    .example {
      font-size: 0.9rem;
      padding: 0.6rem;
      line-height: 1.3;
    }

    .definition-counter,
    .definition-counter-static {
      font-size: 0.6rem;
      margin-top: -0.25rem;
      margin-bottom: 0.25rem;
    }

    .progress-container {
      width: 55px;
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

    .all-definitions {
      gap: 0.75rem;
    }

    .definition-block {
      padding: 1rem;
      margin-bottom: 1rem;
    }

    .definition-block.multiple {
      padding: 0.75rem;
    }

    .definition-block.multiple .definition {
      font-size: 1.1rem;
    }

    .definition-block.multiple .example {
      font-size: 0.95rem;
    }

    .definition {
      font-size: 1.25rem;
    }

    .example {
      font-size: 1rem;
    }
  }
</style>
