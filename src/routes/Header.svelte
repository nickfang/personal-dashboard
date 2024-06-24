<script>
  import { page } from '$app/stores';

  const activeClasses = 'underline text-indigo-600';
  const inactiveClasses = 'text-gray-600';
</script>

<header class="flex justify-between container py-4 border-b border-b-slate-200">
  <h1 class="font-bold text-gray-900 uppercase tracking-widest">Dashboard</h1>

  <nav class="flex flex-nowrap gap-4">
    <a href="/" class={$page.url.pathname === '/' ? activeClasses : inactiveClasses}>Home</a>

    <!-- <a href="/totals" class={$page.url.pathname === '/totals' ? activeClasses : inactiveClasses}>
      Totals
    </a>

    <a
      href="/transactions"
      class={$page.url.pathname === '/transactions' ? activeClasses : inactiveClasses}
    >
      Transactions
    </a> -->
  </nav>
</header>
