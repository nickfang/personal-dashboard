import { error } from '@sveltejs/kit';
import { shapes } from '../Info.js';
import type { ShapesType } from '../Info.js';

interface ShapeOptionsType {
  value: string;
  label: string;
}

export const load = (): { shapes: ShapesType; shapeOptions: ShapeOptionsType[] } => {
  const shapeOptions: ShapeOptionsType[] = Object.entries(shapes).map(([key, value]) => ({
    value: key,
    label: value.name,
  }));
  return { shapes, shapeOptions };
};
