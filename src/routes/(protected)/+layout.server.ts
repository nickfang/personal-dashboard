import { redirect } from '@sveltejs/kit';
import { get } from 'svelte/store';

export const load = async ({ url, locals }) => {
  if (!locals.isAuthenticated) {
    throw redirect(302, `/login?redirectTo=${encodeURIComponent(url.href)}`);
  }
  return { isAuthenticated: locals.isAuthenticated, user: locals.user };
};
