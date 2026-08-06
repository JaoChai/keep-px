# Follow-ups Found During the 2026-08-06 Dependency Update

Two issues surfaced while upgrading dependencies. Neither was caused by that work; both were found because the upgrade made tooling look at code nobody had looked at in a while. Each needs its own PR.

---

## 1. Client IP Spoofing — Rate Limiter Bypass

**Date**: 2026-08-06
**Status**: Found during the dependency update; not yet scheduled
**Type**: Security fix — changes request-IP behaviour behind the proxy

## The Problem

`internal/router/router.go:30` registers `chimiddleware.RealIP`. That middleware rewrites `r.RemoteAddr` from request headers, trusting the **leftmost** value of `X-Forwarded-For`, or `True-Client-IP` / `X-Real-IP`, regardless of whether the infrastructure actually set them.

`frontend/nginx.conf:50` sets:

```nginx
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

`$proxy_add_x_forwarded_for` **appends** the real peer address to whatever the client already sent. A request arriving with `X-Forwarded-For: 203.0.113.9` reaches the backend as `203.0.113.9, <real ip>` — and `RealIP` takes the first entry. The client controls it.

Three consumers read the resulting value:

| Site | Use | Impact when spoofed |
|---|---|---|
| `internal/middleware/ratelimit.go:52` | per-IP rate limiting | **Bypass.** Rotate a fresh forged IP per request and the limiter never trips. |
| `internal/handler/event_handler.go:250` | `client_ip_address` sent to Facebook CAPI | Garbage IPs degrade EMQ score; attacker can attribute events to arbitrary addresses. |
| `internal/middleware/logger.go:32` | request logging | Logs and any incident forensics built on them are unreliable. |

The rate-limit bypass is the load-bearing one. `/api/v1/events/ingest` is protected by both a global limiter and a per-API-key limiter (`RateLimitByAPIKey`, added in #210); the per-IP layer is the one that fails open here.

chi v5.3.1 marks `RealIP` deprecated for exactly this reason and ships replacements. This was discovered on 2026-08-06 when a chi bump surfaced the deprecation during a dependency update — **the vulnerability itself predates that bump and is live in production today.**

## Goals

- Derive the client IP from a source the client cannot forge.
- Keep per-IP rate limiting, CAPI `client_ip_address`, and request logging working behind Nginx on Railway.
- No change to any API contract.

## Non-Goals

- No change to the per-API-key rate limiter — it is unaffected.
- No new rate-limiting strategy, tiers, or thresholds.
- No change to how Nginx is deployed. The fix is in Go.

## Design

Replace `RealIP` with chi's `ClientIPFromHeader("X-Real-IP")`.

Nginx sets `X-Real-IP $remote_addr` (`nginx.conf:49`) — the actual TCP peer as Nginx sees it, overwritten on every request, never appended to. A client-supplied `X-Real-IP` is discarded at the proxy, so the value the backend reads is trustworthy as long as traffic cannot reach the backend without passing through Nginx.

The replacement middlewares do **not** mutate `r.RemoteAddr`; the value is read with `middleware.GetClientIP(r)`. Every consumer must therefore change too:

| File | Change |
|---|---|
| `internal/router/router.go:30` | `chimiddleware.RealIP` → `chimiddleware.ClientIPFromHeader("X-Real-IP")` |
| `internal/middleware/ratelimit.go:52` | read `GetClientIP(r)`, fall back to `r.RemoteAddr` when empty |
| `internal/handler/event_handler.go:250` | same substitution in the CAPI IP helper |
| `internal/middleware/logger.go:32` | log the resolved client IP alongside `RemoteAddr` |

This also unblocks the `go-chi/chi/v5` upgrade to v5.3.1, which was excluded from the 2026-08-06 dependency update because of the deprecation.

**Open question for implementation:** confirm that Railway's edge does not add its own proxy hop in front of the container's Nginx. If it does, `X-Real-IP` would carry Railway's edge address rather than the end user's, and the correct choice becomes `ClientIPFromXFFTrustedProxies(n)` with `n` matching the real hop count. Verify against a live request before choosing.

## Verification

- Unit test: a request carrying a forged `X-Forwarded-For` must not change the IP the rate limiter keys on.
- Unit test: with `X-Real-IP` set, the resolved client IP equals that header.
- Unit test: with no proxy headers, the resolved IP falls back to `RemoteAddr`.
- Manual: on a deployed instance, send a request with a forged `X-Forwarded-For` and confirm the logged IP is the real one.
- Existing rate-limit tests must keep passing.

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
