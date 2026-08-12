/**
 * Headers a client can set on its own request. Nginx used to overwrite these;
 * now that it is gone the Worker must strip them, otherwise a forged value
 * would reach the container and be trusted. Verified 2026-08-12: no Go code
 * reads X-Forwarded-Proto, so dropping it is safe.
 */
const CLIENT_SPOOFABLE = [
  "X-Real-IP",
  "X-Forwarded-For",
  "X-Forwarded-Proto",
  "True-Client-IP",
];

/**
 * Returns a copy of the request carrying exactly one IP header:
 * CF-Connecting-IP, which Cloudflare's edge sets and a client cannot forge.
 * The Go router reads it via chimiddleware.ClientIPFromHeader("CF-Connecting-IP").
 */
export function normalizeClientIP(request: Request): Request {
  const realIP = request.headers.get("CF-Connecting-IP");
  const out = new Request(request);

  for (const h of CLIENT_SPOOFABLE) {
    out.headers.delete(h);
  }

  // The clone above already carries CF-Connecting-IP through; only delete it
  // when there was no trusted value, so a forged header cannot survive.
  if (!realIP) {
    out.headers.delete("CF-Connecting-IP");
  }

  return out;
}
