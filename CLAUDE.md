# CLAUDE.md

## Project Overview

Keep-PX — Facebook Pixel data preservation and replay platform. Captures pixel events via sale page templates, stores in DB, forwards to Facebook CAPI, and replays to new pixels when accounts get banned.

## Monorepo Structure

- **`backend/`** — Go REST API (chi, pgx/v5, Neon PostgreSQL)
- **`frontend/`** — React SPA dashboard (Vite, TanStack Query, Zustand, Tailwind CSS v4)

No shared workspace tooling — two independent packages.

## Development Loop

**FOLLOW THIS LOOP FOR EVERY TASK.** Do not skip steps. Loop back on failure.

### Step 1: Branch
Create feature branch from `main`. NEVER commit directly to `main`.
```
git checkout -b feat/... or fix/... or chore/... or refactor/...
```

### Step 2: Plan → Wait for user confirm
- Non-trivial tasks: ECC `/plan` → present plan, **wait for user approval**.
- Large tasks (backend + frontend together): use `TeamCreate` to spawn parallel agents.
- Use MCP `context7` to look up library docs if unsure about APIs.

### Step 3: Scaffold + Domain Context
**สร้างใหม่:**
| Need | Skill |
|------|-------|
| New backend resource | `/go-service-scaffold` |
| New endpoint on existing resource | `/api-endpoint` |
| Database schema change | `/db-migration` + MCP `neon` to verify |
| New frontend page | `/frontend-feature` |

**แก้ไขของเดิม — อ่าน domain skill ก่อนเขียน code:**
| Domain | Skill | ทำไม |
|--------|-------|-------|
| Sale pages, templates | `sale-page-editor` | 3 templates ต้อง co-change |
| Events, CAPI pipeline | `event-pipeline` | tracking + forwarding pitfalls |
| Auth, JWT, OAuth | `auth-flow` | token refresh + guard patterns |
| Billing, Stripe | `stripe-webhook` | idempotency + credit system |
| Nginx, CSP, proxy | `nginx-csp` | 4-location-block inheritance |

### Step 4: Implement
- **Backend (Go):** ECC `/go-test` — write tests first, then implement (TDD).
- **Frontend (React):** ECC `/tdd` — write tests first + MCP `context7` for library docs.
- **E2E tests:** Read `e2e-write` skill first (project-specific rules) → เขียน test ตาม rules → ECC `/e2e` เพื่อ **run** เท่านั้น → run `npm run e2e` local before push.
- **File co-change:** Check `dev-workflow` — แก้ interfaces.go ต้องแก้ mocks, แก้ types ต้องแก้ hooks.
- **Database:** Use MCP `neon` to query/inspect when needed.

### Step 5: Quality Gates → loop until green
Run only gates for packages you changed. **If fail, fix and re-run. Do NOT proceed.**

| Package | Command |
|---------|---------|
| Backend | `cd backend && go vet ./... && go test -race ./...` |
| Frontend | `cd frontend && npm run lint && npm run test && npm run build` |
| E2E | `cd frontend && npm run e2e` |

**ถ้า fail — ใช้ skill debug ก่อนแก้มั่ว:**
| Failure | Skill | ทำไม |
|---------|-------|-------|
| E2E test fail | `e2e-debug` | Root cause analysis + decision tree |
| Go build fail | ECC `/go-build` | Surgical fix, minimal changes |
| Frontend build/type fail | ECC `build-error-resolver` | Fix type errors, get build green |
| CI pipeline fail | `ci-pipeline` | CI structure + common patterns |

### Step 6: Code Review → loop until clean
Run **only** reviews relevant to changed code. **If issues found, fix and re-review.**

| Scope | Tool |
|-------|------|
| Go code | ECC `/go-review` |
| SQL, migrations, schema | ECC `/database-reviewer` |
| Auth, middleware, user input, API keys | ECC `/security-review` |
| Code quality, reuse, dead code | `/simplify` |

### Step 7: Commit + Push
- **Pre-push check:** Run `deploy-check` skill for deployment readiness.
- Commit format: `type: short description` (lowercase, no period, max 72 chars)
- Git hooks run automatically: `pre-commit` (lint-staged), `commit-msg` (commitlint), `pre-push` (quality gates).

