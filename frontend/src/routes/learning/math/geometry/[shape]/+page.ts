import { shapes } from '$lib/data/geometry.js';

export function load({ params }) {
  const shape = params.shape;
  return { shape };
}
