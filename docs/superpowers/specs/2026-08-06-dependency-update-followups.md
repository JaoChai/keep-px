# Follow-ups Found During the 2026-08-06 Dependency Update

Two issues surfaced while upgrading dependencies. Neither was caused by that work; both were found because the upgrade made tooling look at code nobody had looked at in a while. Each needs its own PR.

---

## 1. Client IP Spoofing — Rate Limiter Bypass

**Date**: 2026-08-06
**Status**: Verified against production 2026-08-06; implementation in progress
**Type**: Security fix — changes request-IP behaviour behind the proxy

> **Revised 2026-08-06 after probing production.** The original draft of this
> section named `X-Forwarded-For` as the spoofable header and treated the
> deployment topology as an open question. Both were wrong. Live probes against
> `api.pixlinks.xyz` and `pixlinks.xyz` (results in *Measured behaviour* below)
> show the forgeable header is `True-Client-IP`, and they surfaced a second
> defect the draft missed entirely. The conclusion — the per-IP limiter can be
> bypassed — stands.

## The Problem

`internal/router/router.go:30` registers `chimiddleware.RealIP`. That middleware rewrites `r.RemoteAddr` from the first header it finds, in this order: `True-Client-IP`, then `X-Real-IP`, then the leftmost `X-Forwarded-For` entry — regardless of whether the infrastructure actually set them.

Neither Railway's edge proxy nor `frontend/nginx.conf` sets or strips `True-Client-IP`, so a client-supplied value passes through untouched and wins the priority order. `X-Forwarded-For` is never even consulted in this deployment, because `X-Real-IP` is always present and outranks it.

Three consumers read the resulting value:

| Site | Use | Impact when spoofed |
|---|---|---|
| `internal/middleware/ratelimit.go:52` | per-IP rate limiting | **Bypass.** Rotate a fresh forged IP per request and the limiter never trips. |
| `internal/handler/event_handler.go:250` | `client_ip_address` sent to Facebook CAPI | Garbage IPs degrade EMQ score; attacker can attribute events to arbitrary addresses. `extractClientIP` reads `True-Client-IP` directly, so it is spoofable independently of `RealIP`. |
| `internal/middleware/logger.go:32` | request logging | Logs and any incident forensics built on them are unreliable. |

