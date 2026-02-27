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

    // Check edge cache
    const cacheKey = new Request(url.toString(), request)
    const cache = caches.default
    const cachedResponse = await cache.match(cacheKey)
    if (cachedResponse) {
      return cachedResponse
    }

    // Proxy to backend using slug-based URL
    const backendUrl = `${env.BACKEND_ORIGIN}/p/${mapping.slug}`
    const proxyRequest = new Request(backendUrl, {
      method: request.method,
      headers: new Headers(request.headers),
    })

    // Add forwarding headers
    proxyRequest.headers.set("X-Forwarded-Host", hostname)
    proxyRequest.headers.set("X-Forwarded-Proto", url.protocol.replace(":", ""))
    proxyRequest.headers.set("X-Page-ID", mapping.sale_page_id)
    proxyRequest.headers.set("X-Customer-ID", mapping.customer_id)

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
