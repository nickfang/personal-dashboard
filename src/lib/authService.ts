import { UserManager, User } from 'oidc-client-ts';
import { authSettings } from './authConfig';
import { writable, type Writable } from 'svelte/store';

export const user: Writable<User | null> = writable(null);
export const isAuthenticated: Writable<boolean> = writable(false);

const userManager = new UserManager(authSettings);

userManager.events.addUserLoaded((loadedUser) => {
  user.set(loadedUser);
  isAuthenticated.set(true);
});

userManager.events.addUserUnloaded(() => {
  user.set(null);
  isAuthenticated.set(false);
});

export const startSignIn = async (): Promise<void> => {
  await userManager.signinRedirect();
};

export const handleCallback = async (): Promise<User | null> => {
  try {
    const loadedUser = await userManager.signinRedirectCallback();
    user.set(loadedUser);
    isAuthenticated.set(true);
    return loadedUser;
  } catch (error) {
    console.error('Authentication callback error: ', error);
    user.set(null);
    isAuthenticated.set(false);
    return null;
  }
};

export const startSignOut = (): Promise<void> => {
  return userManager.signoutRedirect();
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
