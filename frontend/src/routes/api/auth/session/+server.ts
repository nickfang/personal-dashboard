import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import jwt from 'jsonwebtoken';
import { env } from '$env/dynamic/private'; // For JWT_SECRET
import type { User } from 'oidc-client-ts';

const JWT_SECRET = env.JWT_SECRET;
const SESSION_COOKIE_NAME = '__session';
const COOKIE_MAX_AGE_SECONDS = 7 * 24 * 60 * 60; // 7 days

if (!JWT_SECRET) {
  // This log runs at module load time
  console.error(
    'FATAL: JWT_SECRET is not defined in environment variables. Session management will not work.'
  );
  // In a real app, you might want to prevent startup or throw a more visible error.
}

export const POST: RequestHandler = async ({ request, cookies }) => {
  console.log('[API /api/auth/session POST] Received request.');
  if (!JWT_SECRET) {
    console.error('[API /api/auth/session POST] JWT_SECRET is missing at request time.');
    throw error(500, 'Server misconfiguration: JWT secret not set.');
  }
  try {
    const oidcUser = (await request.json()) as User;
    console.log('[API /api/auth/session POST] OIDC User Sub:', oidcUser.profile?.sub);
    // Create a payload for the JWT. Include essential, non-sensitive user info.
    // The entire OIDC User object can be large; select what's needed for `event.locals.user`.
    // Ensure this structure is compatible with what `verifySessionAndGetUser` expects.
    const payload = {
      sub: oidcUser.profile?.sub,
      email: oidcUser.profile?.email,
      name: oidcUser.profile?.name,
      // You might want to include specific claims or roles if available and needed server-side
      // access_token: oidcUser.access_token, // Be cautious about storing access tokens in cookies if not strictly needed
      // id_token: oidcUser.id_token, // Similarly, id_token
      // expires_at: oidcUser.expires_at,
      profile: oidcUser.profile, // Storing the whole profile for convenience, ensure it's not too large
    };

    const token = jwt.sign({ user: payload }, JWT_SECRET, {
      expiresIn: `${COOKIE_MAX_AGE_SECONDS}s`,
    });

    cookies.set(SESSION_COOKIE_NAME, token, {
      httpOnly: true,
      path: '/',
      secure: process.env.NODE_ENV === 'production', // Use secure cookies in production
      sameSite: 'lax',
      maxAge: COOKIE_MAX_AGE_SECONDS,
    });

    console.log(
      `[API /api/auth/session POST] Session cookie '${SESSION_COOKIE_NAME}' set for user:`,
      payload.sub
    );
    return json({ success: true, message: 'Session created' }, { status: 200 });
  } catch (e) {
    console.error('[API /api/auth/session POST] Error creating session:', e);
    throw error(500, 'Failed to create session');
  }
};

export const DELETE: RequestHandler = async ({ cookies }) => {
  console.log(
    `[API /api/auth/session DELETE] Received request to delete session cookie '${SESSION_COOKIE_NAME}'.`
  );
  cookies.delete(SESSION_COOKIE_NAME, { path: '/' });
  console.log(`[API /api/auth/session DELETE] Session cookie '${SESSION_COOKIE_NAME}' deleted.`);
  return json({ success: true, message: 'Session deleted' }, { status: 200 });
};
