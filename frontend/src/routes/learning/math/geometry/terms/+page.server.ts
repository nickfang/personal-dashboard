import { error } from '@sveltejs/kit';
import { terms } from '../../Info.js';

export const load = () => {
  return { terms };
};