The rate-limit bypass is the load-bearing one. `/api/v1/events/ingest` is protected by both a global limiter and a per-API-key limiter (`RateLimitByAPIKey`, added in #210); the per-IP layer is the one that fails open here.

chi v5.3.1 marks `RealIP` deprecated for exactly this reason and ships replacements. This was discovered on 2026-08-06 when a chi bump surfaced the deprecation during a dependency update — **the vulnerability itself predates that bump and is live in production today.**

## The Second Problem — every proxied request shares one rate-limit bucket

`frontend/nginx.conf:49` and `:69` set:

```nginx
proxy_set_header X-Real-IP $remote_addr;
```

At that point `$remote_addr` is Railway's edge proxy, not the visitor — Nginx never sees the public internet directly. So Nginx **discards** the correct value Railway's edge already placed in `X-Real-IP` and replaces it with a fixed internal address (`100.64.0.3`, measured).

Every request arriving through `pixlinks.xyz` therefore keys the per-IP limiter on that one address. That includes the tracking SDK: `internal/templates/sale_pages/tracking.html:104` posts to `{{.BaseURL}}/api/v1/events/ingest`, and `BASE_URL` defaults to `https://pixlinks.xyz`. **The per-IP limiter is effectively a single global bucket for all sale-page and SDK ingest traffic** — one noisy visitor can 429 everyone else.

This is not caused by the fix above and is not fixed by it: swapping `RealIP` for `ClientIPFromHeader("X-Real-IP")` still reads the value Nginx overwrote. Nginx has to stop overwriting it.

## Measured behaviour (production, 2026-08-06)

Probes sent with `curl`; the resolved IP read back from the backend's own request log on Railway. `49.228.186.245` is the real client address.

| # | Path | Header sent | Backend resolved |
|---|---|---|---|
| A | direct to `api.pixlinks.xyz` | `X-Real-IP: 203.0.113.9` | `49.228.186.245` — edge overwrote it ✅ |
| B | direct to `api.pixlinks.xyz` | `True-Client-IP: 198.51.100.7` | `198.51.100.7` — **spoofed** ❌ |
| C | direct to `api.pixlinks.xyz` | `X-Forwarded-For: 192.0.2.44` | `49.228.186.245` ✅ |
| D | direct to `api.pixlinks.xyz` | none | `49.228.186.245` ✅ |
| E | via `pixlinks.xyz` → Nginx | none | `100.64.0.3` — **shared bucket** ⚠️ |
| F | via `pixlinks.xyz` → Nginx | `True-Client-IP: 203.0.113.55` | `203.0.113.55` — **spoofed** ❌ |
| G | via `pixlinks.xyz` → Nginx | `X-Forwarded-For: 192.0.2.99` | `100.64.0.3` ⚠️ |

Two facts follow, and they decide the design:

1. The backend is publicly reachable at `api.pixlinks.xyz` (custom domain on the `pixlinks-api` Railway service) — Nginx is not the only way in. Any fix must hold on both paths.
2. Railway's edge proxy **overwrites** a client-supplied `X-Real-IP` with the real peer address (probe A). That makes `X-Real-IP` trustworthy at the backend — provided nothing downstream clobbers it, which is exactly what Nginx currently does.

## Goals

- Derive the client IP from a source the client cannot forge.
- Keep per-IP rate limiting, CAPI `client_ip_address`, and request logging working behind Nginx on Railway.
- No change to any API contract.

- Restore genuine per-IP rate limiting for traffic arriving through Nginx.

## Non-Goals

- No change to the per-API-key rate limiter — it is unaffected.
- No new rate-limiting strategy, tiers, or thresholds.
- No change to how Nginx is *deployed*; the one Nginx change is two header lines in `nginx.conf`.

## Design

Two changes, both required. Either alone leaves a real defect standing.

**Go — stop trusting forgeable headers.** Replace `RealIP` with chi's `ClientIPFromHeader("X-Real-IP")`. Probe A proves Railway's edge overwrites that header on both ingress paths, so it is the one value a client cannot control. The replacement middlewares do **not** mutate `r.RemoteAddr`; the value is read with `middleware.GetClientIP(r.Context())`. Every consumer must therefore change too:

| File | Change |
|---|---|
| `internal/router/router.go:30` | `chimiddleware.RealIP` → `chimiddleware.ClientIPFromHeader("X-Real-IP")` |
| `internal/middleware/ratelimit.go:52` | read `GetClientIP(r.Context())`, fall back to `r.RemoteAddr` when empty |
| `internal/handler/event_handler.go:250` | drop the `True-Client-IP` / `CF-Connecting-IP` header reads; use `GetClientIP`, fall back to `r.RemoteAddr` |
| `internal/middleware/logger.go:32` | log the resolved client IP alongside `RemoteAddr` |

This also unblocks the `go-chi/chi/v5` upgrade to v5.3.1, which was excluded from the 2026-08-06 dependency update because of the deprecation. `ClientIPFromHeader` does not exist in the currently pinned v5.2.5, so the bump is a prerequisite, not a side effect.

**Nginx — stop clobbering the trustworthy value.** In both proxy blocks:

```nginx
proxy_set_header X-Real-IP $http_x_real_ip;   # was: $remote_addr
```

This forwards the address Railway's edge already resolved instead of Nginx's own upstream peer. `X-Forwarded-For` is left alone — nothing reads it after this change.

Why not an XFF-based middleware instead? The two ingress paths have different hop counts (one edge hop direct, two via Nginx), so `ClientIPFromXFFTrustedProxies(n)` cannot be correct for both, and Railway's edge addresses are dynamic, which rules out `ClientIPFromXFF` with fixed CIDRs.

**Residual risk, stated plainly:** if `X-Real-IP` ever arrives empty, `GetClientIP` returns `""` and every consumer falls back to `r.RemoteAddr` — for Nginx-proxied traffic that is the shared internal address again. That is a degradation to today's behaviour, not a spoofing hole.

## Verification

- Unit test: a request carrying a forged `True-Client-IP` must not change the IP the rate limiter keys on.
- Unit test: a request carrying a forged `X-Forwarded-For` must not change it either.
- Unit test: with `X-Real-IP` set, the resolved client IP equals that header.
- Unit test: with no proxy headers, the resolved IP falls back to `RemoteAddr`.
- Unit test: two requests with *different* forged `True-Client-IP` values share one limiter bucket (this is the bypass, expressed as a test).
- Existing rate-limit tests must keep passing.
- **Manual, after deploy** — the one claim unit tests cannot cover is that Railway's edge sets `X-Real-IP` for the `pixlinks-web` service too. Probe A proves it for `pixlinks-api`; both sit behind the same Railway HTTP proxy, so this is expected but unverified. Re-run probes B, E and F against the deployed build: B and F must return the real client IP, and E must no longer return `100.64.0.3`.

---

## 2. Stripe Webhook Handler Has No Test Coverage

### The Problem

`internal/handler/billing_handler.go:167` verifies Stripe webhook signatures with `webhook.ConstructEventWithOptions` and dispatches on event type — `checkout.session.completed`, `customer.subscription.created/updated/deleted`. Everything that grants credits, activates pixel slots, and records purchases flows through this one function.

Nothing tests it.

`internal/handler/billing_handler_test.go` holds five tests — `GetBillingOverview`, `GetQuota`, `CreateCheckout`, `UpdateSlots`, `CreatePortalSession` — and none of them touch the `Webhook` method. A repo-wide `grep -rn 'ConstructEvent' --include='*.go'` returns exactly one hit: the production call site. No test constructs a signed payload, and no test asserts what happens when a signature is invalid, when an event type is unknown, or when the same event arrives twice.

This was found on 2026-08-06 while upgrading stripe-go to v85. That release added a guard inside `ConstructEvent*` that rejects V2 thin-event payloads, and the plan called for proving the guard harmless by running "the existing webhook tests" — which turned out not to exist. A throwaway test was written to get the evidence, then deleted to keep the dependency commit clean.

### Why It Matters

Every Stripe SDK major bump changes struct shapes and, through the pinned API version, the payload the handler parses. Between v82 and v86 the pinned API version moved `2025-03-31.basil` → `2025-09-30.clover` → `2025-11-17.clover` → `2026-03-25.dahlia`. The only thing standing between a payload-shape change and silently dropped payments is a human running the Stripe CLI by hand. CI cannot do that — it has no Stripe credentials — so the regression net for the payment path is currently a manual step someone has to remember.

### Scope

Add tests to `billing_handler_test.go` covering:

- A correctly signed `checkout.session.completed` envelope parses, and the handler calls the billing service with the right session.
- A correctly signed `customer.subscription.updated` envelope parses and routes to the subscription path.
- An invalid signature returns 400 and does not touch the service.
- A body whose top-level `object` is not `"event"` is rejected by the v85 guard — this is the regression test for the thin-event case.
- A duplicate `stripe_event_id` is a no-op, exercising the idempotency path through `webhook_events`.

Use `webhook.ComputeSignature` to sign fixtures; no network and no Stripe CLI. These run in CI on every push, which is the point.

### Non-Goals

- Not a replacement for the Stripe CLI check on a real SDK upgrade — that verifies the live payload shape, which fixtures cannot.
- No change to `billing_handler.go` itself unless a test exposes a real defect.
