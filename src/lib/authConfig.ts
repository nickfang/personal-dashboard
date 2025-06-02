import type { UserManagerSettings } from 'oidc-client-ts';

export const authSettings: UserManagerSettings = {
  authority: `https://cognito-idp.${import.meta.env.VITE_AWS_REGION}.amazonaws.com/${import.meta.env.VITE_COGNITO_USER_POOL_ID}`,
  client_id: import.meta.env.VITE_COGNITO_CLIENT_ID,
  redirect_uri:
    import.meta.env.MODE === 'production'
      ? import.meta.env.VITE_COGNITO_CALLBACK_PROD
      : import.meta.env.VITE_COGNITO_CALLBACK_DEV,
  post_logout_redirect_uri:
    import.meta.env.MODE === 'production'
      ? import.meta.env.VITE_COGNITO_LOGOUT_PROD
      : import.meta.env.VITE_COGNITO_LOGOUT_DEV,
  response_type: 'code',
  scope: 'email openid phone',
  automaticSilentRenew: false,
};
