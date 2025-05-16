import { isAuthenticated } from '$lib/authService';
import { redirect } from '@sveltejs/kit';

export const load = async ({ url }) => {
  if (!isAuthenticated) {
    throw redirect(302, `/login?redirectTo=${encodeURIComponent(url.href)}`);
  }
  return { isAuthenticated };
};
