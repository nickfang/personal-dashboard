<script>
  import { isAuthenticated } from '$lib/authService';
  import { redirect } from '@sveltejs/kit';
  import '../app.css';

  isAuthenticated.subscribe((value) => {});
</script>

<svelte:head>
  <link
    href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap"
    rel="stylesheet"
  />
</svelte:head>

<slot />

<style>
  :global(body) {
    font-family:
      'Inter',
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      Roboto,
      sans-serif;
    margin: 0;
    padding: 0;
  }
</style>
