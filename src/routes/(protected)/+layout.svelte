<script>
  import { isAuthenticated } from '$lib/authService';
</script>

{#if $isAuthenticated}
  <slot />
{:else}
  <p>Not authenticated</p>
{/if}
