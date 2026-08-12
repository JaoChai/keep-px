/**
 * Serves customer-owned domains that point at a Keep-PX sale page.
 *
 * A customer CNAMEs their own domain here; DOMAIN_MAP holds
 * `hostname -> {"slug": "..."}` and this Worker rewrites the request to the
 * sale page that slug belongs to.
 *
 * Recovered into version control on 2026-08-12 — until then the only copy was
 * the deployed script on Cloudflare, with no source in any repository.
 */

interface DomainMapping {
  slug: string;
}

interface Env {
  DOMAIN_MAP: KVNamespace;
  PLATFORM_DOMAINS: string;
  /**
   * Service binding to the main `keep-px` Worker.
   *
   * This used to be a plain `fetch()` to BACKEND_ORIGIN
   * (https://api.pixlinks.xyz). That worked while the backend lived on
   * Railway, but after the Cloudflare migration that hostname resolves to a
   * Worker in this same zone — and a Worker calling another Worker by URL
   * returns Cloudflare error 1042. A service binding routes the request
   * directly, with no DNS round trip and no self-loop.
   */
  SALE_PAGES: Fetcher;
}

const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const CACHE_TTL_SECONDS = 300;

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const hostname = url.hostname;

    const platformDomains = env.PLATFORM_DOMAINS.split(",").map((d) => d.trim());
    if (platformDomains.includes(hostname)) {
      return env.SALE_PAGES.fetch(request);
    }

    const mappingRaw = await env.DOMAIN_MAP.get(hostname);
    if (!mappingRaw) {
      return new Response("Domain not configured", { status: 404 });
    }

    let mapping: DomainMapping;
    try {
      mapping = JSON.parse(mappingRaw);
    } catch {
      return new Response("Invalid domain configuration", { status: 500 });
    }

    if (!SLUG_PATTERN.test(mapping.slug)) {
      return new Response("Invalid page configuration", { status: 500 });
    }

    const cacheKey = new Request(`https://${hostname}${url.pathname}${url.search}`, {
      method: "GET",
    });
    const cache = caches.default;
    const cached = await cache.match(cacheKey);
    if (cached) {
      return cached;
    }

    const proxyRequest = new Request(
      `https://sale-pages.internal/p/${encodeURIComponent(mapping.slug)}`,
      { method: request.method, headers: new Headers(request.headers) },
    );
    proxyRequest.headers.set("X-Forwarded-Host", hostname);

    try {
      const response = await env.SALE_PAGES.fetch(proxyRequest);
      if (response.ok && response.headers.get("content-type")?.includes("text/html")) {
        const cacheable = new Response(response.body, response);
        cacheable.headers.set("Cache-Control", `public, max-age=${CACHE_TTL_SECONDS}`);
        await cache.put(cacheKey, cacheable.clone());
        return cacheable;
      }
      return response;
    } catch {
      return new Response("Backend unavailable", { status: 502 });
    }
  },
} satisfies ExportedHandler<Env>;
