---
name: auth-flow
description: Google OAuth + JWT authentication, token refresh, ProtectedRoute guard — ใช้เมื่อ: login ไม่ได้, ล็อกอินพัง, token หมดอายุ, 401 error, auth error, หน้าจอกระพริบ, session expired, ล็อกอินไม่ได้, sign in พัง, refresh token
---

# Auth Flow

## When to Activate

Activate this skill when the user says:
- "Login broken" / "Auth not working" / "Can't sign in"
- "Token refresh" / "401 error" / "Session expired"
- "Flash of login page" / "Stuck on loading" / "White screen on auth"
- "Google Sign-In popup" / "COOP header" / "postMessage blocked"
- "StrictMode auth issue" / "Double mount" / "useEffect runs twice"
- "Add auth to [feature]" / "Protect route"
- "Zustand persist" / "localStorage tokens"

## Architecture

```
Google Popup → credential (ID token)
    ↓
Frontend: POST /auth/google {id_token}
    ↓
Backend: Validate with google.golang.org/api/idtoken
    ↓ Find or create customer (3 paths)
    ↓ Generate JWT access token + random refresh token
    ↓
Frontend: Store tokens in localStorage + customer in Zustand
    ↓
ProtectedRoute: Verify once per session via GET /auth/me
    ↓
On 401: Token refresh interceptor → POST /auth/refresh
```

## Backend Auth Flow

**File:** `backend/internal/service/auth_service.go`

### Google Auth (3 resolution paths)

```go
func (s *AuthService) GoogleAuth(ctx context.Context, input GoogleAuthInput) (*AuthTokens, error) {
    // 1. Validate Google ID token
    payload, err := idtoken.Validate(ctx, input.IDToken, s.googleClientID)
    // Extract: sub (Google ID), email, name, email_verified

    // 2. Try to find existing user (3 paths in order):
    //    a. GetByGoogleID(sub) → already linked → issue tokens
    //    b. GetByEmail(email) → exists → link Google ID → issue tokens
    //    c. Create new customer with Google ID (no password)

    // 3. Generate tokens
    return s.generateTokens(ctx, customer)
}
```

### Token Generation

```go
func (s *AuthService) generateTokens(ctx context.Context, customer *domain.Customer) (*AuthTokens, error) {
    // Access token: HS256 JWT, 15min TTL
    claims := jwt.MapClaims{
        "sub":   customer.ID,
        "email": customer.Email,
        "iat":   time.Now().Unix(),
        "exp":   time.Now().Add(s.accessTTL).Unix(),
    }
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    // Refresh token: random 32 bytes, hex encoded, stored hashed in DB
    refreshBytes := make([]byte, 32)
    rand.Read(refreshBytes)
    refreshToken := hex.EncodeToString(refreshBytes)
    // Store SHA-256 hash in refresh_tokens table (7d TTL)
}
```

### JWT Middleware

**File:** `backend/internal/middleware/auth.go`

```go
// Extracts and validates JWT from Authorization: Bearer header
// Enforces HS256 algorithm (prevents algo switching attack)
// Sets customerID in context via context.WithValue()
// All protected endpoints use: middleware.GetCustomerID(r.Context())
```

## Frontend Auth Flow

### Login Page

**File:** `frontend/src/pages/auth/LoginPage.tsx`

Uses `@react-oauth/google` library. Google popup returns credential → mutation sends to backend.

### Auth Store (Zustand + Persist)

**File:** `frontend/src/stores/auth-store.ts`

```typescript
// Persisted to localStorage:
// - customer: Customer | null
// - isAuthenticated: boolean

// NOT persisted (sensitive):
// - api_key (stays in memory only)

// Hydration tracking:
// - _hasHydrated: boolean (set true when persist loads from localStorage)
```

**Rules:**
- Use `partialize` to exclude sensitive fields from localStorage
- Check `_hasHydrated` before making auth decisions
- Tokens stored separately in localStorage (not in Zustand)

### Token Refresh Interceptor

**File:** `frontend/src/lib/api.ts`

```typescript
// Intercepts 401 responses (except auth endpoints)
// Uses mutex flag + promise queue to prevent concurrent refresh:
//
// Request A fails 401 → isRefreshing = true → POST /auth/refresh
// Request B fails 401 → sees isRefreshing → queued in refreshSubscribers[]
// Request C fails 401 → sees isRefreshing → queued in refreshSubscribers[]
// Refresh completes → new token → resolve all queued requests
// isRefreshing = false
//
// Key: Only ONE refresh request, all others wait in queue
```

**Pitfall prevention:**
- `_retry` flag on request config prevents infinite retry loops
- Only intercept non-auth endpoints (skip `/auth/refresh`, `/auth/login`)
- On refresh failure → clear tokens → redirect to login

### ProtectedRoute Guard

**File:** `frontend/src/components/shared/ProtectedRoute.tsx`

