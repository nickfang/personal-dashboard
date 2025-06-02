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
        // Redirect to the original intended page or dashboard.
        const stateFromUrl = $page.url.searchParams.get('state');
        console.log('[Callback Page] State from URL:', stateFromUrl);
        const redirectTo = stateFromUrl || '/dashboard'; // OIDC often puts original URL in 'state' or you might need to retrieve it differently
        await goto(redirectTo, { replaceState: true });
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
