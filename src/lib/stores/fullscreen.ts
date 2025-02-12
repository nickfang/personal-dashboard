import { writable } from 'svelte/store';

type FullscreenState = false | 'weather' | 'calendar' | 'satword';

export const fullscreenStore = writable<FullscreenState>(false); 