import { NextResponse, type NextRequest } from "next/server";

/**
 * Route protection.
 *
 * Next.js runs this before rendering any matching page. We use it to bounce
 * signed-out visitors away from the dashboard, and signed-in ones away from
 * the login form.
 *
 * ⚠️ THIS IS NOT SECURITY. ⚠️
 *
 * All it checks is whether a cookie EXISTS. It does not verify the signature,
 * because the secret lives on the Go backend and must never be shipped to the
 * frontend. Anyone can create a fake `am_session` cookie in their browser and
 * get past this.
 *
 * That is fine, because it is not what protects the data. Every private API
 * endpoint verifies the token properly on the server. If someone forges a
 * cookie, they reach a dashboard whose every request comes back 401.
 *
 * So this is a UX convenience - avoid a flash of empty dashboard before the
 * redirect - not a boundary. Treating a frontend check as security is one of
 * the most common and most serious mistakes in web development.
 */

const SESSION_COOKIE = "am_session";

/** Pages that require a session. */
const PROTECTED_PREFIXES = ["/dashboard", "/monitors", "/projects", "/settings"];

/** Pages a signed-in user has no reason to see. */
const AUTH_PAGES = ["/login", "/register"];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has(SESSION_COOKIE);

  const isProtected = PROTECTED_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );

  if (isProtected && !hasSession) {
    const url = request.nextUrl.clone();
    url.pathname = "/login";
    // Remember where they were heading, so we can send them back after they
    // sign in rather than dumping everyone on the dashboard.
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  if (AUTH_PAGES.includes(pathname) && hasSession) {
    const url = request.nextUrl.clone();
    url.pathname = "/dashboard";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

/**
 * Which requests this middleware runs for.
 *
 * Excluding static files and images matters: without it this would run for
 * every stylesheet, script and icon, adding pointless work to each one.
 */
export const config = {
  matcher: [
    "/dashboard/:path*",
    "/monitors/:path*",
    "/projects/:path*",
    "/settings/:path*",
    "/login",
    "/register",
  ],
};
