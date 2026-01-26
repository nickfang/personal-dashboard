import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
  // Assuming `locals.user` is populated by your auth hook (e.g., src/hooks.server.ts)
  console.log(
    `Login page server load: locals.user found: ${!!locals.user}, user sub: ${locals.user?.profile?.sub}, locals.isAuthenticated: ${locals.isAuthenticated}`
  );
  if (locals.user) {
    const redirectTo = url.searchParams.get('redirectTo');
    console.log(`Login page server load: User is authenticated. redirectTo param: '${redirectTo}'`);
    let safeRedirectPath = '/dashboard'; // Default redirect path

    if (redirectTo) {
      try {
        // Ensure the redirectTo path is relative or on the same origin
        const redirectToUrl = new URL(redirectTo, url.origin);
        if (redirectToUrl.origin === url.origin) {
          safeRedirectPath = redirectToUrl.pathname + redirectToUrl.search + redirectToUrl.hash;
        } else if (redirectTo.startsWith('/')) {
          // Allow relative paths
          safeRedirectPath = redirectTo;
        } else {
          console.warn(
            `Login page server load: redirectTo param '${redirectTo}' is not a valid same-origin URL or relative path. Defaulting to /dashboard.`
          );
        }
        // Otherwise, it defaults to '/dashboard' for security (prevents open redirect)
      } catch (e: any) {
        console.warn(
          `Login page server load: Error parsing redirectTo param '${redirectTo}'. Defaulting to /dashboard. Error: ${e.message}`
        );
      }
    }
    console.log(`Login page server load: Attempting to redirect to '${safeRedirectPath}'`);
    throw redirect(302, safeRedirectPath);
  }
  console.log('Login page server load: User not authenticated, rendering login page.');
  return {}; // User is not authenticated, render the login page
};
