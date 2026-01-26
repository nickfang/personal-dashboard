<script lang="ts">
  import { onMount } from 'svelte';
  import { handleCallback } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';

  let error: any = null;
  let isStateError = false;

  function clearAuthState() {
    // Clear all OIDC-related storage
    const keysToRemove: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key && (key.startsWith('oidc.') || key.includes('state') || key.includes('nonce'))) {
        keysToRemove.push(key);
      }
    }
    keysToRemove.forEach(key => sessionStorage.removeItem(key));

    // Also clear from localStorage
    const localKeysToRemove: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && (key.startsWith('oidc.') || key.includes('state') || key.includes('nonce'))) {
        localKeysToRemove.push(key);
      }
    }
    localKeysToRemove.forEach(key => localStorage.removeItem(key));

    console.log('[Callback] Cleared stale auth state');
  }

  function retryLogin() {
    clearAuthState();
    window.location.href = '/login';
  }

  onMount(async () => {
    try {
      const user = await handleCallback();
      if (user) {
        // Successfully handled callback and server session should be created.
        // Always redirect to dashboard after successful login
        console.log('[Callback Page] User authenticated, redirecting to dashboard');
        await goto('/dashboard', { replaceState: true });
      } else {
        error = 'Login failed or was cancelled.';
      }
    } catch (e: any) {
      console.error('Callback page error:', e);
      error = e;
      // Check if this is a state mismatch error
      if (e?.message?.includes('state') || e?.message?.includes('No matching')) {
        isStateError = true;
      }
    }
  });
</script>

{#if error}
  <div class="callback-container">
    <div class="callback-card">
      <p class="error-message">Error during login: {error.message || error}</p>
      {#if isStateError}
        <p class="error-hint">This can happen if you had a previous login attempt. Click below to clear the stale session and try again.</p>
        <button on:click={retryLogin} class="retry-button">
          Clear Session & Retry
        </button>
      {/if}
      <a href="/login" class="login-link">Try logging in again</a>
    </div>
  </div>
{:else}
  <div class="callback-container">
    <div class="callback-card">
      <p class="processing-message">Processing login...</p>
    </div>
  </div>
{/if}

<style>
  .callback-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(to bottom right, var(--teal-50, #d1efef), var(--teal-100, #a3dfdf));
    padding: 2rem;
  }

  .callback-card {
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    padding: 3rem;
    max-width: 500px;
    width: 100%;
    text-align: center;
  }

  .error-message {
    color: var(--gray-800, #1f2937);
    font-size: 1.125rem;
    margin: 0 0 1rem 0;
  }

  .error-hint {
    color: var(--teal-600, #006666);
    margin: 1rem 0;
    font-size: 0.875rem;
  }

  .retry-button {
    background: var(--teal-600, #006666);
    color: white;
    border: none;
    border-radius: 0.375rem;
    padding: 0.75rem 1.5rem;
    font-size: 1rem;
    cursor: pointer;
    margin-right: 1rem;
    transition: background-color 0.2s ease;
  }

  .retry-button:hover {
    background: var(--teal-800, #004444);
  }

  .login-link {
    color: var(--teal-600, #006666);
    text-decoration: underline;
  }

  .login-link:hover {
    color: var(--teal-800, #004444);
  }

  .processing-message {
    color: var(--teal-600, #006666);
    font-size: 1.125rem;
    margin: 0;
  }
</style>
