import { error } from '@sveltejs/kit';
import { Shapes } from '../Info.js';
import type { ShapesType } from '../Info.js';

export const load = (): { Shapes: ShapesType } => {
  return { Shapes };
};
