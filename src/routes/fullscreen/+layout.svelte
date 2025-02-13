<script>
  import Header from '../Header.svelte';
  import { page } from '$app/stores';
  import { ArrowLeft, ArrowRight } from 'lucide-svelte';

  const sections = [
    { path: '/fullscreen/weather', title: 'Weather' },
    { path: '/fullscreen/calendar', title: 'Calendar' },
    { path: '/fullscreen/sat-word', title: 'Word of the Day' }
  ];

  $: currentPath = $page.url.pathname;
  $: currentIndex = sections.findIndex(section => section.path === currentPath);
  $: prevSection = sections[currentIndex - 1] || sections[sections.length - 1];
  $: nextSection = sections[currentIndex + 1] || sections[0];
</script>

<Header />
<main>
  <slot />
</main>

<nav>
  <div class="nav-content">
    <a href={prevSection.path} class="nav-button prev">
      <ArrowLeft size={24} />
      <span>{prevSection.title}</span>
    </a>
    <a href={nextSection.path} class="nav-button next">
      <span>{nextSection.title}</span>
      <ArrowRight size={24} />
    </a>
  </div>
</nav>

<style>
  main {
    height: calc(100vh - 136px); /* Adjusted for larger nav */
    overflow: hidden;
    padding: 1.5rem;
  }

  nav {
    background-color: var(--teal-50);
    border-top: 1px solid var(--teal-100);
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
  }

  .nav-content {
    max-width: 2000px;
    margin: 0 auto;
    padding: 1rem 1.5rem;
    display: flex;
    justify-content: space-between;
  }

  .nav-button {
    color: var(--teal-800);
    text-decoration: none;
    padding: 0.75rem 1.5rem;
    border-radius: 9999px;
    font-size: 1rem;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    background-color: white;
    border: 1px solid var(--teal-100);
  }

  .nav-button:hover {
    background-color: var(--teal-100);
  }

  .prev {
    padding-right: 2rem;
  }

  .next {
    padding-left: 2rem;
  }
</style> 