### Step 8: Pull Request
```
gh pr create --title "type: description" --body "## Summary\n...\n## Test Plan\n..."
```
Wait for `ci-gate` to pass. **If CI fails, go back to Step 5.**

### Step 9: Verify Deploy
- Check `deploy-verify` job in GitHub Actions.
- `post-deploy-e2e` runs `@smoke` tests against production.
- If failed: use `railway-deploy` skill + MCP `railway` to check logs.
- CSP/proxy issues: use `nginx-csp` skill (4-location-block inheritance trap).
- Fix and go back to Step 5.

### Step 10: Report + Learn
- Tell the user: what was done, PR link, deploy status, any follow-up needed.
- Run ECC `/learn-eval` — extract reusable patterns from this session into instincts.

## Architecture

### Backend — Clean Architecture

`handler → service → repository → database`

| Layer | Path | Notes |
|-------|------|-------|
| Entry | `cmd/server/main.go` | Config, pgxpool, router, graceful shutdown |
| Config | `internal/config/` | `caarlos0/env`, loads `.env` via godotenv |
| Domain | `internal/domain/` | Pure structs (Customer, Pixel, PixelEvent, SalePage, ReplaySession, Notification, Subscription, Purchase, ReplayCredit) |
| Repository | `internal/repository/` | Interfaces in `interfaces.go`, pgx implementations in `postgres/` |
| Service | `internal/service/` | Business logic, sentinel errors for handler error mapping |
| Handler | `internal/handler/` | `go-playground/validator`, `handler.JSON()`/`handler.ErrorJSON()` |
| Middleware | `internal/middleware/` | JWT auth, API key auth, CORS, request ID, logging |
| Router | `internal/router/router.go` | chi router, all DI wiring |
| CAPI | `internal/facebook/capi.go` | Facebook Conversions API client |

### Key Patterns
- **Auth**: Google OAuth + JWT (dashboard, `middleware.GetCustomerID(ctx)`) + API Key (sale pages, `X-API-Key`)
- **Ownership**: Services check `pixel.CustomerID == customerID` before operations
- **CAPI**: Async forwarding via `go s.forwardToCAPI(...)`
- **Replay**: Background goroutine, semaphore (5 workers), ~50 events/sec rate limit
- **Replay credits**: `ConsumeReplayCredit` uses `SELECT ... FOR UPDATE SKIP LOCKED`. If downstream fails, caller must `RefundReplayCredit` (compensating action).
- **API error shape**: Backend `handler.ErrorJSON()` returns `{ "error": "message" }`. Frontend extracts via `axios.isAxiosError(err) && err.response?.data?.error`.
- **Billing**: Stripe checkout sessions → webhook → replay credits (pack system)
- **Admin**: `is_admin` flag on customers table, admin middleware, audit logging
- **Repository nil**: `GetByID` returns `(nil, nil)` when not found — callers check for nil

