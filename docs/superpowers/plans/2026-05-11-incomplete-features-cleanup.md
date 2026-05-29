# Incomplete Features Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the verified gaps in Keep-PX: remove debug leaks, harden CI secrets, add Kubernetes-style readiness probe, add per-API-key rate limiting, harden E2E coverage by replacing conditional skips with seeded data, and expand BlockEditor block types.

**Architecture:** Four independent phases. Each phase produces a shippable PR with passing CI. Phases can be paused/resumed between PRs. Test coverage expansion for `repository/postgres/` and `frontend/src/` is excluded from this plan (separate plan needed — too large).

**Tech Stack:** Go 1.25 (chi, pgx/v5, slog, golang.org/x/time/rate), React 18 + TypeScript + Vite + TanStack Query, Playwright E2E, GitHub Actions CI.

---

## File Structure Overview

**Phase 1 — Quick fixes:**
- Modify: `backend/internal/facebook/capi.go`
- Modify: `backend/internal/handler/health.go`
- Modify: `backend/internal/router/router.go`
- Modify: `.github/workflows/ci.yml`

**Phase 2 — Per-API-key rate limit:**
- Create: `backend/internal/middleware/ratelimit_apikey.go`
- Create: `backend/internal/middleware/ratelimit_apikey_test.go`
- Modify: `backend/internal/middleware/ratelimit.go`
- Modify: `backend/internal/router/router.go`

**Phase 3 — E2E hardening:**
- Create: `frontend/e2e/support/seed.ts`
- Modify: `frontend/e2e/support/global-setup.ts`
- Modify: `frontend/e2e/tests/replay.spec.ts`
- Modify: `frontend/e2e/tests/event-flow.spec.ts`
- Modify: `frontend/e2e/tests/event-log.spec.ts`

**Phase 4 — BlockEditor expansion:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/components/sale-pages/BlockEditor.tsx`
- Modify: `backend/internal/templates/sale_pages/blocks.html`
- Modify: `backend/internal/templates/sale_pages/simple.html`
- Modify: `backend/internal/templates/sale_pages/tracking.html`
- Modify: `frontend/e2e/tests/sale-pages.spec.ts`

---

# Phase 1 — Quick Fixes

Ships as one PR titled `chore: production hygiene cleanup (capi log, /ready, ci secret)`.

## Task 1: Remove CAPI debug log

**Files:**
- Modify: `backend/internal/facebook/capi.go:73-80`

- [ ] **Step 1: Open the file and remove the debug block**

Replace lines 73-80 (the `// Debug: log first 500 bytes ...` block through the closing `}` and `fmt.Printf`) with nothing. The next line `req, err := http.NewRequestWithContext(...)` should follow directly after the `body, err := json.Marshal(reqBody)` block.

After edit, the function reads:

```go
body, err := json.Marshal(reqBody)
if err != nil {
    return nil, fmt.Errorf("marshal request: %w", err)
}

req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
```

- [ ] **Step 2: Remove unused `fmt` import if needed**

Run: `cd backend && go build ./internal/facebook/...`
Expected: PASS (if it fails with "imported and not used: fmt", remove `"fmt"` from imports — but `fmt.Sprintf` and `fmt.Errorf` are still used elsewhere in the file, so it should stay)

- [ ] **Step 3: Run package tests**

Run: `cd backend && go test ./internal/facebook/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/facebook/capi.go
git commit -m "chore: remove temporary CAPI debug log from PR #201"
```

## Task 2: Add `/ready` endpoint

**Files:**
- Modify: `backend/internal/handler/health.go`
- Create: `backend/internal/handler/health_test.go`
- Modify: `backend/internal/router/router.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/health_test.go`:

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReady_ReturnsOK(t *testing.T) {
	h := &HealthHandler{pool: nil}
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"status":"ready"}` && got != `{"status":"ready"}`+"\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handler/ -run TestReady_ReturnsOK -v`
Expected: FAIL with "h.Ready undefined"

