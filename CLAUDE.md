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
- Non-trivial tasks: enter plan mode, present plan, **wait for user approval**.
- Large tasks (backend + frontend together): use `TeamCreate` to spawn parallel agents.
- Use MCP `context7` to look up library docs if unsure about APIs.

### Step 3: Scaffold (if needed)
| Need | Skill |
|------|-------|
| New backend resource | `/go-service-scaffold` |
| New endpoint on existing resource | `/api-endpoint` |
| Database schema change | `/db-migration` + MCP `neon` to verify |
| New frontend page | `/frontend-feature` |

### Step 4: Implement
- Go logic: `/go-test` — **write tests first**, then implement (TDD).
- Use MCP `context7` for up-to-date library documentation.
- Use MCP `neon` to query/inspect database when needed.

### Step 5: Quality Gates → loop until green
Run only gates for packages you changed. **If fail, fix and re-run. Do NOT proceed.**

| Package | Command |
|---------|---------|
| Backend | `cd backend && go vet ./... && go test -race ./...` |
| Frontend | `cd frontend && npm run lint && npm run test && npm run build` |
| E2E | `cd frontend && npm run e2e` |

### Step 6: Code Review → loop until clean
Run applicable reviews. **If issues found, fix and re-review.**

| Scope | Tool |
|-------|------|
| Go code | `/go-review` |
| SQL, migrations, schema | `/database-reviewer` |
| Auth, middleware, user input, API keys | `/security-review` |

### Step 7: Commit + Push
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
- If failed: use MCP `railway` to check logs, fix, go back to Step 5.

### Step 10: Report
Tell the user: what was done, PR link, deploy status, any follow-up needed.

## Architecture

### Backend — Clean Architecture

`handler → service → repository → database`

| Layer | Path | Notes |
|-------|------|-------|
| Entry | `cmd/server/main.go` | Config, pgxpool, router, graceful shutdown |
| Config | `internal/config/` | `caarlos0/env`, loads `.env` via godotenv |
| Domain | `internal/domain/` | Pure structs (Customer, Pixel, PixelEvent, ReplaySession) |
| Repository | `internal/repository/` | Interfaces in `interfaces.go`, pgx implementations in `postgres/` |
| Service | `internal/service/` | Business logic, sentinel errors for handler error mapping |
| Handler | `internal/handler/` | `go-playground/validator`, `handler.JSON()`/`handler.ErrorJSON()` |
| Middleware | `internal/middleware/` | JWT auth, API key auth, CORS, request ID, logging |
| Router | `internal/router/router.go` | chi router, all DI wiring |
| CAPI | `internal/facebook/capi.go` | Facebook Conversions API client |

### Key Patterns
- **Auth**: JWT (dashboard, `middleware.GetCustomerID(ctx)`) + API Key (sale pages, `X-API-Key`)
- **Ownership**: Services check `pixel.CustomerID == customerID` before operations
- **CAPI**: Async forwarding via `go s.forwardToCAPI(...)`
- **Replay**: Background goroutine, semaphore (5 workers), ~50 events/sec rate limit
- **Repository nil**: `GetByID` returns `(nil, nil)` when not found — callers check for nil

### Frontend
- **State**: Zustand (auth) + TanStack Query (server)
- **Routing**: react-router v7, `ProtectedRoute` wrapper
- **API**: Axios with auto token refresh (`src/lib/api.ts`), `@/` = `src/`
- **UI**: shadcn/ui in `src/components/ui/`, Vite proxy `/api` → `:8080`

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register\|login\|refresh` | Public | Auth endpoints |
| POST | `/api/v1/events/ingest` | API Key | Batch event ingestion |
| CRUD | `/api/v1/pixels/*` | JWT | Pixel management |
| GET | `/api/v1/events` | JWT | Event log (paginated) |
| POST/GET | `/api/v1/replays/*` | JWT | Replay sessions |
| GET | `/api/v1/analytics/*` | JWT | Dashboard analytics |

## Reference

- **Database**: PostgreSQL on Neon. Tables: `customers`, `pixels`, `pixel_events`, `replay_sessions`, `refresh_tokens`. UUIDs, `TIMESTAMPTZ`.
- **Deploy**: Railway — Backend (Go/Alpine Dockerfile) + Frontend (Node→Nginx Dockerfile, needs `VITE_API_URL` build arg).
- **Backend env** (`backend/.env.example`): `DATABASE_URL`, `JWT_SECRET` (required), `PORT`, `ENV`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `FB_GRAPH_API_URL`, `CORS_ALLOWED_ORIGINS`, `RATE_LIMIT_RPS`
- **Frontend env**: `VITE_API_URL` (empty = Vite proxy)
- **sqlc**: `cd backend && sqlc generate` — NEVER edit `db/generated/` manually.
- **Migrations**: Auto-run on deploy via `golang-migrate` in `cmd/server/main.go` (`m.Up()` at startup). Migration files in `backend/db/migrations/` are included in the Docker image. No manual step needed.
