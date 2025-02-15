<script lang="ts">
  import SatWord from '$lib/components/SatWord.svelte';
</script>

<SatWord />

<style>
  .fullscreen-word-container {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    padding-bottom: 8rem;
  }

  .word {
    font-size: 6rem;
    font-weight: 600;
    color: var(--teal-800);
    margin-bottom: 3rem;
    text-align: center;
    letter-spacing: -0.03em;
    line-height: 1;
  }

  @media (max-width: 768px) {
    .fullscreen-word-container {
      padding: 1rem;
      padding-bottom: 7rem;
    }

    .word {
      font-size: 4rem;
      margin-bottom: 2rem;
    }
  }
</style> 