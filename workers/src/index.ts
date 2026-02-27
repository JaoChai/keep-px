interface Env {
  DOMAIN_MAP: KVNamespace
  BACKEND_ORIGIN: string
  PLATFORM_DOMAINS: string
}

interface DomainMapping {
  slug: string
  sale_page_id: string
  customer_id: string
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url)
    const hostname = url.hostname

    // Pass through platform domains (pixlinks.xyz, app.pixlinks.xyz, etc.)
    const platformDomains = env.PLATFORM_DOMAINS.split(",").map(d => d.trim())
    if (platformDomains.includes(hostname)) {
      return fetch(request)
    }

    // Lookup custom domain in KV
    const mappingRaw = await env.DOMAIN_MAP.get(hostname)
    if (!mappingRaw) {
      return new Response("Domain not configured", { status: 404 })
    }

    let mapping: DomainMapping
    try {
      mapping = JSON.parse(mappingRaw)
    } catch {
      return new Response("Invalid domain configuration", { status: 500 })
    }

    // Validate slug format to prevent SSRF via malicious KV data
    const slugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
    if (!slugPattern.test(mapping.slug)) {
      return new Response("Invalid page configuration", { status: 500 })
    }

    // Check edge cache (include hostname explicitly to prevent cross-domain cache poisoning)
    const cacheKey = new Request(`https://${hostname}${url.pathname}${url.search}`, {
      method: "GET",
    })
    const cache = caches.default
    const cachedResponse = await cache.match(cacheKey)
    if (cachedResponse) {
      return cachedResponse
    }

    // Proxy to backend using slug-based URL (encodeURIComponent as defense-in-depth)
    const backendUrl = `${env.BACKEND_ORIGIN}/p/${encodeURIComponent(mapping.slug)}`
    const proxyRequest = new Request(backendUrl, {
      method: request.method,
      headers: new Headers(request.headers),
    })

    // Add standard proxy forwarding headers only
    proxyRequest.headers.set("X-Forwarded-Host", hostname)
    proxyRequest.headers.set("X-Forwarded-Proto", url.protocol.replace(":", ""))

    try {
      const response = await fetch(proxyRequest)

      // Only cache successful HTML responses
      if (response.ok && response.headers.get("content-type")?.includes("text/html")) {
        const responseToCache = new Response(response.body, response)
        responseToCache.headers.set("Cache-Control", "public, max-age=300") // 5 minutes
        // Store in edge cache (non-blocking)
        cache.put(cacheKey, responseToCache.clone())
        return responseToCache
      }

      return response
    } catch {
      return new Response("Backend unavailable", { status: 502 })
    }
  },
} satisfies ExportedHandler<Env>
