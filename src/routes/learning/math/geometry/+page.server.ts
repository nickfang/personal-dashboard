import { error } from '@sveltejs/kit';
import { Lines, Shapes, Volumes } from '../Info.js';

export const load = () => {
  return { Lines, Shapes, Volumes };
};
