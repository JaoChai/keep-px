const DASHBOARD_CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline' js.stripe.com connect.facebook.net",
  "style-src 'self' 'unsafe-inline' fonts.googleapis.com",
  "connect-src 'self' *.neon.tech js.stripe.com",
  "frame-src js.stripe.com",
  "img-src 'self' data: *.googleusercontent.com *.r2.dev",
  "font-src 'self' data: fonts.gstatic.com",
].join("; ");

const SALE_PAGE_CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline'",
  "style-src 'self' 'unsafe-inline' fonts.googleapis.com",
  "connect-src 'self'",
  "img-src * data:",
  "font-src 'self' data: fonts.gstatic.com",
].join("; ");

const HSTS = "max-age=31536000; includeSubDomains";
const REFERRER = "strict-origin-when-cross-origin";

/**
 * Sale pages are meant to be embedded on customer domains, so they
 * deliberately omit X-Frame-Options / Permissions-Policy / COOP — the same
 * split the old nginx.conf made between its `/p/` block and the server block.
 */
function isSalePage(pathname: string): boolean {
  return pathname.startsWith("/p/");
}

export function applySecurityHeaders(response: Response, pathname: string): Response {
  const res = new Response(response.body, response);

  res.headers.set("X-Content-Type-Options", "nosniff");
  res.headers.set("Referrer-Policy", REFERRER);
  res.headers.set("Strict-Transport-Security", HSTS);

  if (isSalePage(pathname)) {
    res.headers.set("Content-Security-Policy", SALE_PAGE_CSP);
    return res;
  }

  res.headers.set("X-Frame-Options", "DENY");
  res.headers.set("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
  res.headers.set("Cross-Origin-Opener-Policy", "same-origin");
  res.headers.set("Content-Security-Policy", DASHBOARD_CSP);
  return res;
}
