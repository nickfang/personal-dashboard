<script>
  // import '../app.postcss';
  import Header from '../Header.svelte';
</script>

<Header />
<main class="container py-6">
  <div class="main">
    <slot />
  </div>
</main>
