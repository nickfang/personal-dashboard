<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import Calendar2 from '$lib/components/Calendar2.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
  import { isAuthenticated, startSignOutComplete, user } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  const handleSignOut = async () => {
    await startSignOutComplete();
  };

  onMount(() => {
    // Any initialization code can go here
  });
</script>

<div class="dashboard-grid">
  <div class="nav">
    <div class="nav-content">
      <div class="nav-left">
        <h1 class="dashboard-title">Personal Dashboard</h1>
      </div>
      <div class="nav-right">
        {#if $isAuthenticated && $user}
          <div class="user-info">
            <span class="user-name">
              {$user.profile?.name || $user.profile?.email || 'User'}
            </span>
            <button on:click={handleSignOut} class="logout-button">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M9 21H5a2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                <polyline points="16,17 21,12 16,7" />
                <line x1="21" y1="12" x2="9" y2="12" />
              </svg>
              Sign Out
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
  <div class="weather-section">
    <Weather />
  </div>
  <div class="word-section">
    <SatWord />
  </div>
  <div class="calendar-section">
    <Calendar2 />
  </div>
</div>

<style>
  /* Navigation Bar Styles */
  .nav {
    grid-column: 1 / span 2;
    background: white;
    color: var(--gray-800);
    padding: 1rem 2rem;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  }

  .nav-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    max-width: 100%;
  }

  .nav-left {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .dashboard-title {
    font-size: 1.5rem;
    font-weight: 600;
    margin: 0;
    color: var(--gray-800);
  }

  .nav-right {
    display: flex;
    align-items: center;
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .user-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--teal-600);
  }

  .logout-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: var(--teal-600);
    border: none;
    color: white;
    padding: 0.5rem 1rem;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.2s ease;
  }

  .logout-button:hover {
    background: var(--teal-800);
  }

  .logout-button:focus {
    outline: 2px solid var(--teal-600);
    outline-offset: 2px;
  }

  /* General overflow fix for all resolutions */
  .dashboard-grid {
    display: grid;
    gap: 2rem;
    padding: 1rem;
    grid-template-columns: 1fr 1fr;
    height: 100vh;
    box-sizing: border-box;
    grid-template-rows: 72px 1fr 1fr;
    width: 100%;
    max-width: 1800px;
    margin: 0 auto;
    position: relative;
    left: 0;
    right: 0;
    overflow: hidden;
  }

  /* Ensure all sections respect their container bounds */
  .weather-section :global(.weather-grid),
  .word-section :global(.word-container),
  .calendar-section :global(.calendar-container) {
    overflow: hidden;
  }

  .weather-section,
  .word-section,
  .calendar-section {
    overflow: hidden;
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0; /* Ensures proper flex shrinking */
  }

  .weather-section :global(*),
  .word-section :global(*),
  .calendar-section :global(*) {
    box-sizing: border-box;
  }

  .nav {
    grid-column: 1 / span 2;
    grid-row: 1;
  }

  .weather-section {
    grid-column: 1;
    grid-row: 2;
  }

  .word-section {
    grid-column: 2;
    grid-row: 2;
  }

  .calendar-section {
    grid-column: 1 / span 2;
    grid-row: 3;
  }

  .weather-section :global(.weather-container),
  .word-section :global(.word-container) {
    transform: scale(var(--scale, 1));
    transform-origin: top center;
  }

  :global(.fullscreen) {
    overflow: hidden;
  }

  :global(.fullscreen) .dashboard-grid {
    height: 100vh;
    padding: 0.5rem;
    gap: 0.5rem;
    max-width: 1800px;
    margin: 0 auto;
  }

  /* Square-ish aspect ratios (4:3, 5:4) - more vertical space available */
  @media (min-aspect-ratio: 1/1) and (max-aspect-ratio: 4/3) and (min-width: 769px) {
    .dashboard-grid {
      gap: 2.5rem;
      padding: 2rem;
      grid-template-rows: 100px 1fr 1fr;
      height: 100vh;
      overflow: hidden;
      max-width: 1800px;
      margin: 0 auto;
    }

    .nav {
      padding: 2rem 3rem;
    }

    .dashboard-title {
      font-size: 2rem;
    }

    .user-name {
      font-size: 1rem;
    }

    .logout-button {
      padding: 0.75rem 1.5rem;
      font-size: 1rem;
    }
  }

  /* Standard widescreen (16:10, 1920x1200) - balanced approach */
  @media (min-aspect-ratio: 4/3) and (max-aspect-ratio: 8/5) and (min-width: 769px) {
    .dashboard-grid {
      gap: 2rem;
      padding: 1.5rem;
      grid-template-rows: 85px 1fr 1fr;
      height: 100vh;
      overflow: hidden;
      max-width: 1800px;
      margin: 0 auto;
    }

    .nav {
      padding: 1.5rem 2.5rem;
    }

    .dashboard-title {
      font-size: 1.75rem;
    }

    .user-name {
      font-size: 0.95rem;
    }

    .logout-button {
      padding: 0.6rem 1.2rem;
      font-size: 0.9rem;
    }
  }

  /* Widescreen 16:9 - less vertical space */
  @media (min-aspect-ratio: 8/5) and (max-aspect-ratio: 16/9) and (min-width: 769px) {
    .dashboard-grid {
      gap: 1.5rem;
      padding: 1rem;
      grid-template-rows: 75px 1fr 1fr;
      height: 100vh;
      overflow: hidden;
      max-width: 1800px;
      margin: 0 auto;
    }

    .nav {
      padding: 1.25rem 2rem;
    }

    .dashboard-title {
      font-size: 1.5rem;
    }

    .user-name {
      font-size: 0.875rem;
    }

    .logout-button {
      padding: 0.5rem 1rem;
      font-size: 0.875rem;
    }
  }

  /* Ultrawide aspect ratios (21:9, 2:1+) - very limited vertical space */
  @media (min-aspect-ratio: 16/9) and (min-width: 769px) {
    .dashboard-grid {
      gap: 1rem;
      padding: 0.75rem;
      grid-template-rows: 65px 1fr 1fr;
      height: 100vh;
      overflow: hidden;
      max-width: 1800px;
      margin: 0 auto;
    }

    .weather-section,
    .word-section,
    .calendar-section {
      overflow: hidden;
    }

    .weather-section :global(.weather-grid) {
      overflow: hidden !important;
    }

    .nav {
      padding: 1rem 1.5rem;
    }

    .dashboard-title {
      font-size: 1.35rem;
    }

    .user-name {
      font-size: 0.8rem;
    }

    .logout-button {
      padding: 0.4rem 0.8rem;
      font-size: 0.8rem;
    }
  }

  /* Medium height displays (like old laptops) */
  @media (max-height: 768px) and (min-width: 769px) {
    .dashboard-grid {
      gap: 1rem;
      padding: 0.5rem;
      width: 100%;
      grid-template-rows: 60px 1fr 1fr;
      height: 100vh;
      overflow: hidden;
      max-width: 1800px;
      margin: 0 auto;
    }

    .weather-section,
    .word-section,
    .calendar-section {
      overflow: hidden;
    }

    .weather-section :global(.weather-grid) {
      overflow: hidden !important;
    }

    .nav {
      padding: 0.75rem 1.5rem;
    }

    .dashboard-title {
      font-size: 1.25rem;
    }

    .user-name {
      font-size: 0.8rem;
    }

    .logout-button {
      padding: 0.4rem 0.8rem;
      font-size: 0.8rem;
    }
  }

  /* Tablet styles */
  @media (max-width: 1024px) and (min-width: 769px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      height: auto;
      min-height: 100vh;
      padding: 1rem;
      gap: 1.5rem;
      grid-template-rows: auto auto auto auto;
      width: 100%;
      position: relative;
      overflow: visible;
    }

    .nav {
      grid-column: 1;
      grid-row: 1;
    }

    .weather-section {
      grid-column: 1;
      grid-row: 2;
      min-height: 350px;
    }

    .word-section {
      grid-column: 1;
      grid-row: 3;
      min-height: 400px;
    }

    .calendar-section {
      grid-column: 1;
      grid-row: 4;
      min-height: 600px;
    }
  }

  /* Small (mobile) styles */
  @media (max-width: 768px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      height: auto;
      min-height: 100vh;
      padding: 0.75rem;
      gap: 1rem;
      grid-template-rows: auto auto auto auto;
      width: 100%;
      position: relative;
      overflow: visible;
    }

    .nav-content {
      flex-direction: column;
      align-items: flex-start;
      gap: 1rem;
    }

    .dashboard-title {
      font-size: 1.25rem;
    }

    .nav {
      grid-column: 1;
      grid-row: 1;
      padding: 1rem 1.5rem;
    }

    .weather-section {
      grid-column: 1;
      grid-row: 2;
      min-height: 300px;
    }

    .word-section {
      grid-column: 1;
      grid-row: 3;
      min-height: 350px;
    }

    .calendar-section {
      grid-column: 1;
      grid-row: 4;
      min-height: 500px;
    }

    .weather-section,
    .word-section,
    .calendar-section {
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    }
  }

  /* Very small mobile screens */
  @media (max-width: 480px) {
    .dashboard-grid {
      padding: 0.5rem;
      gap: 0.75rem;
    }

    .nav {
      padding: 0.75rem 1rem;
    }

    .dashboard-title {
      font-size: 1.125rem;
    }

    .user-info {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.5rem;
    }

    .weather-section {
      min-height: 280px;
    }

    .word-section {
      min-height: 320px;
    }

    .calendar-section {
      min-height: 450px;
    }
  }

  /* Extra large displays (4K and above) - maintain aspect ratio logic */
  @media (min-width: 3840px) {
    /* Square-ish 4K displays */
    .dashboard-grid {
      gap: 4rem;
      padding: 3rem;
      grid-template-rows: 120px 1fr 1fr;
      max-width: 1800px;
      margin: 0 auto;
    }

    .nav {
      padding: 3rem 4rem;
    }

    .dashboard-title {
      font-size: 2.5rem;
    }

    .user-name {
      font-size: 1.25rem;
    }

    .logout-button {
      padding: 1rem 2rem;
      font-size: 1.125rem;
    }
  }

  /* Ultrawide 4K+ displays */
  @media (min-width: 3840px) and (min-aspect-ratio: 16/9) {
    .dashboard-grid {
      gap: 2.5rem;
      padding: 2rem;
      grid-template-rows: 90px 1fr 1fr;
      max-width: 1800px;
      margin: 0 auto;
    }

    .nav {
      padding: 2rem 3rem;
    }

    .dashboard-title {
      font-size: 2rem;
    }

    .user-name {
      font-size: 1rem;
    }

    .logout-button {
      padding: 0.75rem 1.5rem;
      font-size: 1rem;
    }
  }
</style>
