import { Container, getContainer } from "@cloudflare/containers";
import { env } from "cloudflare:workers";
import { applySecurityHeaders } from "./headers";
import { normalizeClientIP } from "./client-ip";

export class Backend extends Container {
  defaultPort = 8080;
  sleepAfter = "1h";
  pingEndpoint = "health";

  envVars = {
    PORT: "8080",
    ENV: "production",
    DATABASE_URL: env.DATABASE_URL,
    JWT_SECRET: env.JWT_SECRET,
    // router.go:57 calls os.Exit(1) when this is missing and ENV=production.
    // The container dies before it listens, which surfaces only as a port
    // connection error from the Worker — never as the real reason.
    TOKEN_ENCRYPTION_KEY: env.TOKEN_ENCRYPTION_KEY,
    GOOGLE_CLIENT_ID: env.GOOGLE_CLIENT_ID,
    S3_ENDPOINT: env.S3_ENDPOINT,
    S3_BUCKET: env.S3_BUCKET,
    S3_ACCESS_KEY: env.S3_ACCESS_KEY,
    S3_SECRET_KEY: env.S3_SECRET_KEY,
    S3_PUBLIC_URL: env.S3_PUBLIC_URL,
    STRIPE_SECRET_KEY: env.STRIPE_SECRET_KEY,
    STRIPE_WEBHOOK_SECRET: env.STRIPE_WEBHOOK_SECRET,
    STRIPE_PUBLISHABLE_KEY: env.STRIPE_PUBLISHABLE_KEY,
    STRIPE_PRICE_PIXEL_SLOT: env.STRIPE_PRICE_PIXEL_SLOT,
    STRIPE_PRICE_REPLAY_SINGLE: env.STRIPE_PRICE_REPLAY_SINGLE,
    STRIPE_PRICE_REPLAY_MONTHLY: env.STRIPE_PRICE_REPLAY_MONTHLY,
    BASE_URL: env.BASE_URL,
    FRONTEND_URL: env.FRONTEND_URL,
    CORS_ALLOWED_ORIGINS: env.CORS_ALLOWED_ORIGINS,
  };

  override onError(error: unknown) {
    console.error("container error", error);
  }

  /**
   * The SDK gives a cold container 8s to become reachable and 20s for its port.
   * This image needs longer: cmd/server/main.go opens the pgx pool, pings Neon
   * in us-east-1, runs 27 migrations, recovers orphaned replay sessions and
   * re-encrypts tokens — all before it reaches ListenAndServe.
   *
   * Measured 2026-08-12: requests failed at 8.5s with "Failed to start
   * container: The container is not running", matching the SDK's 8s default
   * exactly.
   */
  override async fetch(request: Request): Promise<Response> {
    await this.startAndWaitForPorts({
      ports: [8080],
      cancellationOptions: {
        instanceGetTimeoutMS: 60_000,
        portReadyTimeoutMS: 180_000,
      },
    });
    return super.fetch(request);
  }
}

/**
 * Rewrites the request onto a container-local origin before proxying.
 *
 * Passing the original request through unchanged makes the runtime resolve its
 * public hostname, which loops the Worker back into itself and returns
 * Cloudflare error 1042 ("Worker tried to fetch from another Worker on the same
 * zone"). Only the origin changes — method, headers and body are preserved.
 */
function toContainerRequest(request: Request): Request {
  const url = new URL(request.url);
  return new Request(`http://container${url.pathname}${url.search}`, request);
}

/** Serves the stored nginx error page for a given status, falling back to plain text. */
async function errorPage(env: Env, status: 502 | 503 | 504): Promise<Response> {
  const res = await env.ASSETS.fetch(
    new Request(`https://assets.local/error-pages/${status}.html`),
  );
  if (res.ok) {
    return new Response(res.body, {
      status,
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  }
  return new Response(`Backend unavailable (${status})`, { status });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const pathname = new URL(request.url).pathname;

    try {
      // Both wrappers are load-bearing: normalizeClientIP stops header forgery,
      // toContainerRequest stops the runtime self-loop (Cloudflare error 1042).
      const response = await getContainer(env.BACKEND).fetch(
        toContainerRequest(normalizeClientIP(request)),
      );
      return applySecurityHeaders(response, pathname);
    } catch (error) {
      console.error("failed to reach backend container", error);
      return applySecurityHeaders(await errorPage(env, 502), pathname);
    }
  },
} satisfies ExportedHandler<Env>;
