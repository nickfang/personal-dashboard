<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
  import Calendar2 from '$lib/components/Calendar2.svelte';
  import SatWord from '$lib/components/SatWord.svelte';
  import { isAuthenticated, startSignOut, user } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  const handleSignOut = async () => {
    await startSignOut();
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

  /* Large (default) styles */
  .dashboard-grid {
    display: grid;
    gap: 2rem;
    padding: 1rem;
    grid-template-columns: 1fr 1fr;
    height: 100vh;
    box-sizing: border-box;
    grid-template-rows: 72px 1fr 1fr;
    width: 100%;
    position: absolute;
    left: 0;
    right: 0;
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
    width: 100%;
  }

  /* Medium (1360x768) styles */
  @media (max-width: 1360px) and (max-height: 768px) {
    .dashboard-grid {
      gap: 1rem;
      padding: 0.5rem;
      width: 100%;
      grid-template-rows: auto auto;
      position: relative;
      height: auto;
      min-height: 100vh;
      overflow: visible;
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

  /* Large TV displays (1920px+) */
  @media (min-width: 1920px) {
    .dashboard-grid {
      gap: 3rem;
      padding: 2rem;
      grid-template-rows: 100px 1fr 1fr;
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

  /* Extra large displays (4K and above) */
  @media (min-width: 3840px) {
    .dashboard-grid {
      gap: 4rem;
      padding: 3rem;
      grid-template-rows: 120px 1fr 1fr;
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
</style>
