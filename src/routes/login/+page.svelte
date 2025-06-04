<script lang="ts">
  import { isAuthenticated, startSignIn, startSignOut, user } from '$lib/authService';
  // Client-side redirect is no longer needed here if server-side redirect is working
  // import { goto } from '$app/navigation';
  // import { page } from '$app/stores';
  import { onMount } from 'svelte';

  onMount(() => {
    // Optional: Subscribe to changes in authentication status
    isAuthenticated.subscribe((value) => {
      console.log('Login onMount, Is authenticated:', value);
    });

    user.subscribe((currentUser) => {
      console.log('Current user:', currentUser);
    });
    console.log('Environment:', import.meta.env.MODE);
  });
</script>

<nav>
  <a href="/">Home</a>
  {#if $isAuthenticated}
    <span>Welcome, {$user?.profile?.name || $user?.profile?.email || 'User'}</span>
    <button on:click={startSignOut}>Sign Out</button>
  {:else}
    <button on:click={startSignIn}>Sign In</button>
  {/if}
</nav>

<main>
  <h1>Welcome to the App</h1>
  {#if $isAuthenticated}
    <p>You are logged in!</p>
  {:else}
    <p>Please log in to access the full content.</p>
  {/if}
</main>
