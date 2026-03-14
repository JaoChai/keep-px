---
name: deploy-check
description: Run pre-deployment verification checklist across backend and frontend packages before pushing to Railway
---

# Deploy Check

## When to Activate

Activate this skill when the user says:
- "Ready to deploy?"
- "Pre-deployment check"
- "Check before push"
- "Can I deploy?"
- "Verify everything works"
- "Run deploy checks"

## Step-by-Step Workflow

Run each step sequentially. Stop on first failure and report the issue.

### Step 1: Backend Build & Test

```bash
cd backend && go build ./cmd/server
```

```bash
cd backend && go vet ./...
```

```bash
cd backend && go test ./...
```

All three must pass. Report which command failed and the error output.

### Step 2: Frontend Build

```bash
cd frontend && npm run build
```

This runs `tsc` + `vite build`. Must exit 0 with no TypeScript errors.

### Step 3: Security Scan

Check for common security issues:

1. **No .env files committed:**
```bash
git ls-files | grep -E '\.env$|\.env\.' | grep -v '.env.example'
```
Must return empty.

2. **No hardcoded secrets:**
```bash
grep -rn --include='*.go' --include='*.ts' --include='*.tsx' --include='*.js' \
  -E '(password|secret|token|api_key)\s*[:=]\s*["\x27][^"\x27]{8,}' \
  backend/internal/ frontend/src/ || true
```
Review any matches — false positives are OK (like variable names), but actual hardcoded credentials must be flagged.

3. **No TODO security issues:**
```bash
grep -rn --include='*.go' --include='*.ts' --include='*.tsx' \
  -i 'TODO.*secur\|FIXME.*auth\|HACK.*token' \
  backend/ frontend/src/ || true
```

### Step 4: Deployment Files

Verify Railway configuration files exist:

```bash
ls backend/railway.json frontend/railway.json
```

Both files must exist.

Check that `backend/Dockerfile` and `frontend/Dockerfile` exist:

```bash
ls backend/Dockerfile frontend/Dockerfile
```

### Step 5: Output Summary

Print a summary table:

```
Pre-Deployment Check Results
============================
[PASS/FAIL] Backend build     (go build ./cmd/server)
[PASS/FAIL] Backend lint      (go vet ./...)
[PASS/FAIL] Backend tests     (go test ./...)
[PASS/FAIL] Frontend build    (npm run build)
[PASS/FAIL] Security scan     (no secrets, no .env)
[PASS/FAIL] Deploy configs    (railway.json + Dockerfiles)
============================
Result: READY TO DEPLOY / BLOCKED — fix issues above
```

## Important Notes

- This skill does NOT deploy. It only verifies readiness.
- Always run from the project root (`/Users/jaochai/Code/keep-px/`).
- If backend tests need a database (integration tests), note that they may fail without `DATABASE_URL`. Unit tests (service layer) should always pass.
- Frontend build requires `node_modules` — run `npm install` first if needed.

## Related

- Railway MCP tools for actual deployment (`deploy`, `get-logs`)
- Built-in `security-review` skill for deeper security analysis
