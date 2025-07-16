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
  user.set(null);
  isAuthenticated.set(false);
  // Redirect to login page when token expires
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
});

userManager.events.addSilentRenewError((error) => {
  console.error('[authService] Silent renew error:', error);
  user.set(null);
  isAuthenticated.set(false);
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
    // Clear any stored tokens
    await userManager.removeUser();
  } catch (error) {
    console.error('Error removing user:', error);
  }
  
  // Direct redirect to login without using Cognito hosted UI
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
