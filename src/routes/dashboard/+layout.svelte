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
    min-height: calc(100vh - 48px);
    overflow: hidden;
  }

  .main {
    min-height: 100%;
  }

  @media (max-width: 768px) {
    .dashboard-main {
      overflow: auto;
    }
  }
</style>
