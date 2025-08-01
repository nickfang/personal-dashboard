<script lang="ts">
  import { isAuthenticated, startSignIn, user, checkAuthStatus } from '$lib/authService';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { UserManager } from 'oidc-client-ts';
  import { authSettings } from '$lib/authConfig';

  onMount(() => {
    let redirectTimeout: NodeJS.Timeout;
    const userManager = new UserManager(authSettings);

    // Check if we need to clear stale auth state
    const checkAndClearStaleAuth = async () => {
      // If we have client-side auth but got redirected here, it means server-side auth failed
      // This happens when JWT_SECRET was missing and session creation failed
      if ($isAuthenticated && $page.url.searchParams.has('redirectTo')) {
        console.log('Detected stale authentication state, clearing...');
        // Clear the stale authentication state
        user.set(null);
        isAuthenticated.set(false);
        // Clear stored tokens
        try {
          await userManager.removeUser();
        } catch (error) {
          console.error('Error clearing stale tokens:', error);
        }
        // Recheck auth status after clearing
        checkAuthStatus();
      }
    };

    checkAndClearStaleAuth();

    // If user is already authenticated, redirect to dashboard
    const unsubscribe = isAuthenticated.subscribe((value) => {
      console.log('Login page - Is authenticated:', value);
      if (value) {
        const redirectTo = $page.url.searchParams.get('redirectTo') || '/dashboard';
        console.log('Redirecting to:', redirectTo);

        // Clear any existing timeout
        if (redirectTimeout) {
          clearTimeout(redirectTimeout);
        }

        // Try immediate redirect
        goto(redirectTo, { replaceState: true }).catch((error) => {
          console.error('Redirect failed:', error);
          // Fallback to manual redirect after 3 seconds if goto fails
          redirectTimeout = setTimeout(() => {
            window.location.href = redirectTo;
          }, 3000);
        });
      }
    });

    user.subscribe((currentUser) => {
      console.log('Current user:', currentUser);
    });
    console.log('Environment:', import.meta.env.MODE);

    // Cleanup function
    return () => {
      unsubscribe();
      if (redirectTimeout) {
        clearTimeout(redirectTimeout);
      }
    };
  });

  const handleSignIn = async () => {
    try {
      await startSignIn();
    } catch (error) {
      console.error('Sign in error:', error);
    }
  };
</script>

<div class="login-container">
  <div class="login-card">
    <div class="header">
      <div class="dashboard-icon">🏠</div>
      <h1 class="title">Personal Dashboard</h1>
      <p class="subtitle">Sign in to access your dashboard</p>
    </div>

    <div class="login-content">
      {#if $isAuthenticated}
        <div class="success-message">
          <p>
            ✓ Successfully logged in as {$user?.profile?.name || $user?.profile?.email || 'User'}
          </p>
          <p>Redirecting to dashboard...</p>
          <div class="manual-redirect">
            <p>If you're not redirected automatically:</p>
            <a href="/dashboard" class="dashboard-link">Go to Dashboard</a>
          </div>
        </div>
      {:else}
        <button on:click={handleSignIn} class="signin-button">
          <svg class="google-icon" viewBox="0 0 24 24" width="20" height="20">
            <path
              fill="#4285F4"
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
            />
            <path
              fill="#34A853"
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
            />
            <path
              fill="#FBBC05"
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
            />
            <path
              fill="#EA4335"
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
            />
          </svg>
          Sign in with Google
        </button>

        <div class="info-text">
          <p><strong>Living Room Dashboard</strong></p>
          <p>Optimized for large screen displays</p>
          <p>Weather • Calendar • Daily Word • Time</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(to bottom right, var(--teal-50, #d1efef), var(--teal-100, #a3dfdf));
    padding: 2rem;
    font-family:
      'Inter',
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      Roboto,
      sans-serif;
  }

  .login-card {
    background: white;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    padding: 3rem;
    max-width: 500px;
    width: 100%;
    text-align: center;
    transition: box-shadow 0.3s ease;
  }

  .login-card:hover {
    box-shadow: 0 10px 15px rgba(0, 0, 0, 0.1);
  }

  .header {
    margin-bottom: 2.5rem;
  }

  .title {
    font-size: 2.5rem;
    font-weight: 600;
    color: var(--gray-800, #1f2937);
    margin: 0 0 0.5rem 0;
    line-height: 1.2;
  }

  .subtitle {
    font-size: 1.125rem;
    color: var(--teal-600, #006666);
    margin: 0;
    font-weight: 500;
  }

  .login-content {
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .signin-button {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    background: var(--teal-600, #006666);
    border: none;
    color: white;
    border-radius: 0.375rem;
    padding: 1rem 2rem;
    font-size: 1.125rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .signin-button:hover {
    background: var(--teal-800, #004444);
    box-shadow: 0 4px 12px rgba(0, 102, 102, 0.25);
    transform: translateY(-1px);
  }

  .signin-button:active {
    transform: translateY(0);
  }

  .signin-button:focus {
    outline: 2px solid var(--teal-600, #006666);
    outline-offset: 2px;
  }

  .google-icon {
    flex-shrink: 0;
    background: white;
    border-radius: 0.25rem;
    padding: 0.25rem;
  }

  .success-message {
    color: var(--teal-600, #006666);
    font-size: 1.125rem;
    font-weight: 500;
    background: var(--teal-50, #d1efef);
    padding: 1rem;
    border-radius: 0.375rem;
    border: 1px solid var(--teal-100, #a3dfdf);
  }

  .success-message p {
    margin: 0.5rem 0;
  }

  .manual-redirect {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--teal-200, #7dd3fc);
  }

  .dashboard-link {
    display: inline-block;
    background: var(--teal-600, #006666);
    color: white;
    text-decoration: none;
    padding: 0.75rem 1.5rem;
    border-radius: 0.375rem;
    font-weight: 500;
    margin-top: 0.5rem;
    transition: background-color 0.2s ease;
  }

  .dashboard-link:hover {
    background: var(--teal-700, #004d4d);
  }

  .info-text {
    font-size: 0.875rem;
    color: var(--teal-600, #006666);
    line-height: 1.5;
    background: rgba(255, 255, 255, 0.7);
    padding: 1.5rem;
    border-radius: 0.375rem;
    border: 1px solid var(--teal-100, #a3dfdf);
  }

  .info-text p {
    margin: 0.25rem 0;
  }

  .dashboard-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
    display: block;
  }

  /* Responsive design for large displays */
  @media (min-width: 1200px) {
    .title {
      font-size: 3rem;
    }

    .login-card {
      padding: 4rem;
      max-width: 600px;
    }

    .signin-button {
      font-size: 1.25rem;
      padding: 1.25rem 2.5rem;
    }

    .dashboard-icon {
      font-size: 4rem;
    }
  }

  /* Ensure it looks good on very large displays */
  @media (min-width: 1920px) {
    .title {
      font-size: 3.5rem;
    }

    .login-card {
      padding: 5rem;
      max-width: 700px;
    }

    .signin-button {
      font-size: 1.375rem;
      padding: 1.5rem 3rem;
    }

    .dashboard-icon {
      font-size: 5rem;
    }
  }
</style>
