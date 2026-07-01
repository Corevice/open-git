import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Public routes that must be reachable without authentication. The `/:owner`
// and `/:owner/:repo/:path*` matchers below also match single-/multi-segment
// public routes such as `/login` and `/login/oauth/authorize`, so without this
// allowlist an unauthenticated visit to `/login` would redirect to
// `/login?callbackUrl=/login` — an infinite loop that makes the app unusable
// while logged out.
const PUBLIC_PREFIXES = ['/login', '/signup', '/about', '/licenses', '/docs'];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_PREFIXES.some((prefix) => pathname === prefix || pathname.startsWith(prefix + '/'));
}

export function middleware(request: NextRequest) {
  if (isPublicPath(request.nextUrl.pathname)) {
    return NextResponse.next();
  }

  const token = request.cookies.get('authToken')?.value;

  if (!token) {
    const loginUrl = new URL('/login', request.url);
    const callbackPath = request.nextUrl.pathname + request.nextUrl.search;
    loginUrl.searchParams.set('callbackUrl', callbackPath);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/dashboard/:path*',
    '/dashboard',
    '/new/:path*',
    '/new',
    '/:owner',
    '/:owner/people',
    '/:owner/settings',
    '/settings/:path*',
    '/:owner/:repo/:path*',
    '/:owner/:repo',
  ],
};
