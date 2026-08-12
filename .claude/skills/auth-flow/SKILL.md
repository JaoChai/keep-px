---
name: auth-flow
description: Neon Auth (managed) + JWT/JWKS verification, ProtectedRoute guard — ใช้เมื่อ: login ไม่ได้, ล็อกอินพัง, token หมดอายุ, 401 error, auth error, หน้าจอกระพริบ, session expired, ล็อกอินไม่ได้, sign in พัง, customer_not_provisioned
---

# Auth Flow

## When to Activate

Activate this skill when the user says:
- "Login broken" / "Auth not working" / "Can't sign in"
- "401 error" / "customer_not_provisioned"
- "Flash of login page" / "Stuck on loading" / "White screen on auth"
- "Add auth to [feature]" / "Protect route"
- "Neon Auth" / "JWKS" / "auth_user_id"

## Architecture (เปลี่ยนไป Neon Auth ตั้งแต่ 2026-08-12)

Neon Auth (managed Better Auth) ถือ session ทั้งหมด — เราไม่ออก JWT เองแล้ว ไม่มี refresh token
ในฐานข้อมูลของเราแล้ว

```
Frontend: authClient.signIn.social({provider:'google'}) → Neon ↔ Google โดยตรง
    ↓
Neon ออก session cookie (โดเมนของ Neon) — ไม่ใช่ของเรา
    ↓
Frontend ทุก request: GET {authURL}/token → ได้ JWT (EdDSA, อายุ 15 นาที, cache ไว้)
    ↓ Authorization: Bearer <jwt>
Backend middleware: ตรวจกับ JWKS ของ Neon → sub → GetByAuthUserID → customer
    ↓ ไม่มี customer (บัญชีใหม่) → 401 customer_not_provisioned
Frontend interceptor: POST /auth/session → ProvisionCustomer สร้าง/ผูกบัญชี → retry request เดิม
```

**ของที่หายไปแล้ว ห้ามสร้างกลับมา:** `JWT_SECRET`, refresh token ในฐานข้อมูล, `/auth/google`,
`/auth/refresh`, `/auth/dev-login`, `google_id` column, `@react-oauth/google`

## Backend Auth Flow

**File:** `backend/internal/service/auth_service.go`

### ProvisionCustomer — สร้างหรือผูก customer กับ user ของ Neon

```go
func (s *AuthService) ProvisionCustomer(ctx context.Context, in ProvisionInput) (*domain.Customer, error) {
    // 1. GetByAuthUserID(in.AuthUserID) → มีอยู่แล้ว → คืนเลย (เช็ค SuspendedAt ก่อน)
    // 2. ยังไม่มี → ต้อง EmailVerified == true ก่อนเสมอ (กันสวมบัญชีด้วย email ปลอม)
    // 3. GetByEmail(in.Email) → เจอบัญชีเดิม:
    //    - ถ้า AuthUserID เดิมผูกกับคนอื่นแล้ว (ไม่ nil และไม่ตรงกัน) → ErrEmailAlreadyLinked
    //    - ไม่งั้น → ผูก AuthUserID เข้ากับบัญชีเดิม (คง api_key/plan/credit เดิมไว้)
    // 4. ไม่เจอเลย → สร้างใหม่ (plan sandbox, retention 7 วัน)
}
```

**ลำดับการเช็คสำคัญมาก:** `EmailVerified` ต้องเช็คก่อน `GetByEmail` เสมอ — สลับลำดับ = ช่องโหว่
สวมบัญชี · การเช็ค `AuthUserID != nil && *AuthUserID != in.AuthUserID` ต้องมี — ไม่งั้นบัญชีที่สอง
ที่ email ตรงกันยึดบัญชีแรกได้

### JWT Middleware

**File:** `backend/internal/middleware/auth.go`

