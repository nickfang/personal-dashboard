import { redirect } from '@sveltejs/kit';
import { UserManager } from 'oidc-client-ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
  const userManager = new UserManager({
    authority: 'https://cognito-idp.us-east-1.amazonaws.com/us-east-1_9v6V05dbM',
    client_id: '42p433f7ubb3ogda4j53mb4pl4',
    redirect_uri: 'https://ianbeefang.com/',
    response_type: 'code',
    scope: 'phone openid email',
  });
  throw redirect(307, '/dashboard');
};
