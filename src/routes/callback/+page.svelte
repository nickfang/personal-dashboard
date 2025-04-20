<script lang="ts">
  import { onMount } from 'svelte';
  import { handleCallback } from '$lib/authService';
  import { goto } from '$app/navigation';

  onMount(async () => {
    const user = await handleCallback();
    if (user) {
      goto('/dashboard'); //post-login route
    } else {
      console.error('Authentication failed.');
    }
  });
</script>

<div>
  <h1>Processing Authentication...</h1>
  <p>You will be redirected to the dashboard shortly.</p>
</div>