```go
// JWTAuth(keyFn jwt.Keyfunc, issuer string, lookup CustomerLookup)
// ตรวจ EdDSA (ไม่ใช่ RS256) กับกุญแจจาก JWKS ของ Neon
// อ่านสิทธิ์ (is_admin/suspended_at) จาก DB เสมอ — ไม่เชื่อ claim ที่ปลอมได้
// jwks มาจาก keyfunc.NewDefaultOverrideCtx(..., Override{NoErrorReturnFirstHTTPReq: &false})
// — ต้อง false ไม่งั้น container ขึ้นมาได้แม้ JWKS ดึงไม่สำเร็จ แล้ว login พังเงียบๆ ทุกคน
```

**`Session` handler** (`auth_handler.go`) ตรวจ JWT ซ้ำเองแบบเดียวกัน (ไม่ผ่าน `JWTAuth` middleware)
เพราะปัญหาไก่กับไข่: middleware จะปฏิเสธด้วย `customer_not_provisioned` ก่อนที่จะสร้าง customer ได้
— duplication นี้ตั้งใจ ไม่ใช่ต้องแก้

## Frontend Auth Flow

### neon-auth.ts — ตัวกลางคุยกับ Neon

**File:** `frontend/src/lib/neon-auth.ts`

```typescript
// authClient = createAuthClient(VITE_NEON_AUTH_URL) — ใช้ signIn.social/signOut
//
// getAccessToken() — JWT ที่ backend ตรวจได้ไม่ใช่ authClient.getSession().session.token
// (นั่นเป็น session token ทึบ ตรวจกับ JWKS ไม่ได้) ต้องขอจาก GET {authURL}/token แยก
// cache ไว้ 15 นาที (เผื่อ 60 วิ) · decode ด้วย base64url ไม่ใช่ atob() ตรงๆ
// (atob รับแค่ base64 มาตรฐาน — payload จริงมี -/_ บ่อย จะ throw)
```

**กฎเหล็ก:** `VITE_NEON_AUTH_URL` ต้องตั้งค่าเสมอ — ไฟล์นี้ throw ตอน module-load ถ้าไม่มี
ซึ่งเกิดก่อน React mount → ErrorBoundary จับไม่ได้ → หน้าขาวทั้งแอป เช็คใน `.env`/CI ก่อนถ้าเจอ
หน้าขาว

### Auth Store (Zustand + Persist)

**File:** `frontend/src/stores/auth-store.ts`

```typescript
// Persisted: customer, isAuthenticated — api_key ไม่ persist (อยู่ในหน่วยความจำเท่านั้น)
// ไม่มี token ให้เก็บแล้ว — Neon ถือ session ผ่านคุกกี้ของโดเมนตัวเอง
```

### api.ts — interceptor

**File:** `frontend/src/lib/api.ts`

```typescript
// request: await getAccessToken() → แนบ Authorization header
// response 401 + "customer_not_provisioned": POST /auth/session → setCustomer → retry ครั้งเดียว
// response 401 อื่น: clearAccessToken() + authClient.signOut() + logout() + redirect /login
// response 403 "account suspended": toast
```

### ProtectedRoute Guard

**File:** `frontend/src/components/shared/ProtectedRoute.tsx`

```typescript
// getAccessToken() → มี token → GET /auth/me → setCustomer → render children
// ไม่มี token หรือ auth error จริง (err.response มีค่า) → Navigate /login
// network error (err.response ไม่มีค่า) → ถือว่ายัง "มี session" ปล่อยผ่าน (กันเน็ตสะดุดแล้วเด้งออก)
```

**สำคัญ:** ต้องเรียก `/auth/me` แล้ว `setCustomer` เอง — ไม่งั้น `AdminRoute`/`Sidebar` ที่อ่าน
`customer.is_admin` จาก store จะได้ `null` ตลอดสำหรับผู้ใช้ที่ login ค้างอยู่แล้ว (แอดมินจะถูกเด้ง
ออกจากหน้าแอดมินทุกครั้งที่ reload)

### Logout

**File:** `frontend/src/components/layout/Sidebar.tsx` (จุดเดียวในแอปที่มีปุ่ม logout)

