# Plan — Client IP Spoofing Fix + Stripe Webhook Test Coverage

Implements `docs/superpowers/specs/2026-08-06-dependency-update-followups.md`
(revised 2026-08-06 after probing production — read the *Measured behaviour*
table in that spec before starting; the original draft named the wrong header).

Branch: `fix/client-ip-spoofing-and-webhook-tests`

Three independent tasks, all in one wave. No task depends on another's output.

Project rules that override anything here: `CLAUDE.md` at the repo root.
Surgical changes only — every changed line must trace to this plan. Do not
reformat, refactor, or "improve" adjacent code.

Repo root: `/Users/jaochai/Code/keep-px`

---

## Task 1: Resolve the client IP from a header the client cannot forge

**Files:**
- `backend/go.mod`, `backend/go.sum` (chi bump)
- `backend/internal/router/router.go`
- `backend/internal/middleware/ratelimit.go`
- `backend/internal/middleware/logger.go`
- `backend/internal/handler/event_handler.go`
- `backend/internal/handler/event_handler_test.go` (existing test must change — see step 6)

**Create:**
- `backend/internal/middleware/clientip_test.go`

**Interfaces:** chi v5.3.1 `github.com/go-chi/chi/v5/middleware` provides:

```go
func ClientIPFromHeader(trustedHeader string) func(http.Handler) http.Handler
func GetClientIP(ctx context.Context) string   // "" when nothing was set
```

`ClientIPFromHeader` does **not** mutate `r.RemoteAddr`. It stores the IP in the
request context; `GetClientIP` reads it. If the header is absent or unparseable,
nothing is stored and `GetClientIP` returns `""`.

### Why

`chimiddleware.RealIP` reads `True-Client-IP` first, and nothing in the
deployment strips that header — so any client can set it and choose the IP the
rate limiter keys on. Verified live in production (probes B and F in the spec).

### Steps

1. Bump chi: `cd backend && go get github.com/go-chi/chi/v5@v5.3.1 && go mod tidy`.
   v5.2.5 has no `ClientIPFromHeader`, so this is a prerequisite. Do not bump
   anything else — if `go mod tidy` moves an unrelated dependency, revert that
   line by hand.

2. **RED.** Create `backend/internal/middleware/clientip_test.go`, package
   `middleware`. Follow the style of the existing
   `internal/middleware/ratelimit_apikey_test.go` (plain stdlib `testing`,
   `httptest`, no testify). Write these tests against the real middleware chain
   — build the handler as `chimiddleware.ClientIPFromHeader("X-Real-IP")(
   RateLimitWithContext(ctx, rps)(finalHandler))` so the test exercises the same
   wiring `router.go` uses:

   | Test | Setup | Assert |
   |---|---|---|
   | `TestRateLimitKey_IgnoresForgedTrueClientIP` | limiter rps=1, send 5 requests, each with a **different** `True-Client-IP` and `X-Real-IP: 203.0.113.1`, same `RemoteAddr` | at least one 429 — forging does not buy fresh buckets |
   | `TestRateLimitKey_IgnoresForgedXForwardedFor` | same but rotating `X-Forwarded-For` | at least one 429 |
   | `TestRateLimitKey_SeparatesDistinctXRealIP` | limiter rps=1, 5 requests each with a different `X-Real-IP` | all 200 — genuine distinct clients still get their own bucket |
   | `TestRateLimitKey_FallsBackToRemoteAddr` | no proxy headers at all, rps=1, 5 requests from one `RemoteAddr` | at least one 429 |

   Run `go test ./internal/middleware/ -run TestRateLimitKey -race -count=1`.

   **Expect exactly one RED: `TestRateLimitKey_SeparatesDistinctXRealIP`.** Today
   the limiter keys on `r.RemoteAddr`, so five requests sharing one `RemoteAddr`
   collapse into a single bucket and that test's "all 200" assertion fails. After
   step 4 it keys on the resolved client IP and passes.

   The other three are expected to pass already, and that is correct — they are
   regression guards, not RED drivers. The test chain wires
   `ClientIPFromHeader`, which never touches `r.RemoteAddr`, so forged
   `True-Client-IP` / `X-Forwarded-For` values cannot move the key even before
   the fix. The production bug needs `RealIP` in the chain to reproduce, and
   `RealIP` is what step 3 deletes; pinning a test to it would only test code
   being removed. What these three lock in is that the *new* wiring stays immune.
   The guard against someone re-introducing `RealIP` is the `grep` in **Verify**.

   Report the real `go test` output. Do not describe a test as failing when it
   passed.

3. `internal/router/router.go:30` — replace
   `r.Use(chimiddleware.RealIP)` with
   `r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))`.
   Keep it in the same position in the chain: it must run before `Logger` and
   before the rate limiter.

