import type { UserManagerSettings } from 'oidc-client-ts';

export const authSettings: UserManagerSettings = {
  authority: `https://cognito-idp.${import.meta.env.VITE_AWS_REGION}.amazonaws.com/${import.meta.env.VITE_COGNITO_USER_POOL_ID}`,
  client_id: import.meta.env.VITE_COGNITO_CLIENT_ID,
  redirect_uri: `http://ianbeefang.com/callback`,
  post_logout_redirect_uri: `http://ianbeefang.com`,
  response_type: 'code',
  scope: 'openid profile email',
  automaticSilentRenew: false,
};
