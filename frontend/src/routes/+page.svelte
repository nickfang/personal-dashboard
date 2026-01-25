<script lang="ts">
  import { isAuthenticated, checkAuthStatus } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  onMount(async () => {
    // Check authentication status on page load
    await checkAuthStatus();
    
    // Subscribe to authentication changes
    isAuthenticated.subscribe((value) => {
      if (value) {
        // If authenticated, redirect to dashboard
        goto('/dashboard', { replaceState: true });
      } else {
        // If not authenticated, redirect to login
        goto('/login', { replaceState: true });
      }
    });
  });
</script>

<div class="loading-container">
  <div class="loading-content">
    <div class="spinner"></div>
    <p>Loading dashboard...</p>
  </div>
</div>

<style>
  .loading-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }

  .loading-content {
    text-align: center;
    color: white;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: white;
    animation: spin 1s ease-in-out infinite;
    margin: 0 auto 1rem auto;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  p {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 500;
  }
</style>