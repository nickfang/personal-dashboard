<script>
  export let data;
</script>

{#if data.isAuthenticated}
  <slot />
{:else}
  <div>
    <div class="container">
      <h1 class="title">Uh oh, something went wrong</h1>
      <p class="message">
        You are not authenticated to view this page. If not redirected automatically, click the
        button below to go to the login page.
      </p>
      <a href="/login" class="login-link"> Go to Login </a>
    </div>
  </div>
{/if}

<style>
  .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background-color: #f3f4f6;
  }
  .title {
    font-size: 2.25rem;
    font-weight: bold;
    color: #1f2937;
    margin-bottom: 1rem;
  }
  .message {
    font-size: 1.125rem;
    color: #4b5563;
    margin-bottom: 2rem;
    text-align: center;
    max-width: 28rem;
  }
  .login-link {
    padding: 0.75rem 1.5rem;
    background-color: #3b82f6;
    color: #fff;
    font-weight: 600;
    border-radius: 0.375rem;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    transition: background-color 0.3s;
    text-decoration: none;
    display: inline-block;
  }
  .login-link:hover {
    background-color: #2563eb;
  }
</style>