```typescript
// ต้องเรียกทั้งสามอย่างก่อน navigate:
// clearAccessToken() → await authClient.signOut() → logout() (zustand)
// ขาด authClient.signOut() = session ของ Neon ยังอยู่ → กลับเข้า dashboard = login คืนอัตโนมัติ
```

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| ไม่ตั้ง `VITE_NEON_AUTH_URL` | หน้าเว็บขาวทั้งหน้า ไม่มี error ใน UI | เช็ค `.env`/CI repo variable — module throw ก่อน React mount |
| `atob()` ตรงๆ กับ JWT payload | login ล้มเหลวแบบสุ่ม (ไม่คงที่) | ต้องแปลง base64url→base64 ก่อน (`neon-auth.ts:decodeJwtPayload`) |
| Logout ไม่เรียก `authClient.signOut()` | logout แล้วกลับเข้ามา login อัตโนมัติ | เรียก `clearAccessToken()` + `signOut()` ก่อน `logout()` เสมอ |
| ProvisionCustomer ไม่เช็ค `AuthUserID` เดิม | บัญชีที่สองยึดบัญชีแรกได้ | เช็ค `customer.AuthUserID != nil && *AuthUserID != in.AuthUserID` ก่อน link |
| `ProtectedRoute` ไม่เรียก `/auth/me` | แอดมินถูกเด้งออกจากหน้าแอดมินทุก reload | ต้อง `setCustomer` จาก `/auth/me` เสมอหลังยืนยัน session |
| Network error logs user out | เน็ตสะดุดแป๊บ = ถูกเด้งออก | เช็ค `err.response` แยก auth error จาก network error |
| Claim name พิมพ์ผิด (`email_verified` แทน `emailVerified`) | ไม่มีใคร provision บัญชีใหม่ได้ ไม่มี error ฟ้อง | claim เป็น **camelCase** เสมอ — มีเทสต์ `TestAuthHandler_Session` คุมอยู่ |
| `keyfunc.NewDefaultCtx` (ไม่ใช่ Override) | container ขึ้นได้แม้ JWKS ดึงไม่สำเร็จ ทุกคน login ไม่ได้เงียบๆ นานถึง 1 ชม. | ต้องใช้ `NewDefaultOverrideCtx` + `NoErrorReturnFirstHTTPReq: &false` |

## Key Files

| File | Purpose |
|------|---------|
| `backend/internal/service/auth_service.go` | `ProvisionCustomer`, `GetCustomerByID`, `RegenerateAPIKey` |
| `backend/internal/handler/auth_handler.go` | `Session`, `Me`, `RegenerateAPIKey` |
| `backend/internal/middleware/auth.go` | ตรวจ JWT ผ่าน JWKS ของ Neon |
| `backend/internal/router/router.go` | โหลด JWKS ตอน startup, ต่อสาย middleware+handler |
| `frontend/src/lib/neon-auth.ts` | `authClient`, `getAccessToken`, `clearAccessToken` |
| `frontend/src/lib/api.ts` | axios interceptor: แนบ token + retry provision |
| `frontend/src/stores/auth-store.ts` | Zustand persist store (ไม่มี token แล้ว) |
| `frontend/src/components/shared/ProtectedRoute.tsx` | Route guard |
| `frontend/src/components/layout/Sidebar.tsx` | ปุ่ม login/logout |
| `frontend/src/pages/auth/LoginPage.tsx` | ปุ่ม `authClient.signIn.social` |

## Verification

```bash
# Backend auth tests
cd backend && go test ./internal/service/... -run TestProvisionCustomer
cd backend && go test ./internal/handler/... -run TestAuthHandler_Session
cd backend && go test ./internal/middleware/... -run TestJWTAuth

# Frontend
cd frontend && npx vitest run src/lib/__tests__/neon-auth.test.ts src/lib/__tests__/api.test.ts
cd frontend && npm run build

# Manual test: Google login → dashboard → reload → still authenticated → logout → login ไม่คืนอัตโนมัติ
```

## Related

- `cloudflare-deploy` สำหรับ secret/env ที่ Worker ต้องมี
- `worker-headers` สำหรับ CSP ที่ต้องยอม `*.neon.tech`
- `docs/superpowers/specs/2026-08-12-neon-auth-migration-design.md` — design เต็ม
- `docs/superpowers/plans/2026-08-12-neon-auth-migration.md` — implementation plan + ผล spike
