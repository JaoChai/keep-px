---
name: dev-workflow
description: Keep-PX development workflow patterns extracted from 120 commits — file co-change maps, recurring pitfalls, and iteration shortcuts
version: 1.1.0
source: local-git-analysis
analyzed_commits: 200
---

# Keep-PX Development Workflow Patterns

## Commit Conventions

92% of commits follow **conventional commits** format:

```
type: short description (lowercase, no period, max 72 chars)
```

| Type | Usage | Count |
|------|-------|-------|
| `fix:` | Bug fixes, CSP fixes, proxy fixes | 50 (46%) |
| `feat:` | New features, new pages, new endpoints | 44 (40%) |
| `chore:` | Config, env vars, cleanup | 7 (6%) |
| `refactor:` | Code restructuring without behavior change | 6 (5%) |
| `ci:` | GitHub Actions, hooks | 2 (2%) |

Branch naming: `feat/`, `fix/`, `chore/`, `refactor/`, `perf/`

## File Co-Change Map

When changing one file, you almost always need to change these others:

### Backend: Adding/Modifying a Feature

```
1. backend/internal/domain/*.go          → Add/modify struct
2. backend/internal/repository/interfaces.go → Add/modify interface method
3. backend/internal/repository/postgres/*.go → Implement query
4. backend/internal/service/*_service.go  → Business logic
5. backend/internal/service/mocks_test.go → Update mock (ALWAYS!)
6. backend/internal/service/*_test.go     → Update/add tests
7. backend/internal/handler/*_handler.go  → HTTP handler
8. backend/internal/router/router.go      → Wire route + DI
9. backend/cmd/server/main.go             → Wire new dependencies (if new service)
```

**router.go** changes 27/120 commits — it's the DI wiring hub.
**interfaces.go** changes 21/120 — every new repo method touches it.
**mocks_test.go** changes 18/120 — must update when interfaces change.

### Frontend: Adding/Modifying a Page

```
1. frontend/src/types/index.ts             → Add/modify TypeScript types
2. frontend/src/hooks/use-*.ts             → Add TanStack Query hook
3. frontend/src/pages/*Page.tsx            → Page component
4. frontend/src/router.tsx                 → Add route
5. frontend/src/components/layout/Sidebar.tsx → Add nav entry
```

**types/index.ts** changes 19/120 — every API change touches it.
**router.tsx** changes 13/120 — every new page.
**Sidebar.tsx** changes 12/120 — every new page.

### Full-Stack Feature (both sides)

All backend files above + all frontend files above + often:
- `backend/db/migrations/*.sql` — schema changes
- `frontend/src/lib/api.ts` — rarely (Axios instance), but check if new headers needed

## Recurring Pitfalls (Learned from Git History)

### 1. Nginx CSP Headers (6+ fix rounds)

**Pattern:** Every new external service (Google OAuth, Stripe, R2 CDN) needs CSP updates in **6 places** in `nginx.conf`:
- Server-level (line ~25)
- `/assets/` location
- `/` location
- `/p/` location (separate sale page CSP)

**Pitfall:** Nginx drops parent `add_header` when child location has its own `add_header`. You must redeclare ALL headers in every location block.

**Checklist for new external domains:**
- [ ] `script-src` — if loading JS
- [ ] `style-src` — if loading CSS
- [ ] `connect-src` — if making API calls
- [ ] `frame-src` — if embedding iframes
- [ ] `img-src` — if loading images
- [ ] All 4 location blocks updated

### 2. Nginx Proxy Configuration (5+ fix rounds)

**Pattern:** `/p/` and `/api/` proxy to backend. Common mistakes:

| Mistake | Symptom | Fix |
|---------|---------|-----|
| `Host $host` instead of `$proxy_host` | 504 routing loop | Use `$proxy_host` |
| Default buffer size | 502 "too big header" | Add `proxy_buffer_size 16k` |
| Missing proxy location | 404 on API calls | Add `location /api/` block |
| Private networking | 504 timeout | Services must be same region |

### 3. Backend Interface → Mock Sync

**Pattern:** When you add a method to `interfaces.go`, you MUST update `mocks_test.go` or tests break.

```
Change interfaces.go → Update mocks_test.go → Run tests
```

### 4. Railway Deploy Environment Variables

**Pattern:** New env vars need to be set in Railway AND added to `.env.example`.

**Common miss:** Frontend `VITE_*` vars are build-time ARGs in Dockerfile, not runtime env vars. Must be set as Railway build args.

### 5. Database Migration Idempotency

**Pattern:** All migrations must use `IF NOT EXISTS` / `IF EXISTS` because they may run multiple times (server restarts, CI).

```sql
-- GOOD
CREATE TABLE IF NOT EXISTS ...
CREATE INDEX IF NOT EXISTS ...
ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...

-- BAD (will crash on re-run)
CREATE TABLE ...
```

### 6. Stripe Webhook → Event Type Matching

**Pattern:** Stripe API version mismatches cause webhook handler to reject events. Always use `stripe.Event` parsing that tolerates version differences.

## Hot Files (Change Frequency)

Files that change most often — review these carefully:

| File | Changes | Role |
|------|---------|------|
| `router.go` | 27 | DI wiring, route registration |
| `interfaces.go` | 21 | Repository contracts |
| `types/index.ts` | 19 | Frontend type definitions |
| `nginx.conf` | 19 | Proxy + CSP headers |
| `mocks_test.go` | 18 | Test mocks |
| `router.tsx` | 13 | Frontend routing |
| `Sidebar.tsx` | 12 | Navigation menu |
| `SalePageEditorPage.tsx` | 12 | Core feature page |
| `replay_service.go` | 12 | Complex async logic |
| `event_service.go` | 12 | Event pipeline |

## Iteration Patterns

### Quick Fix Cycle (most common)
```
1. Identify bug → 2. Fix 1-2 files → 3. Push → 4. CI → 5. Merge
Average: 1 commit per PR
```

### Feature Cycle
```
1. Branch → 2. Plan → 3. Backend (domain→repo→service→handler→router)
→ 4. Frontend (types→hook→page→router→sidebar)
→ 5. Tests → 6. Push → 7. CI → 8. Review → 9. Merge
Average: 1-3 commits per PR
```

### Deploy Fix Cycle (nginx/CSP/proxy)
```
1. Deploy → 2. Check production → 3. Find error in Railway logs
→ 4. Fix nginx.conf → 5. Push → 6. Merge → 7. Wait deploy → 8. Verify
Often requires 2-3 rounds
```

## Quality Gates

| Package | Command | When |
|---------|---------|------|
| Backend | `cd backend && go vet ./... && go test -race ./...` | Any `.go` change |
| Frontend | `cd frontend && npm run lint && npm run test && npm run build` | Any `.ts/.tsx` change |
| E2E | `cd frontend && npm run e2e` | UI/routing/auth changes |

Git hooks run automatically: `pre-commit` (lint-staged), `commit-msg` (commitlint), `pre-push` (quality gates for changed packages only).