4. `internal/middleware/ratelimit.go` — replace the `net.SplitHostPort(r.RemoteAddr)`
   block at line ~52 with a call to a new unexported helper in the same file:

   ```go
   // clientIPKey returns the rate-limit key for r: the client IP resolved by
   // chi's ClientIPFromHeader middleware (registered in router.go), falling
   // back to the TCP peer address when no trusted header was present.
   func clientIPKey(r *http.Request) string {
       if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
           return ip
       }
       if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
           return host
       }
       return r.RemoteAddr
   }
   ```

   Import as `chimiddleware "github.com/go-chi/chi/v5/middleware"` — the file's
   own package is also called `middleware`, so the alias is required.
   Delete the now-stale comment at lines 49–51 that describes `RealIP` rewriting
   `r.RemoteAddr`; it is no longer true.

5. `internal/middleware/logger.go` — add the resolved client IP to the log
   attrs, keeping `remote_addr` as it is:

   ```go
   "remote_addr", r.RemoteAddr,
   "client_ip", chimiddleware.GetClientIP(r.Context()),
   ```

6. `internal/handler/event_handler.go` — `extractClientIP` (line ~242) currently
   trusts `CF-Connecting-IP` and `True-Client-IP` straight from the request,
   which is spoofable on its own. Replace the body with:

   ```go
   // extractClientIP returns the client IP resolved by chi's ClientIPFromHeader
   // middleware (registered in router.go), falling back to the TCP peer address.
   // Client-supplied headers are never trusted here — see
   // docs/superpowers/specs/2026-08-06-dependency-update-followups.md.
   func extractClientIP(r *http.Request) string {
       if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
           return ip
       }
       if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
           return host
       }
       return r.RemoteAddr
   }
   ```

   Then remove the `headerCFConnectingIP` and `headerTrueClientIP` constants at
   lines 23–24 — your change is what orphaned them. The `strings` import is
   orphaned too: its only use in this file is the `strings.TrimSpace` on line 245
   inside the function you are replacing. Remove it as well.

7. `internal/handler/event_handler_test.go` — the existing `extractClientIP`
   table test asserts the old header-trusting behaviour and will now fail.
   Rewrite the table so the cases describe the new contract:
   - client IP present in context (set it with
     `chimiddleware.ClientIPFromHeader("X-Real-IP")` wrapping a handler, or by
     driving the request through that middleware) → that value is returned
   - forged `CF-Connecting-IP` / `True-Client-IP`, nothing in context → falls
     back to `RemoteAddr`
   - no headers, nothing in context → falls back to `RemoteAddr`
   - `RemoteAddr` with no port → returned verbatim

   Do not delete the test. It is the regression net for exactly this bug.

8. **GREEN.** `go test ./... -race -count=1` from `backend/`. All packages pass.

### Verify

```bash
cd /Users/jaochai/Code/keep-px/backend
go build ./...
go test ./... -race -count=1
grep -n 'RealIP' internal/router/router.go       # must print nothing
grep -n 'go-chi/chi/v5 v5.3.1' go.mod            # must match
```

### Ask, do not guess

If `go mod tidy` wants to move dependencies other than chi, or if any existing
test outside the files listed above starts failing, stop and write
`$GLM_WS/task-1-question.md` describing what you saw, with a table of options.
Do not "fix" unrelated tests to make the suite green.

---

## Task 2: Stop Nginx from discarding the trustworthy client IP

**Files:** `frontend/nginx.conf`

**Create:** none

**Interfaces:** none — config only.

### Why

Nginx sits behind Railway's edge proxy, so `$remote_addr` at Nginx is the edge
(measured: `100.64.0.3`), not the visitor. Setting `X-Real-IP $remote_addr`
therefore throws away the correct value the edge already put in that header and
replaces it with one fixed internal address. Every request through
`pixlinks.xyz` — including all sale-page tracking SDK calls to
`/api/v1/events/ingest` — ends up sharing a single rate-limit bucket.

### Steps

1. In `frontend/nginx.conf` there are two proxy blocks: `location /p/`
   (line ~49) and `location /api/` (line ~69). In **both**, change:

   ```nginx
   proxy_set_header X-Real-IP $remote_addr;
   ```

   to:

   ```nginx
   # Railway's edge proxy already resolved the real client IP into X-Real-IP and
   # overwrites any client-supplied value; forward it rather than replacing it
   # with our own upstream peer. See
   # docs/superpowers/specs/2026-08-06-dependency-update-followups.md
   proxy_set_header X-Real-IP $http_x_real_ip;
   ```

2. Change nothing else. Leave the `X-Forwarded-For`, `Host` and
   `X-Forwarded-Proto` lines exactly as they are — nothing reads XFF after this
   change, and touching them widens the blast radius for no gain.

