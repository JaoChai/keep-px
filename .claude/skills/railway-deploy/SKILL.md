---
name: railway-deploy
description: Railway deployment patterns, Nginx proxy rules, and common pitfalls for Keep-PX monorepo
---

# Railway Deploy

## When to Activate

Activate this skill when the user says:
- "Deploy to Railway" / "Fix deployment"
- "Nginx config" / "Proxy not working" / "502 error"
- "Dockerfile" / "Docker build" / "Build fails"
- "Frontend blank page" / "API routes not working in production"
- "Host header" / "Proxy loop"

## Architecture

Keep-PX runs as 2 separate Railway services:

```
Railway Project
├── Backend service  → backend/Dockerfile (Go binary, port 8080)
└── Frontend service → frontend/Dockerfile (Nginx, port 80)
    └── nginx.conf proxies /api/ and /p/ → backend internal URL
```

## Common Pitfalls

| Pitfall | Root Cause | Fix |
|---------|-----------|-----|
| Proxy loop (502/infinite redirect) | `Host` header override in nginx.conf | NEVER set `proxy_set_header Host` to backend |
| SDK events don't reach backend | Missing `/api/` proxy in nginx.conf | Add `location /api/ { proxy_pass $BACKEND_URL; }` |
| Sale pages 404 | Missing `/p/` route proxy | Add `location /p/ { proxy_pass $BACKEND_URL; }` |
| Build fails "no Go files" | railway.json buildCommand in wrong dir | Use per-service Dockerfiles, not railway.json |
| Frontend shows blank page | `VITE_API_URL` not set during build | Pass as Docker build arg |

## Nginx Configuration

```nginx
# Proxy API routes to backend
location /api/ {
    proxy_pass http://$BACKEND_INTERNAL_URL;
    # DO NOT add: proxy_set_header Host $BACKEND_HOST;
    # Railway internal networking handles Host headers automatically
}

# Proxy sale page routes
location /p/ {
    proxy_pass http://$BACKEND_INTERNAL_URL;
}

# Serve frontend SPA with fallback
location / {
    try_files $uri $uri/ /index.html;
}
```

**Critical rule**: NEVER override the `Host` header when proxying to backend on Railway. This causes infinite redirect loops. Railway's internal networking handles Host headers automatically.

## Deployment Checklist

### Backend Dockerfile
- Multi-stage build: `golang:alpine` → `alpine`
- Include `ca-certificates` in final stage (needed for HTTPS to Neon DB and Meta API)
- Expose port 8080
- Copy migrations if auto-run on startup

### Frontend Dockerfile
- Stage 1: `node:alpine` for `npm run build` with `--build-arg VITE_API_URL`
- Stage 2: `nginx:alpine` to serve static files
- Use `envsubst` for runtime environment variables in nginx.conf
- Expose port 80

### Environment Variables
**Backend service:**
- `DATABASE_URL` — Neon PostgreSQL connection string
- `JWT_SECRET` — HMAC signing key
- `PORT=8080`
- `CORS_ALLOWED_ORIGINS` — Frontend domain

**Frontend service:**
- `VITE_API_URL` — Backend public URL (build-time)
- `BACKEND_URL` — Backend internal Railway URL (runtime, for nginx proxy)

## Adding New Backend Routes

When adding a new route that must be accessible from the frontend in production, remember to add the route prefix to nginx.conf:

```nginx
# Example: adding /webhooks/ route
location /webhooks/ {
    proxy_pass http://$BACKEND_INTERNAL_URL;
}
```

Without this, the route works in dev (Vite proxy) but 404s in production.

## Verification

```bash
# After deploy, check:
curl -s https://your-app.railway.app/api/v1/auth/login  # Should reach backend
curl -s https://your-app.railway.app/p/test-slug         # Should reach backend
curl -s https://your-app.railway.app/                     # Should serve frontend
```

## Related

- `deploy-check` for pre-deployment quality gates
- `event-pipeline` for verifying SDK events reach backend after deploy
