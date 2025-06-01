import { isAuthenticated } from '$lib/authService';
import { redirect } from '@sveltejs/kit';
import { get } from 'svelte/store';


export const load = async ({ url }) => {
  const isUserAuthenticated = get(isAuthenticated);
  console.log("layout.server::isAuthenticated", isUserAuthenticated)
  // if (!isUserAuthenticated) {
  //   throw redirect(302, `/login?redirectTo=${encodeURIComponent(url.href)}`);
  // }
  return { isAuthenticated:isUserAuthenticated };
};
