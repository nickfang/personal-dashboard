import { writable } from 'svelte/store';
import { browser } from '$app/environment';

/**
 * Creates a store that persists to localStorage
 */
function createPersistedStore<T>(key: string, initial: T) {
  const stored = browser ? localStorage.getItem(key) : null;
  const store = writable<T>(stored ? JSON.parse(stored) : initial);

  if (browser) {
    store.subscribe((value) => localStorage.setItem(key, JSON.stringify(value)));
  }

  return store;
}

/**
 * Calendar view mode preference
 * - 'auto': Automatically choose based on container size (wide → week grid, narrow → list)
 * - 'list': Always show agenda/list format
 * - '3-day': Always show 3-day grid
 * - 'week': Always show 7-day grid
 */
export const calendarViewMode = createPersistedStore<'auto' | 'list' | '3-day' | 'week'>(
  'calendar-view-mode',
  'auto'
);
