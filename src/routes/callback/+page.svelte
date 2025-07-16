<script lang="ts">
  import { onMount } from 'svelte';
  import { handleCallback } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';

  let error: any = null;

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
    } catch (e) {
      console.error('Callback page error:', e);
      error = e;
    }
  });
</script>

{#if error}
  <p>Error during login: {error.message || error}</p>
  <a href="/login">Try logging in again</a>
{:else}
  <p>Processing login...</p>
{/if}