```typescript
// Module-level flag (survives React remounts):
let hasVerifiedThisSession = false

// Reset on logout:
useAuthStore.subscribe((state, prev) => {
    if (prev.isAuthenticated && !state.isAuthenticated) {
        hasVerifiedThisSession = false
    }
})

// Guard decision tree (evaluated in order):
// 1. !hasHydrated → Loading (wait for Zustand persist)
// 2. isAuthenticated → Show children (trust persisted state)
// 3. hasToken && !verifyComplete → Loading (recovering via /auth/me)
// 4. default → Navigate to /login
```

**Guard states:**

| State | Render | Why |
|-------|--------|-----|
| `!hasHydrated` | Loading spinner | Zustand restoring from localStorage |
| `isAuthenticated` (store) | Children immediately | Trust persisted auth state |
| `hasToken && !verifyComplete` | Loading spinner | Token recovery: call /auth/me |
| Default | Navigate /login | Not authenticated |

## StrictMode Compatibility

React StrictMode mounts/unmounts components twice in development.

**Problem:**
```
Mount 1: starts /auth/me, verifyingRef = true
Cleanup: aborts fetch, but verifyingRef stays true  ← BUG
Mount 2: sees verifyingRef = true, skips → stuck loading forever
```

**Fix:**
```typescript
useEffect(() => {
    verifyingRef.current = true
    const controller = new AbortController()
    // ... fetch /auth/me ...

    return () => {
        controller.abort()
        verifyingRef.current = false  // CRITICAL: Reset for StrictMode remount
    }
}, [...])
```

**Rule:** Always reset ref flags in effect cleanup when using AbortController pattern.

## Nginx Headers for Auth

**File:** `frontend/nginx.conf`

```nginx
# REQUIRED for Google Sign-In popup postMessage
add_header Cross-Origin-Opener-Policy "same-origin-allow-popups" always;
```

- Must be in ALL location blocks (nginx inheritance limitation)
- `same-origin-allow-popups` (NOT `same-origin`) — allows Google popup communication
- Without this: Google button click does nothing, no error in console

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| Flash of login page | Briefly see login then dashboard | Use session-level flag (`hasVerifiedThisSession`) |
| Stuck on loading in dev | Page never leaves spinner | Reset `verifyingRef` in effect cleanup (StrictMode) |
| Lost tokens on store clear | Authenticated → suddenly logged out | ProtectedRoute checks localStorage directly |
| Multiple concurrent refreshes | 401 → refresh → 401 → refresh loop | Mutex flag + promise subscriber queue |
| Google popup silent failure | Button click does nothing | Add COOP header: `same-origin-allow-popups` |
| Network error logs user out | Temporary glitch = forced re-login | Only logout on 401/403, not network errors |
| API key in localStorage | Security risk | Use `partialize` to exclude from persist |
| Infinite retry on 401 | Request loops forever | Add `_retry` flag, only retry once |

## Adding a New Protected Endpoint (Frontend)

When adding a new API call that requires auth:

1. Use `api` instance from `@/lib/api` (has token interceptor)
2. Backend route must be inside JWT-protected group in router
3. Handler extracts `customerID := middleware.GetCustomerID(r.Context())`
4. Service checks ownership: `item.CustomerID != customerID`
5. Frontend hook invalidates queries on success

```typescript
// Frontend hook example:
export function useMyFeature() {
    return useQuery({
        queryKey: ['my-feature'],
        queryFn: async () => {
            const { data } = await api.get<APIResponse<MyFeature[]>>('/my-feature')
            return data.data!
        },
    })
}
```

The token refresh interceptor handles 401s automatically — no additional auth code needed in hooks.

## Key Files

| File | Purpose |
|------|---------|
| `backend/internal/service/auth_service.go` | Google auth, token generation, refresh |
| `backend/internal/handler/auth_handler.go` | Auth HTTP endpoints |
| `backend/internal/middleware/auth.go` | JWT validation middleware |
| `frontend/src/stores/auth-store.ts` | Zustand persist store |
| `frontend/src/lib/api.ts` | Axios + token refresh interceptor |
| `frontend/src/components/shared/ProtectedRoute.tsx` | Route guard |
| `frontend/src/pages/auth/LoginPage.tsx` | Google Sign-In UI |
| `frontend/src/hooks/use-auth.ts` | Auth mutations (login, logout) |
| `frontend/nginx.conf` | COOP header for Google popup |

## Verification

```bash
# Backend auth tests
cd backend && go test ./internal/service/... -run TestAuth

# Frontend builds without type errors
cd frontend && npm run build

# Manual test: Google login → dashboard → refresh page → still authenticated
```

## Related

- `nginx-csp` for CSP headers that affect Google Sign-In
- `frontend-feature` for creating new protected pages
- `go-service-scaffold` for creating new authenticated resources
- Built-in `security-review` for auth security audit
