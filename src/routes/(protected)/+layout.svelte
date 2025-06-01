<script>
  import { isAuthenticated } from '$lib/authService';
  import { onMount } from 'svelte';

  let authenticated = false;

  onMount(() => {
    isAuthenticated.subscribe((value) => {
      console.log("layout::onMount::isAuthenticated", value);
    });
  })
</script>

{#if $isAuthenticated}
  <slot />
{:else}
  <p>Not authenticated</p>
{/if}
