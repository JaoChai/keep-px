---
name: nginx-csp
description: Nginx CSP header management, security headers, and envsubst variable checklist for Keep-PX frontend deployment
---

# Nginx CSP

## When to Activate

Activate this skill when the user says:
- "CSP error" / "Content Security Policy" / "blocked by CSP"
- "Add external script/style/font/image" / "Add Google/Stripe/Facebook domain"
- "Nginx config" / "Security headers"
- "Image not loading" / "Script blocked" / "Font not loading"
- "Google login broken in production" / "Stripe not loading"
- "envsubst" / "nginx won't start" / "undefined variable in nginx"

## Architecture

Keep-PX nginx.conf has **4 location blocks** with separate CSP policies:

```
server {
    # Global security headers (COOP, X-Frame-Options, etc.)

    location /assets/ {
        # Dashboard CSP (same as root)
    }

    location /p/ {
        # Sale Pages CSP (more permissive img-src)
        proxy_pass to backend
    }

    location / {
        # Dashboard CSP
        try_files SPA fallback
    }
}
```

**Critical**: Nginx does NOT inherit `add_header` into nested locations. Each location must redeclare ALL headers.

## Current CSP Directives (Dashboard)

```nginx
add_header Content-Security-Policy "
  default-src 'self';
  script-src 'self' 'unsafe-inline' js.stripe.com connect.facebook.net accounts.google.com;
  style-src 'self' 'unsafe-inline' fonts.googleapis.com accounts.google.com;
  connect-src 'self' ${BACKEND_URL} accounts.google.com js.stripe.com;
  frame-src js.stripe.com accounts.google.com;
  img-src 'self' data: *.googleusercontent.com *.r2.dev;
  font-src 'self' data: fonts.gstatic.com;
" always;
```

## Current CSP Directives (Sale Pages)

```nginx
add_header Content-Security-Policy "
  default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline' fonts.googleapis.com;
  connect-src 'self';
  img-src * data:;
  font-src 'self' data: fonts.gstatic.com;
" always;
```

## Domain-to-Directive Reference

When adding an external service, add its domains to the correct CSP directives:

### Google Sign-In (accounts.google.com)
| Directive | Domain | Why |
|-----------|--------|-----|
| script-src | accounts.google.com | GSI JavaScript library |
| style-src | accounts.google.com | GSI injects stylesheet |
| connect-src | accounts.google.com | Token exchange XHR |
| frame-src | accounts.google.com | Popup window/iframe |

### Google Fonts
| Directive | Domain | Why |
|-----------|--------|-----|
| style-src | fonts.googleapis.com | CSS font-face declarations |
| font-src | fonts.gstatic.com | Actual font files (.woff2) |

### Stripe
| Directive | Domain | Why |
|-----------|--------|-----|
| script-src | js.stripe.com | Stripe.js SDK |
| connect-src | js.stripe.com | Payment API calls |
| frame-src | js.stripe.com | Checkout iframe |

### Facebook Pixel
| Directive | Domain | Why |
|-----------|--------|-----|
| script-src | connect.facebook.net | fbevents.js SDK |

### Cloudflare R2 (Image Storage)
| Directive | Domain | Why |
|-----------|--------|-----|
| img-src | *.r2.dev | Uploaded images from R2 bucket |

### Backend API
| Directive | Domain | Why |
|-----------|--------|-----|
| connect-src | ${BACKEND_URL} | API calls from different subdomain in production |

## Checklist: Adding External Resource

Before adding any external script, stylesheet, font, image, or API call:

- [ ] Identify ALL resource types the service loads (script, style, font, image, XHR, iframe)
- [ ] Add each domain to the correct CSP directive
- [ ] Update ALL 3 dashboard CSP locations (server-level /assets/, root /)
- [ ] If sale pages need it too, update the /p/ location CSP separately
- [ ] If using envsubst variable, verify it exists in Railway environment
- [ ] Test in production URL (not localhost — 'self' resolves differently)
- [ ] Check browser DevTools Console for CSP violation errors

## Checklist: Security Headers Sync

When modifying any `add_header` in nginx.conf, update ALL locations:

- [ ] Server-level block (before first location)
- [ ] `location /assets/` block
- [ ] `location /` block (root/SPA)
- [ ] `location /p/` block (sale pages — may have different CSP)

Headers that MUST be in all locations:
```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header Cross-Origin-Opener-Policy "same-origin-allow-popups" always;
```

**COOP Header**: `same-origin-allow-popups` is REQUIRED for Google Sign-In popup to postMessage back to the parent window. Do NOT change this to `same-origin`.

## Environment Variables

| Variable | Used In | Set By |
|----------|---------|--------|
| `${BACKEND_URL}` | connect-src | Railway env var (required) |

**Rules:**
- Only use envsubst variables that are GUARANTEED to exist in Railway
- Prefer wildcards (`*.r2.dev`) over variables for static domain lists
- Never add `${VAR}` to nginx.conf without verifying it's in Railway environment
- Undefined variables cause nginx startup failure (envsubst leaves literal `${VAR}`)

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| Missing domain in one directive | Script loads but XHR blocked | Check ALL resource types the service uses |
| Header not in nested location | Works on `/` but not `/assets/` | Redeclare headers in every location block |
| Undefined envsubst variable | Nginx container crash loop | Only use vars defined in Railway env |
| COOP header missing | Google Sign-In popup fails silently | Add `same-origin-allow-popups` to all locations |
| 'self' mismatch in production | API calls blocked | Add `${BACKEND_URL}` to connect-src (different subdomain) |
| Wildcard in wrong directive | Security gap | Use wildcards only in img-src, never in script-src |
| Build-time vs runtime var | VITE_* not resolved | VITE_* = Docker build ARGs, BACKEND_URL = runtime envsubst |
| R2 CDN wildcard scope | Images blocked elsewhere | `*.r2.dev` only valid in `img-src`, not `connect-src` |
| COOP `same-origin` | Google Sign-In popup broken | Must be `same-origin-allow-popups` (Google needs postMessage) |

## Sale Page CSP Note

The `/p/` location intentionally does NOT include `X-Frame-Options: DENY` — this allows sale pages to be embedded in iframes on customer sites. The sale page CSP is more permissive on `img-src` (allows `*`) to support user-uploaded images from any source.

## Verification Tips

After deploy, check **Response** headers (not Request headers) in browser DevTools Network tab. Common mistake: checking Request headers which don't show server-set CSP.

## Diagnostic: CSP Violation

1. Open browser DevTools → Console tab
2. Look for `Refused to load/execute...violates CSP directive`
3. The error tells you exactly which directive and domain are missing
4. Add the domain to the correct directive in ALL location blocks
5. Rebuild and redeploy frontend

## Verification

After modifying nginx.conf:
```bash
# Check syntax locally (if nginx installed)
nginx -t -c frontend/nginx.conf

# Check no undefined envsubst variables
grep -oP '\$\{[^}]+\}' frontend/nginx.conf | sort -u
# Every variable listed must exist in Railway env

# After deploy, check headers
curl -sI https://your-domain.com | grep -i "content-security-policy\|cross-origin"
```

## Related

- `railway-deploy` for nginx proxy patterns and deployment
- `deploy-check` for pre-deployment verification
- Built-in `security-review` for comprehensive security analysis