- [ ] **Step 3: Add Ready handler**

Append to `backend/internal/handler/health.go` after the `Health` method:

```go
// Ready is a lightweight liveness check that does not touch the DB.
// Use this for Railway/Kubernetes liveness probes; use /health for readiness with DB ping.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}` + "\n"))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/handler/ -run TestReady_ReturnsOK -v`
Expected: PASS

- [ ] **Step 5: Register the route**

Edit `backend/internal/router/router.go`. Find the existing health route registration (search for `healthHandler.Health` or `/health`). Add the `/ready` route directly below it:

```go
r.Get("/health", healthHandler.Health)
r.Get("/ready", healthHandler.Ready)
```

- [ ] **Step 6: Build and verify**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handler/health.go backend/internal/handler/health_test.go backend/internal/router/router.go
git commit -m "feat: add /ready endpoint for liveness probe"
```

## Task 3: Move JWT_SECRET out of CI inline script

**Files:**
- Modify: `.github/workflows/ci.yml:200-215` (the E2E setup step that uses crypto.createHmac with JWT_SECRET)

- [ ] **Step 1: Read context around line 207**

Run: `sed -n '195,225p' .github/workflows/ci.yml`
Expected: View the surrounding step. The line is `const sig = crypto.createHmac('sha256','${{ secrets.JWT_SECRET }}').update(...)`.

- [ ] **Step 2: Refactor to read from env**

In `.github/workflows/ci.yml`, locate the step that contains the `crypto.createHmac` line. Add an `env:` block to that step exposing `JWT_SECRET`, and change the inline script to read from `process.env.JWT_SECRET`. The full step should look like:

```yaml
      - name: Generate E2E JWT
        env:
          JWT_SECRET: ${{ secrets.JWT_SECRET }}
        run: |
          node -e "
          const crypto = require('crypto');
          const secret = process.env.JWT_SECRET;
          if (!secret) { console.error('JWT_SECRET missing'); process.exit(1); }
          const header = Buffer.from(JSON.stringify({alg:'HS256',typ:'JWT'})).toString('base64url');
          const payload = Buffer.from(JSON.stringify({sub:'e2e-user',exp:Math.floor(Date.now()/1000)+3600})).toString('base64url');
          const sig = crypto.createHmac('sha256', secret).update(header+'.'+payload).digest('base64url');
          console.log(header+'.'+payload+'.'+sig);
          "
```

Match the surrounding step's indentation exactly.

- [ ] **Step 3: Lint the workflow file**

Run: `cd /Users/jaochai/Code/keep-px && yamllint .github/workflows/ci.yml || true`
Expected: No new errors introduced (if yamllint is not installed, skip)

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: pass JWT_SECRET via env instead of inline interpolation"
```

## Task 4: Open Phase 1 PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin HEAD
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "chore: production hygiene cleanup" --body "$(cat <<'EOF'
## Summary
- Remove temporary CAPI debug log (PR #201 follow-up)
- Add /ready endpoint for liveness probes
- Move JWT_SECRET out of inline CI script

## Test Plan
- [ ] go test ./... passes
- [ ] curl /ready returns 200
- [ ] CI workflow runs green
EOF
)"
```

- [ ] **Step 3: Wait for CI green, then merge**

---

# Phase 2 — Per-API-Key Rate Limiting

Ships as one PR titled `feat(security): per-API-key rate limit for /events/ingest`.

## Task 5: Define rate limit middleware

**Files:**
- Create: `backend/internal/middleware/ratelimit_apikey.go`
- Create: `backend/internal/middleware/ratelimit_apikey_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/middleware/ratelimit_apikey_test.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitByAPIKey_AllowsUnderLimit(t *testing.T) {
	mw := RateLimitByAPIKey(5, 5) // 5 rps, burst 5
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, "key-a"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimitByAPIKey_BlocksWhenExceeded(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, "key-b"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if i > 0 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d: want 429, got %d", i, w.Code)
		}
	}
}

func TestRateLimitByAPIKey_SeparateBucketsPerKey(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, key := range []string{"key-c", "key-d"} {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), apiKeyCtxKey{}, key))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("key %s: want 200, got %d", key, w.Code)
		}
	}
}

func TestRateLimitByAPIKey_PassthroughWhenNoKey(t *testing.T) {
	mw := RateLimitByAPIKey(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("no key: want 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/middleware/ -run RateLimitByAPIKey -v`
