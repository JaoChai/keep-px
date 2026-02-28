# CLAUDE.md

## Project Overview

Keep-PX — Facebook Pixel data preservation and replay platform. Captures pixel events via sale page templates, stores in DB, forwards to Facebook CAPI, and replays to new pixels when accounts get banned.

## Monorepo Structure

- **`backend/`** — Go REST API (chi, pgx/v5, Neon PostgreSQL)
- **`frontend/`** — React SPA dashboard (Vite, TanStack Query, Zustand, Tailwind CSS v4)

No shared workspace tooling — two independent packages.

## Commands

```bash
# Backend
cd backend && go run ./cmd/server              # dev server :8080
cd backend && go build -o server ./cmd/server   # build
cd backend && go vet ./... && go test -race ./... # quality gate

# Frontend
cd frontend && npm run dev      # dev server :5173 (proxies /api → :8080)
cd frontend && npm run build    # tsc + vite build
cd frontend && npm run lint     # eslint

# Database
cd backend && sqlc generate     # regenerate db/generated/ — NEVER edit generated files
# Migrations: run backend/db/migrations/*.up.sql manually against Neon
```

## Development Workflow

### Branch Rules
- NEVER commit directly to `main`. Always create a feature branch (`feat/`, `fix/`, `chore/`, `refactor/`).
- One logical change per branch. Squash merge only.
- Commit format: `type: short description` (lowercase, no period, max 72 chars)

### Before Writing Code
- Non-trivial tasks: use plan mode first.
- New backend resources: `/go-service-scaffold`
- New endpoints on existing resources: `/api-endpoint`
- Database schema changes: `/db-migration`
- New frontend pages: `/frontend-feature`
- New/changed Go logic: `/go-test` (write tests first, then implement)

### Quality Gates (MUST Pass Before Push)
Run only gates for packages you changed. Do NOT push code that fails.

| Package | Command |
|---------|---------|
| Backend | `cd backend && go vet ./... && go test -race ./...` |
| Frontend | `cd frontend && npm run lint && npm run test && npm run build` |
| E2E | `cd frontend && npm run e2e` |

Git hooks run automatically: `pre-commit` (lint-staged), `commit-msg` (commitlint), `pre-push` (quality gates).

### Code Review (MUST Run Before Creating PR)
- Backend Go changes: `/go-review`
- SQL, migrations, or schema changes: `/database-reviewer`
- Auth, middleware, user input, API keys: `/security-review`

### Pull Requests
- Create via `gh pr create` with conventional commit title.
- Include Summary and Test Plan in PR body.
- Wait for `ci-gate` to pass before merging.

### After Merge
- Check `deploy-verify` job in GitHub Actions.
- `post-deploy-e2e` runs `@smoke` tests against production.
- If failed, check Railway logs immediately.

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
- **Auth**: JWT (dashboard, Bearer token, `middleware.GetCustomerID(ctx)`) + API Key (sale pages, `X-API-Key` header)
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

## Database

PostgreSQL on Neon. Tables: `customers`, `pixels`, `pixel_events`, `replay_sessions`, `refresh_tokens`. All IDs UUID (`gen_random_uuid()`), timestamps `TIMESTAMPTZ`.

## Deployment

Railway — Backend (`backend/Dockerfile`, multi-stage Go/Alpine) + Frontend (`frontend/Dockerfile`, Node build → Nginx, needs `VITE_API_URL` build arg).

## Environment Variables

Backend (`backend/.env.example`): `DATABASE_URL`, `JWT_SECRET` (required), `PORT`, `ENV`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `FB_GRAPH_API_URL`, `CORS_ALLOWED_ORIGINS`, `RATE_LIMIT_RPS`

Frontend: `VITE_API_URL` (empty = relative path with Vite proxy)
