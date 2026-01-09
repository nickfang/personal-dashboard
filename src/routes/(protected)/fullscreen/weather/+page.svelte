<script lang="ts">
  import Weather from '$lib/components/Weather.svelte';
</script>

<div class="fullscreen-wrapper">
  <Weather />
</div>

<style>
  .fullscreen-wrapper {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
</style>