Expected: FAIL with "undefined: RateLimitByAPIKey" and "undefined: apiKeyCtxKey"

- [ ] **Step 3: Inspect existing APIKey middleware to find context key**

Run: `grep -n "apiKey\|APIKey\|ctxKey" backend/internal/middleware/auth.go backend/internal/middleware/apikey.go 2>/dev/null | head -30`
Expected: Find the existing context key constant or type. If the existing middleware uses a different ctx key name, align Task 7 with whatever exists. If no API key context key exists yet, the new middleware will define `apiKeyCtxKey{}`.

- [ ] **Step 4: Create the middleware**

Create `backend/internal/middleware/ratelimit_apikey.go`:

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type apiKeyCtxKey struct{}

const apiKeyLimiterTTL = 1 * time.Hour

type apiKeyLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type apiKeyLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*apiKeyLimiter
	rps      rate.Limit
	burst    int
}

func newAPIKeyLimiterStore(rps int, burst int) *apiKeyLimiterStore {
	s := &apiKeyLimiterStore{
		limiters: make(map[string]*apiKeyLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go s.gcLoop()
	return s
}

func (s *apiKeyLimiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.limiters[key]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := &apiKeyLimiter{
		limiter:  rate.NewLimiter(s.rps, s.burst),
		lastSeen: time.Now(),
	}
	s.limiters[key] = l
	return l.limiter
}

func (s *apiKeyLimiterStore) gcLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-apiKeyLimiterTTL)
		for k, l := range s.limiters {
			if l.lastSeen.Before(cutoff) {
				delete(s.limiters, k)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitByAPIKey returns middleware that rate-limits requests by the API key
// stored in request context under apiKeyCtxKey{}. Requests without an API key
// pass through unchanged (defer to other middleware for auth enforcement).
func RateLimitByAPIKey(rps int, burst int) func(http.Handler) http.Handler {
	store := newAPIKeyLimiterStore(rps, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyVal := r.Context().Value(apiKeyCtxKey{})
			key, ok := keyVal.(string)
			if !ok || key == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !store.get(key).Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/middleware/ -run RateLimitByAPIKey -v`
Expected: PASS (4 tests)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/middleware/ratelimit_apikey.go backend/internal/middleware/ratelimit_apikey_test.go
git commit -m "feat(middleware): add per-API-key rate limiter"
```

## Task 6: Wire API key into request context from APIKeyAuth middleware

**Files:**
- Modify: `backend/internal/middleware/apikey.go` (or wherever the existing API key auth middleware lives — verify path in Task 5 Step 3)

- [ ] **Step 1: Locate existing API key middleware**

Run: `grep -rln "X-API-Key\|x-api-key" backend/internal/middleware/`
Expected: One file path (e.g., `apikey.go` or `auth.go`).

- [ ] **Step 2: Add context injection**

In the file from Step 1, find the function that extracts the API key from `r.Header.Get("X-API-Key")` and validates it. After the validation passes (the part that calls `next.ServeHTTP`), wrap the request:

```go
ctx := context.WithValue(r.Context(), apiKeyCtxKey{}, key)
next.ServeHTTP(w, r.WithContext(ctx))
```

Where `key` is whatever variable holds the validated raw API key string. If `context` is not imported, add it.

- [ ] **Step 3: Build and run middleware tests**

Run: `cd backend && go test ./internal/middleware/...`
Expected: PASS (all middleware tests)

- [ ] **Step 4: Commit**

```bash
git add backend/internal/middleware/
git commit -m "feat(middleware): inject API key into request context"
```

## Task 7: Mount per-API-key limiter on `/events/ingest`

**Files:**
- Modify: `backend/internal/router/router.go`
- Modify: `backend/internal/config/config.go` (add config field)

- [ ] **Step 1: Add config field**

In `backend/internal/config/config.go`, add a field to the Config struct alongside other rate limit fields:

```go
RateLimitAPIKeyRPS   int `env:"RATE_LIMIT_API_KEY_RPS" envDefault:"100"`
RateLimitAPIKeyBurst int `env:"RATE_LIMIT_API_KEY_BURST" envDefault:"200"`
```

- [ ] **Step 2: Update `.env.example`**

Append to `backend/.env.example`:

```
# Per-API-key rate limit for /events/ingest (requests per second / burst)
RATE_LIMIT_API_KEY_RPS=100
RATE_LIMIT_API_KEY_BURST=200
```

- [ ] **Step 3: Mount middleware on ingest route**

In `backend/internal/router/router.go`, find the `/events/ingest` route registration (it uses APIKeyAuth middleware). Insert `RateLimitByAPIKey` immediately after `APIKeyAuth` so the key is available in context:

```go
r.With(
    middleware.APIKeyAuth(customerRepo),
    middleware.RateLimitByAPIKey(cfg.RateLimitAPIKeyRPS, cfg.RateLimitAPIKeyBurst),
).Post("/events/ingest", eventHandler.Ingest)
```

If the existing registration uses `r.Group` or `r.Route` instead of `r.With`, follow the existing pattern.

- [ ] **Step 4: Build and run all backend tests**

Run: `cd backend && go vet ./... && go test -race ./...`
Expected: PASS

- [ ] **Step 5: Commit and PR**

```bash
git add backend/internal/config/config.go backend/.env.example backend/internal/router/router.go
git commit -m "feat(security): mount per-API-key rate limit on /events/ingest"
git push -u origin HEAD
gh pr create --title "feat(security): per-API-key rate limit for /events/ingest" --body "$(cat <<'EOF'
## Summary
- Add per-API-key token bucket rate limiter (defaults 100 rps, burst 200)
- Inject API key into request context from APIKeyAuth middleware
- Mount limiter on POST /events/ingest

## Test Plan
- [ ] Unit tests pass (4 new middleware tests)
- [ ] go vet + go test -race pass
- [ ] Manual: hammer /events/ingest with one key, see 429
- [ ] Manual: two keys in parallel each get full bucket
EOF
)"
```

---

# Phase 3 — E2E Test Hardening

Ships as one PR titled `test(e2e): seed test user data and remove conditional skips`.

## Task 8: Create E2E seed script

**Files:**
- Create: `frontend/e2e/support/seed.ts`

- [ ] **Step 1: Inspect existing global-setup for auth pattern**

Run: `cat frontend/e2e/support/global-setup.ts`
Expected: Shows how the test user is currently provisioned (likely dev-login or direct DB insert).

- [ ] **Step 2: Create the seed script**

Create `frontend/e2e/support/seed.ts`:

```typescript
import { request, type APIRequestContext } from '@playwright/test'

const API_BASE = process.env.E2E_API_BASE ?? 'http://localhost:8080'
const TEST_EMAIL = process.env.E2E_TEST_EMAIL ?? 'e2e@keep-px.test'

export interface SeedResult {
  accessToken: string
  customerId: string
  pixelId: string
}

export async function seedTestData(): Promise<SeedResult> {
  const ctx: APIRequestContext = await request.newContext()

  const loginRes = await ctx.post(`${API_BASE}/api/v1/auth/dev-login`, {
    data: { email: TEST_EMAIL },
  })
  if (!loginRes.ok()) {
    throw new Error(`dev-login failed: ${loginRes.status()} ${await loginRes.text()}`)
  }
  const { access_token: accessToken, customer } = await loginRes.json()

  const pixelsRes = await ctx.get(`${API_BASE}/api/v1/pixels`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
  const pixels = (await pixelsRes.json()).data ?? []

  let pixelId: string
  if (pixels.length > 0) {
    pixelId = pixels[0].id
  } else {
    const createRes = await ctx.post(`${API_BASE}/api/v1/pixels`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      data: {
        name: 'E2E Seed Pixel',
        pixel_id: '000000000000001',
        access_token: 'seed-token',
      },
    })
    if (!createRes.ok()) {
      throw new Error(`create pixel failed: ${createRes.status()} ${await createRes.text()}`)
    }
    pixelId = (await createRes.json()).id
  }

  await ctx.dispose()
  return { accessToken, customerId: customer.id, pixelId }
}
```

- [ ] **Step 3: Wire seed into global-setup**

Edit `frontend/e2e/support/global-setup.ts`. After the existing login/auth setup, add:

```typescript
import { seedTestData } from './seed'

// ... existing code ...

export default async function globalSetup() {
  // ... existing setup ...
  const seed = await seedTestData()
  process.env.E2E_SEED_PIXEL_ID = seed.pixelId
  process.env.E2E_SEED_ACCESS_TOKEN = seed.accessToken
}
```

If `global-setup.ts` already exports a default function, append the seed call to it rather than creating a duplicate.

- [ ] **Step 4: Run E2E to verify seed works**

Run: `cd frontend && npm run e2e -- --grep "@smoke" --reporter=line`
Expected: Smoke tests pass and seed runs without error in setup output.

- [ ] **Step 5: Commit**

```bash
git add frontend/e2e/support/seed.ts frontend/e2e/support/global-setup.ts
git commit -m "test(e2e): add seed script for guaranteed pixel availability"
```

## Task 9: Remove conditional skips in replay tests

**Files:**
- Modify: `frontend/e2e/tests/replay.spec.ts:59,77,99,107,129,150,177,197,205,285`

- [ ] **Step 1: Inspect existing skips**

Run: `grep -n "test.skip" frontend/e2e/tests/replay.spec.ts`
Expected: List of `test.skip(...)` calls with conditional messages.

- [ ] **Step 2: Remove "no pixels" skips**

For each `test.skip` that says "no pixels available" or similar pixel-presence check, delete the entire skip block. The seed script guarantees at least one pixel exists.

Example transformation — before:

```typescript
const pixels = await getPixelCount(page)
test.skip(pixels === 0, 'no pixels available')
```

After: delete those two lines entirely.

- [ ] **Step 3: Convert "no credits" skips to seed grant**

For skips guarded on `noCredits === true`, replace with an admin-grant call in test setup. Add at top of replay test file:

```typescript
import { request } from '@playwright/test'

async function grantCredits(pixelId: string, amount: number, adminToken: string) {
  const ctx = await request.newContext()
  await ctx.post(`${process.env.E2E_API_BASE ?? 'http://localhost:8080'}/api/v1/admin/credits/grant`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { pixel_id: pixelId, amount },
  })
  await ctx.dispose()
}
```

If the admin grant endpoint does not exist, instead add it to the seed script (Task 8) and remove this Step 3 entirely.

- [ ] **Step 4: Run replay tests**

Run: `cd frontend && npm run e2e -- frontend/e2e/tests/replay.spec.ts --reporter=line`
Expected: All previously-skipped tests now execute. Any genuine failures get reported.

- [ ] **Step 5: Commit**

```bash
git add frontend/e2e/tests/replay.spec.ts
git commit -m "test(e2e): remove conditional skips in replay tests"
```

## Task 10: Remove conditional skips in event-flow and event-log tests

**Files:**
- Modify: `frontend/e2e/tests/event-flow.spec.ts:138,152`
- Modify: `frontend/e2e/tests/event-log.spec.ts:80,94`

- [ ] **Step 1: Inspect skips in event-flow**

Run: `grep -n "test.skip" frontend/e2e/tests/event-flow.spec.ts frontend/e2e/tests/event-log.spec.ts`
Expected: 4 skip calls total.

- [ ] **Step 2: Replace live-polling skips with explicit wait**

For each skip with reason "live polling may override state in CI" or "backend may be unavailable", remove the skip and instead `await page.waitForResponse(/\/events\/recent/)` before the assertion that depends on polling.

Example transformation — before:

```typescript
test.skip(!hasLiveData, 'live polling may override state in CI')
await expect(page.getByText('Live')).toBeVisible()
```

After:

```typescript
await page.waitForResponse((res) => res.url().includes('/events/recent') && res.ok())
await expect(page.getByText('Live')).toBeVisible()
```

- [ ] **Step 3: Run the two test files**

Run: `cd frontend && npm run e2e -- frontend/e2e/tests/event-flow.spec.ts frontend/e2e/tests/event-log.spec.ts --reporter=line`
Expected: PASS

- [ ] **Step 4: Commit and open PR**

```bash
git add frontend/e2e/tests/event-flow.spec.ts frontend/e2e/tests/event-log.spec.ts
git commit -m "test(e2e): remove live-polling conditional skips"
git push -u origin HEAD
gh pr create --title "test(e2e): seed test data and remove conditional skips" --body "$(cat <<'EOF'
## Summary
- Add e2e seed script that guarantees pixel + credits exist
- Remove 14+ conditional test.skip() calls that masked coverage
- Convert live-polling skips into explicit waitForResponse

## Test Plan
- [ ] cd frontend && npm run e2e passes locally
- [ ] All previously-skipped tests now run
EOF
)"
```

---

# Phase 4 — BlockEditor Block Type Expansion

Ships as one PR titled `feat(sale-pages): add video and divider blocks to editor`.

## Task 11: Extend block type union

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Locate current block type**

Run: `grep -n "BlockType\|block_type\|kind.*image.*text.*button" frontend/src/types/index.ts`
Expected: Find the existing union or interface.

- [ ] **Step 2: Add new variants**

In `frontend/src/types/index.ts`, find the block type definition. Extend the union:

```typescript
export type SalePageBlock =
  | { kind: 'image'; src: string; alt?: string }
  | { kind: 'text'; html: string }
  | { kind: 'button'; label: string; href: string }
  | { kind: 'video'; src: string; poster?: string; autoplay?: boolean }
  | { kind: 'divider'; style?: 'solid' | 'dashed' | 'dotted'; thickness?: number }
```

Match the existing block schema's field naming (`src` vs `url`, etc.) — if existing uses different keys, mirror those for the new variants.

- [ ] **Step 3: Verify type compiles**

Run: `cd frontend && npx tsc -b --noEmit`
Expected: PASS (no errors in types/index.ts)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "feat(types): add video and divider block variants"
```

## Task 12: Render video block in editor

**Files:**
- Modify: `frontend/src/components/sale-pages/BlockEditor.tsx`

- [ ] **Step 1: Locate the block renderer switch**

Run: `grep -n "case 'image'\|case 'text'\|case 'button'" frontend/src/components/sale-pages/BlockEditor.tsx`
Expected: Find the switch statement that renders each block type.

- [ ] **Step 2: Add video and divider cases**

In `BlockEditor.tsx`, inside the renderer switch, add:

```typescript
case 'video':
  return (
    <div className="space-y-2">
      <input
        type="url"
        placeholder="Video URL (mp4 or YouTube embed)"
        value={block.src}
        onChange={(e) => onChange({ ...block, src: e.target.value })}
        className="w-full rounded border px-3 py-2"
      />
      <input
        type="url"
        placeholder="Poster image URL (optional)"
        value={block.poster ?? ''}
        onChange={(e) => onChange({ ...block, poster: e.target.value })}
        className="w-full rounded border px-3 py-2"
      />
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={block.autoplay ?? false}
          onChange={(e) => onChange({ ...block, autoplay: e.target.checked })}
        />
        Autoplay
      </label>
    </div>
  )

case 'divider':
  return (
    <div className="flex items-center gap-2">
      <select
        value={block.style ?? 'solid'}
        onChange={(e) => onChange({ ...block, style: e.target.value as 'solid' | 'dashed' | 'dotted' })}
        className="rounded border px-2 py-1"
      >
        <option value="solid">Solid</option>
        <option value="dashed">Dashed</option>
        <option value="dotted">Dotted</option>
      </select>
      <input
        type="number"
        min={1}
        max={10}
        value={block.thickness ?? 1}
        onChange={(e) => onChange({ ...block, thickness: Number(e.target.value) })}
        className="w-20 rounded border px-2 py-1"
      />
      <span className="text-sm text-muted-foreground">px</span>
    </div>
  )
```

Match the surrounding styling conventions (Tailwind classes) of the existing image/text/button cases.

- [ ] **Step 3: Add "Add Video" and "Add Divider" buttons**

Find the "Add Image / Add Text / Add Button" toolbar (search for `'Add Image'` or `kind: 'image'`). Add two more buttons next to them that push the corresponding default block:

```typescript
<button
  type="button"
  onClick={() => onAdd({ kind: 'video', src: '' })}
  className="rounded border px-3 py-1"
>
  เพิ่มวิดีโอ
</button>
<button
  type="button"
  onClick={() => onAdd({ kind: 'divider', style: 'solid', thickness: 1 })}
  className="rounded border px-3 py-1"
>
  เพิ่มเส้นคั่น
</button>
```

- [ ] **Step 4: Type-check and lint**

Run: `cd frontend && npx tsc -b --noEmit && npm run lint`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/sale-pages/BlockEditor.tsx
git commit -m "feat(editor): render video and divider blocks in BlockEditor"
```

## Task 13: Render video and divider in sale page templates

**Files:**
- Modify: `backend/internal/templates/sale_pages/blocks.html`
- Modify: `backend/internal/templates/sale_pages/simple.html`
- Modify: `backend/internal/templates/sale_pages/tracking.html`

- [ ] **Step 1: Inspect blocks.html for existing block rendering**

Run: `grep -n '\.Kind\|{{ if\|{{- if' backend/internal/templates/sale_pages/blocks.html`
Expected: Find the Go template `if`/`switch` that renders each block kind.

- [ ] **Step 2: Add video and divider rendering to blocks.html**

In `backend/internal/templates/sale_pages/blocks.html`, inside the block loop, add (matching the existing template syntax):

```html
{{ else if eq .Kind "video" }}
  <div class="block block-video">
    <video
      src="{{ .Src }}"
      {{ if .Poster }}poster="{{ .Poster }}"{{ end }}
      {{ if .Autoplay }}autoplay muted playsinline{{ end }}
      controls
      style="width:100%;max-width:100%;"
    ></video>
  </div>
{{ else if eq .Kind "divider" }}
  <hr class="block block-divider" style="border-style:{{ or .Style "solid" }};border-width:{{ or .Thickness 1 }}px 0 0 0;" />
```

Field names (`.Src`, `.Poster`, etc.) must match the Go struct tag mapping for the block. Verify against `backend/internal/domain/sale_page.go` or wherever blocks are defined; if the Go struct uses different field names, mirror those.

- [ ] **Step 3: Mirror the changes in simple.html and tracking.html**

Repeat Step 2 verbatim in both other template files. **Critical:** all three templates must stay in sync (per CLAUDE.md gotcha — forgetting one causes silent tracking failure).

- [ ] **Step 4: Verify Go struct supports new fields**

Run: `grep -n "type SalePageBlock\|type Block " backend/internal/domain/*.go`
Expected: Find the block struct. If `Src`, `Poster`, `Autoplay`, `Style`, `Thickness` are not fields, extend the struct:

```go
type SalePageBlock struct {
    Kind      string `json:"kind"`
    Src       string `json:"src,omitempty"`
    Alt       string `json:"alt,omitempty"`
    HTML      string `json:"html,omitempty"`
    Label     string `json:"label,omitempty"`
    Href      string `json:"href,omitempty"`
    Poster    string `json:"poster,omitempty"`
    Autoplay  bool   `json:"autoplay,omitempty"`
    Style     string `json:"style,omitempty"`
    Thickness int    `json:"thickness,omitempty"`
}
```

If the existing struct uses a different shape (interface, map, etc.), align with that pattern instead.

- [ ] **Step 5: Run backend tests**

Run: `cd backend && go test ./...`
Expected: PASS

- [ ] **Step 6: Manually render a sale page locally**

Run: `cd backend && go run cmd/server/main.go` (in one terminal) and `cd frontend && npm run dev` (in another). Open the editor, add a video block, save, then open the public sale page URL (`/p/<slug>`). Verify video renders.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/templates/sale_pages/blocks.html backend/internal/templates/sale_pages/simple.html backend/internal/templates/sale_pages/tracking.html backend/internal/domain/sale_page.go
git commit -m "feat(templates): render video and divider blocks on public sale pages"
```

## Task 14: Add E2E coverage for new block types

**Files:**
- Modify: `frontend/e2e/tests/sale-pages.spec.ts` (or create if missing)

- [ ] **Step 1: Locate existing sale page E2E**

Run: `ls frontend/e2e/tests/ | grep -i sale`
Expected: One or more sale page spec files.

- [ ] **Step 2: Add test for video block creation**

Append to the relevant spec file:

```typescript
test('add video block in editor and verify on public page', async ({ page }) => {
  await page.goto('/sale-pages/new')
  await page.getByRole('button', { name: 'เพิ่มวิดีโอ' }).click()
  await page.getByPlaceholder('Video URL (mp4 or YouTube embed)').fill('https://example.com/test.mp4')
  await page.getByRole('button', { name: /บันทึก|Save/ }).click()

  await page.waitForURL(/\/sale-pages\/[^/]+/)
  const slugMatch = page.url().match(/\/sale-pages\/([^/]+)/)
  const slug = slugMatch?.[1]
  expect(slug).toBeTruthy()

  await page.goto(`/p/${slug}`)
  await expect(page.locator('video[src="https://example.com/test.mp4"]')).toBeVisible()
})
```

Adjust button/placeholder text to match what Task 12 produced.

- [ ] **Step 3: Run the new test**

Run: `cd frontend && npm run e2e -- frontend/e2e/tests/sale-pages.spec.ts -g "video block"`
Expected: PASS

- [ ] **Step 4: Commit and open PR**

```bash
git add frontend/e2e/tests/sale-pages.spec.ts
git commit -m "test(e2e): cover video block end-to-end"
git push -u origin HEAD
gh pr create --title "feat(sale-pages): add video and divider blocks to editor" --body "$(cat <<'EOF'
## Summary
- Extend SalePageBlock type with video and divider variants
- Render new blocks in BlockEditor with appropriate form controls
- Render new blocks in all 3 sale page templates (blocks/simple/tracking)
- E2E test for video block creation → public render

## Test Plan
- [ ] tsc + lint pass
- [ ] go test passes
- [ ] E2E sale page video test passes
- [ ] Manual: create a sale page with video and divider, open public URL, verify both render
EOF
)"
```

---

## Self-Review Notes

- All 14 tasks have file path anchors with line numbers where applicable.
- All code blocks are complete, no `// TODO` or `// implement here` placeholders.
- Type names are consistent across tasks: `SalePageBlock`, `apiKeyCtxKey{}`, `RateLimitByAPIKey`, `HealthHandler`, `seedTestData`.
- Each phase ends with a PR creation step including title and body.
- Co-change rule for sale page templates (CLAUDE.md gotcha) is enforced in Task 13.
- Tests precede implementation for new code (Tasks 1's test was Health-only since the change was a deletion; Tasks 5 and 11 onward follow TDD).
