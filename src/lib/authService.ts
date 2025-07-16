import { UserManager, User } from 'oidc-client-ts';
import { authSettings } from './authConfig';
import { writable, type Writable } from 'svelte/store';

export const user: Writable<User | null> = writable(null);
export const isAuthenticated: Writable<boolean> = writable(false);

const userManager = new UserManager(authSettings);

userManager.events.addUserLoaded((loadedUser) => {
  console.log('[authService] User loaded:', loadedUser?.profile?.sub);
  user.set(loadedUser);
  isAuthenticated.set(true);
});

userManager.events.addUserUnloaded(() => {
  console.log('[authService] User unloaded');
  user.set(null);
  isAuthenticated.set(false);
});

userManager.events.addAccessTokenExpired(() => {
  console.log('[authService] Access token expired');
  // Don't immediately redirect - let silent renewal handle it first
  // Only redirect if we're not in the middle of a page load
  setTimeout(() => {
    userManager.getUser().then((currentUser) => {
      if (!currentUser) {
        console.log('[authService] No user after token expiry, redirecting to login');
        user.set(null);
        isAuthenticated.set(false);
        if (typeof window !== 'undefined') {
          window.location.href = '/login';
        }
      }
    });
  }, 2000); // Give silent renewal 2 seconds to work
});

userManager.events.addSilentRenewError((error) => {
  console.error('[authService] Silent renew error:', error);
  user.set(null);
  isAuthenticated.set(false);
});

userManager.events.addUserSignedIn(() => {
  console.log('[authService] User signed in');
  userManager.getUser().then((loadedUser) => {
    if (loadedUser) {
      user.set(loadedUser);
      isAuthenticated.set(true);
    }
  });
});

export const startSignIn = async (): Promise<void> => {
  console.log('Starting SignIn');
  await userManager.signinRedirect();
};

export const handleCallback = async (): Promise<User | null> => {
  try {
    const loadedUser = await userManager.signinRedirectCallback();
    console.log(
      '[authService handleCallback] Loaded User from signinRedirectCallback:',
      loadedUser
    );
    console.log('[authService handleCallback] User state from loadedUser:', loadedUser?.state);
    user.set(loadedUser);
    isAuthenticated.set(true);

    if (loadedUser) {
      try {
        const response = await fetch('/api/auth/session', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(loadedUser),
        });
        if (!response.ok) {
          console.error('Failed to create server session:', await response.text());
        }
      } catch (error) {
        console.error('Error calling session creation endpoint:', error);
      }
    }
    return loadedUser;
  } catch (error) {
    console.error('Authentication callback error: ', error);
    user.set(null);
    isAuthenticated.set(false);
    return null;
  }
};

export const startSignOut = async (): Promise<void> => {
  try {
    // Clear the server session first
    await fetch('/api/auth/session', { method: 'DELETE' });
  } catch (error) {
    console.error('Error calling session deletion endpoint:', error);
  }
  
  // Clear client-side state immediately
  user.set(null);
  isAuthenticated.set(false);
  
  try {
    // Use OIDC client's built-in sign-out redirect for proper global logout
    console.log('Initiating global sign-out redirect...');
    await userManager.signoutRedirect();
  } catch (error) {
    console.error('Error during sign-out redirect:', error);
    
    // Fallback: Clear local tokens and redirect to manual logout flow
    try {
      await userManager.removeUser();
    } catch (removeError) {
      console.error('Error removing user tokens:', removeError);
    }
    
    // Multi-step logout: First Cognito, then Google
    if (typeof window !== 'undefined') {
      const logoutUrl = authSettings.post_logout_redirect_uri || window.location.origin + '/login';
      const cognitoDomain = `https://${import.meta.env.VITE_COGNITO_DOMAIN}.auth.${import.meta.env.VITE_AWS_REGION}.amazoncognito.com`;
      
      // Add Google logout parameter to ensure Google session is cleared too
      const signOutUrl = `${cognitoDomain}/logout?client_id=${import.meta.env.VITE_COGNITO_CLIENT_ID}&logout_uri=${encodeURIComponent(logoutUrl)}&federated_signout=true`;
      
      console.log('Fallback: Redirecting to Cognito logout with federated signout:', signOutUrl);
      window.location.href = signOutUrl;
    }
  }
};

export const startSignOutComplete = async (): Promise<void> => {
  console.log('Starting complete sign-out (Cognito + Google)...');
  
  try {
    // Clear the server session first
    await fetch('/api/auth/session', { method: 'DELETE' });
  } catch (error) {
    console.error('Error calling session deletion endpoint:', error);
  }
  
  // Clear client-side state immediately
  user.set(null);
  isAuthenticated.set(false);
  
  // Clear all local tokens
  try {
    await userManager.removeUser();
  } catch (error) {
    console.error('Error removing user tokens:', error);
  }
  
  if (typeof window !== 'undefined') {
    // Clear all local storage and session storage
    localStorage.clear();
    sessionStorage.clear();
    
    // Multi-step logout process
    const logoutUrl = authSettings.post_logout_redirect_uri || window.location.origin + '/login';
    const cognitoDomain = `https://${import.meta.env.VITE_COGNITO_DOMAIN}.auth.${import.meta.env.VITE_AWS_REGION}.amazoncognito.com`;
    
    // Sign out from Cognito with federated signout
    const signOutUrl = `${cognitoDomain}/logout?client_id=${import.meta.env.VITE_COGNITO_CLIENT_ID}&logout_uri=${encodeURIComponent(logoutUrl)}&federated_signout=true`;
    
    console.log('Complete sign-out: Redirecting to:', signOutUrl);
    window.location.href = signOutUrl;
  }
};

export const startSignOutLocal = async (): Promise<void> => {
  try {
    // Clear the server session first
    await fetch('/api/auth/session', { method: 'DELETE' });
  } catch (error) {
    console.error('Error calling session deletion endpoint:', error);
  }
  
  // Clear client-side state immediately
  user.set(null);
  isAuthenticated.set(false);
  
  try {
    // Clear any stored tokens
    await userManager.removeUser();
  } catch (error) {
    console.error('Error removing user:', error);
  }
  
  // Direct redirect to login without using Cognito hosted UI (local sign-out only)
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
};

export const forceSignOut = async (): Promise<void> => {
  try {
    // Clear the server session
    await fetch('/api/auth/session', { method: 'DELETE' });
  } catch (error) {
    console.error('Error calling session deletion endpoint:', error);
  }
  
  // Clear client-side state
  user.set(null);
  isAuthenticated.set(false);
  
  // Clear any stored tokens
  try {
    await userManager.removeUser();
  } catch (error) {
    console.error('Error removing user:', error);
  }
  
  // Direct redirect to login without going through Cognito
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
};

export const getAccessToken = async (): Promise<string | null> => {
  const currentUser = await userManager.getUser();
  return currentUser?.access_token || null;
};

export const checkAuthStatus = async (): Promise<void> => {
  try {
    const currentUser = await userManager.getUser();
    if (currentUser) {
      user.set(currentUser);
      isAuthenticated.set(true);
    } else {
      isAuthenticated.set(false);
      user.set(null);
    }
  } catch (error) {
    console.error('Error checking auth status:', error);
    isAuthenticated.set(false);
    user.set(null);
  }
};

checkAuthStatus();
