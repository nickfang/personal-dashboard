import { error } from '@sveltejs/kit';
import { Terms } from '../../Info.js';

export const load = () => {
  return { Terms };
};
