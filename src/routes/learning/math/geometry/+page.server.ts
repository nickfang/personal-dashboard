import { error } from '@sveltejs/kit';
import { Shapes } from '../Info.js';

export const load = () => {
  return { Shapes };
};