### Frontend
- **State**: Zustand (auth) + TanStack Query (server)
- **Routing**: react-router v7, `ProtectedRoute` wrapper, lazy-loaded admin routes
- **API**: Axios with auto token refresh + mutex (`src/lib/api.ts`), `@/` = `src/`
- **UI**: shadcn/ui in `src/components/ui/`, Vite proxy `/api` → `:8080`
- **Shared**: Reusable components in `src/components/shared/` — `StatCard`, `QueryErrorAlert`, `ProtectedRoute`, `AdminRoute`
- **Admin**: Separate admin pages under `/admin/*`, requires `is_admin` flag

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/google` | Public | Google OAuth login |
| POST | `/api/v1/auth/refresh` | Public | Token refresh |
| POST | `/api/v1/auth/logout` | JWT | Logout (revoke refresh token) |
| POST | `/api/v1/events/ingest` | API Key | Batch event ingestion |
| CRUD | `/api/v1/pixels/*` | JWT | Pixel management + test connection |
| GET | `/api/v1/events` | JWT | Event log (paginated, filterable by pixel_id, event_name, from, to) |
| GET | `/api/v1/events/event-types` | JWT | Distinct event names for customer |
| GET | `/api/v1/events/recent` | JWT | Realtime events (polling) |
| GET | `/api/v1/events/{id}` | JWT | Event detail with ownership check |
| CRUD | `/api/v1/sale-pages/*` | JWT | Sale page CRUD + pixel assignment |
| GET | `/p/:slug` | Public | Public sale page rendering |
| POST/GET | `/api/v1/replays/*` | JWT | Replay sessions |
| GET | `/api/v1/analytics/*` | JWT | Dashboard analytics |
| POST | `/api/v1/billing/checkout` | JWT | Stripe checkout session |
| GET | `/api/v1/billing/status` | JWT | Billing status + credits |
| POST | `/api/v1/billing/webhook` | Stripe sig | Stripe webhook handler |
| GET | `/api/v1/notifications` | JWT | User notifications |
| GET/PUT | `/api/v1/settings/*` | JWT | User settings + API key |
| GET/PUT/POST | `/api/v1/admin/*` | JWT+Admin | Admin panel endpoints |

## Reference

- **Database**: PostgreSQL on Neon. 16 active tables: `customers`, `pixels`, `pixel_events`, `event_rules`, `replay_sessions`, `refresh_tokens`, `sale_pages`, `sale_page_pixels`, `notifications`, `purchases`, `replay_credits`, `subscriptions`, `event_usage`, `stripe_webhook_events`, `admin_credit_grants`, `admin_audit_logs`. UUIDs, `TIMESTAMPTZ`. 24 migrations.
- **Deploy**: Railway — Backend (Go/Alpine Dockerfile) + Frontend (Node→Nginx Dockerfile, needs `VITE_API_URL` build arg).
- **Backend env** (`backend/.env.example`): `DATABASE_URL`, `JWT_SECRET` (required), `PORT`, `ENV`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `FB_GRAPH_API_URL`, `CORS_ALLOWED_ORIGINS`, `RATE_LIMIT_RPS`
- **Frontend env**: `VITE_API_URL` (empty = Vite proxy)
- **sqlc**: `cd backend && sqlc generate` — NEVER edit `db/generated/` manually.
- **Migrations**: Auto-run on deploy via `golang-migrate` in `cmd/server/main.go` (`m.Up()` at startup). Migration files in `backend/db/migrations/` are included in the Docker image. No manual step needed.

## Gotchas

- **No `components.json`**: `npx shadcn add` won't work. Write shadcn components manually + `npm install @radix-ui/*` in `frontend/`.
- **Custom Popover**: `components/ui/popover.tsx` is NOT Radix — it's a custom implementation. No `asChild` prop. Use `className` directly on `PopoverTrigger`.
- **Mock files exist in TWO packages**: When changing a repository interface, update mocks in BOTH `service/mocks_test.go` AND `handler/testhelpers_test.go`.
- **Recharts Tooltip name collision**: When using shadcn Tooltip alongside Recharts, alias as `Tooltip as RechartsTooltip` and `Tooltip as ShadTooltip`.
- **Handler perPage clamp**: Always clamp `perPage` in handler (not just service) — handler uses it for `TotalPages` calculation.
- **chi route order**: Static routes (`/events/event-types`) MUST be registered before wildcard (`/events/{id}`).
- **LSP diagnostics can be stale**: After editing `.tsx` files, LSP may show false errors. Verify with `cd frontend && npx tsc --noEmit` before investigating.
- **E2E Thai text collisions**: `getByText` + Thai text ต้องใช้ `{ exact: true }` หรือ scope ด้วย parent locator — "สร้าง", "ยกเลิก", "จัดการ" ปรากฏใน 12-16 components
- **E2E responsive duplicates**: Sidebar/nav ซ้ำ mobile/desktop → ใช้ `.first()` หรือ scope locator, ห้ามใช้ bare `getByRole` ที่ match หลาย element
- **E2E sandbox empty state**: Test user อาจไม่มี data → ใช้ `test.skip()` + guard check ก่อน interact กับ empty lists
