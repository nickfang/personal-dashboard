// /Users/nfang/workspace/personal-dashboard/src/hooks.server.ts
import type { Handle } from '@sveltejs/kit';
import type { User } from 'oidc-client-ts';
import jwt from 'jsonwebtoken';
import { env } from '$env/dynamic/private';

const JWT_SECRET = env.JWT_SECRET;
const SESSION_COOKIE_NAME = '__session';

if (!JWT_SECRET && process.env.NODE_ENV !== 'test') {
  // Avoid error during tests if not set, but log for other environments
  console.error(
    'FATAL: JWT_SECRET is not defined in environment variables. Session verification will not work.'
  );
}

async function verifySessionAndGetUser(sessionCookieValue: string): Promise<User | null> {
  if (!JWT_SECRET) {
    console.error('hooks.server.ts: JWT_SECRET not available for session verification.');
    return null;
  }
  console.log('[HOOKS] verifySessionAndGetUser: Attempting to verify session cookie.');
  try {
    const decoded = jwt.verify(sessionCookieValue, JWT_SECRET) as {
      user: User;
      iat: number;
      exp: number;
    };
    console.log(
      '[HOOKS] verifySessionAndGetUser: JWT verification successful. Decoded user sub:',
      decoded.user?.profile?.sub
    );
    return decoded.user;
  } catch (error: any) {
    console.warn(
      '[HOOKS] verifySessionAndGetUser: JWT Session verification FAILED. Error:',
      error.message,
      error.name ? `(Type: ${error.name})` : ''
    );
    return null;
  }
}

export const handle: Handle = async ({ event, resolve }) => {
  console.log(`[HOOKS] handle: Processing request for ${event.url.pathname}`);
  const sessionCookie = event.cookies.get(SESSION_COOKIE_NAME);

  event.locals.user = null;
  event.locals.isAuthenticated = false;

  if (!sessionCookie) {
    console.log(`[HOOKS] handle: Session cookie ('${SESSION_COOKIE_NAME}') NOT FOUND.`);
  } else {
    console.log(
      `[HOOKS] handle: Session cookie ('${SESSION_COOKIE_NAME}') FOUND. Length: ${sessionCookie.length}. Attempting to verify.`
    );
  }

  if (sessionCookie) {
    try {
      const user = await verifySessionAndGetUser(sessionCookie);
      if (user) {
        event.locals.user = user;
        event.locals.isAuthenticated = true;
        console.log(
          '[HOOKS] handle: User successfully authenticated from session. User sub:',
          event.locals.user?.profile?.sub
        );
      }
    } catch (error) {
      console.error('[HOOKS] handle: Unexpected error during session verification process:', error);
    }
  }

  return resolve(event);
};
