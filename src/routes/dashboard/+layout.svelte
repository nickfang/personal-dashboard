<script>
  // import '../app.postcss';
  import Header from '../Header.svelte';
</script>

<Header />
<main class="dashboard-main">
  <div class="main">
    <slot />
  </div>
</main>

<style>
  .dashboard-main {
    height: calc(100vh - 48px); /* Shorter header height */
    overflow: hidden;
  }

  .main {
    height: 100%;
    overflow: hidden;
  }
</style>