### Verify

```bash
cd /Users/jaochai/Code/keep-px
grep -n 'X-Real-IP' frontend/nginx.conf     # both lines must read $http_x_real_ip
git diff --stat frontend/nginx.conf          # exactly one file, small diff
```

The config uses `${...}` envsubst placeholders, so `nginx -t` on the raw file
will not parse and is **not** a valid check. Do not try to run it, and do not
"fix" the placeholders to make it parse.

---

## Task 3: Test coverage for the Stripe webhook handler

**Files:** `backend/internal/handler/billing_handler_test.go`

**Create:** none — extend the existing file.

**Interfaces:**

```go
// handler under test
func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request)

// stripe-go v86 test helper — no network, no Stripe CLI
func webhook.ComputeSignature(t time.Time, payload []byte, secret string) []byte

// idempotency path the handler reaches through the service
func (s *BillingService) ProcessWebhookEvent(ctx context.Context, stripeEventID, eventType string, process func() error) error
//   → webhookRepo.CreateIfNotExists(ctx, id, type) (inserted bool, err error)
//   → if !inserted: no-op, returns nil
//   → if process() fails: webhookRepo.Delete(ctx, id), returns the error
```

### Why

`billing_handler.go:167` is the single path through which every credit grant,
pixel-slot activation and purchase record flows, and nothing tests it. Every
Stripe SDK major bump changes struct shapes and the pinned API version; today
the only regression net is a human remembering to run the Stripe CLI. CI has no
Stripe credentials and cannot do that. Signed fixtures can, and they run on
every push.

### Steps

1. Read the existing helpers at the top of `billing_handler_test.go`
   (`nbMockPool`, `nbBillingConfig`, `nbNewTestBillingService`). Follow that
   pattern exactly: a **real** `service.BillingService` wired to mock repos from
   `internal/repository/mocks`. Do not introduce an interface for
   `BillingService` and do not modify `billing_handler.go` — the handler takes a
   concrete `*service.BillingService` and that is out of scope here.

2. Add a config helper alongside `nbBillingConfig` that sets a webhook secret,
   e.g. `whsec_test_secret`, since `nbBillingConfig` intentionally leaves Stripe
   keys empty. Add a signing helper that builds the `Stripe-Signature` header
   from `webhook.ComputeSignature` in the format the SDK expects
   (`t=<unix>,v1=<hex>`); read `webhook/client.go` in the module cache if you
   need to confirm the exact format rather than guessing.

3. Add these tests. Name them `TestBillingHandler_Webhook_<case>`:

   | Case | Setup | Assert |
   |---|---|---|
   | `ValidCheckoutSessionCompleted` | correctly signed `checkout.session.completed` envelope; `MockWebhookEventRepo.CreateIfNotExists` returns `(true, nil)` | 200; `CreateIfNotExists` called once with the event's ID and type |
   | `ValidSubscriptionUpdated` | correctly signed `customer.subscription.updated` envelope | 200; routes down the subscription path, not the checkout path |
   | `InvalidSignature` | body signed with a different secret | 400; **no** repo mock is called at all |
   | `NonEventObjectRejected` | correctly signed body whose top-level `"object"` is not `"event"` | 400 — this is the regression test for the v85 thin-event guard inside `ConstructEvent*` |
   | `DuplicateEventIsNoOp` | correctly signed envelope; `CreateIfNotExists` returns `(false, nil)` | 200; the downstream handler is never reached — assert via the repo mocks that no purchase/credit/subscription write happened |
   | `UnknownEventType` | correctly signed envelope with a type the handler does not switch on | 200; no repo mock called |

   Use `mock.AssertExpectations` / `AssertNotCalled` as the existing tests in
   this file do.

4. Keep the five existing tests in the file passing untouched.

### Verify

```bash
cd /Users/jaochai/Code/keep-px/backend
go test ./internal/handler/ -run TestBillingHandler -race -count=1 -v
go test ./... -race -count=1
```

Every new test must genuinely exercise `ConstructEventWithOptions` — if you find
yourself stubbing out signature verification, stop: that defeats the entire
point of the task.

### Ask, do not guess

If a mock in `internal/repository/mocks` lacks a method you need, or if the
duplicate-event assertion cannot be expressed with the existing mocks, stop and
write `$GLM_WS/task-3-question.md` with what you found and the options. Do not
add production code to make a test convenient.

---

## Out of scope for all tasks

- No change to `RateLimitByAPIKey` — unaffected.
- No new rate-limit thresholds, tiers or strategies.
- No `git commit`, no `git push`, no deploy, no env changes.
- No edits to files not listed under a task's **Files:**.
