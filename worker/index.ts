import { Container, getContainer } from "@cloudflare/containers";
import { env } from "cloudflare:workers";

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
  };

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

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    return getContainer(env.BACKEND).fetch(toContainerRequest(request));
  },
} satisfies ExportedHandler<Env>;
