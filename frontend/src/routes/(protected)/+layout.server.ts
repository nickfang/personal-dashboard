import { redirect } from '@sveltejs/kit';

export const load = async ({ url, locals }) => {
  console.log('[Protected Layout] Load function called with URL:', url.href);
  console.log('[Protected Layout] Locals:', locals);
  if (!locals.isAuthenticated) {
    throw redirect(302, `/login?redirectTo=${encodeURIComponent(url.href)}`);
  }
  return { isAuthenticated: locals.isAuthenticated, user: locals.user };
};
