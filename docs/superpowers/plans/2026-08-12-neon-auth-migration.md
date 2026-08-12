# ย้ายระบบ login ไป Neon Auth — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ให้ Neon Auth ถือ session ทั้งหมด แล้วรื้อระบบ JWT + refresh token ที่เราเขียนเองทิ้ง

**Architecture:** Neon Auth (Managed Better Auth) คุยกับ Google เองและออก JWT · React แนบ JWT
ไปกับทุก request · Go ตรวจลายเซ็นกับ JWKS ของ Neon แล้วแปลง `sub` เป็น `customer` ผ่านคอลัมน์ใหม่
`customers.auth_user_id` · ตาราง `customers` ยังถือของทางธุรกิจทั้งหมดตามเดิม

**Tech Stack:** Go 1.23+ (chi, golang-jwt/v5, MicahParks/keyfunc/v3) · React + TypeScript + axios
+ zustand · `@neondatabase/neon-js` · Neon Postgres · Cloudflare Workers + Containers

**Spec:** `docs/superpowers/specs/2026-08-12-neon-auth-migration-design.md`

## Global Constraints

- CLAUDE.md §2 **Simplicity First** — ห้ามใส่ความยืดหยุ่นที่ไม่มีใครขอ ห้าม cache ที่ยังไม่ได้วัด
- CLAUDE.md §3 **Surgical Changes** — แตะเฉพาะที่แผนสั่ง ห้ามปรับปรุงโค้ดข้างเคียง
- CLAUDE.md §4 **TDD** — เขียนเทสต์ที่ล้มก่อนเสมอ แล้วต้องเห็นมันล้มจริงก่อนเขียน implementation
- **ห้ามรันคำสั่งที่เขียนข้อมูลลงฐานข้อมูล production** — `backend/.env` ชี้ production อยู่
  ต้อง override ก่อนเสมอ
- `NEON_AUTH_URL` รูปแบบ `https://ep-xxx.neon.tech/neondb/auth` · JWKS อยู่ที่
  `{NEON_AUTH_URL}/.well-known/jwks.json` · issuer คือ origin ของ URL นั้น
- ชื่อคอลัมน์ใหม่คือ `auth_user_id` (ไม่ใช่ `neon_user_id` หรืออย่างอื่น) ใช้ชื่อนี้ทุกที่
- ข้อความที่ผู้ใช้เห็นเป็นภาษาไทย · ข้อความ error ใน API เป็นภาษาอังกฤษตามของเดิม

## File Structure

**สร้างใหม่**
- `backend/db/migrations/000028_add_auth_user_id.{up,down}.sql` — เพิ่มคอลัมน์เชื่อมกับ Neon
- `backend/db/migrations/000029_drop_legacy_auth.{up,down}.sql` — ทิ้งของเก่าหลังโค้ดเลิกใช้แล้ว
- `frontend/src/lib/neon-auth.ts` — สร้างและ export `authClient` ตัวเดียวของทั้งแอป

**แก้**
- `backend/internal/middleware/auth.go` — เปลี่ยนวิธีตรวจ + เพิ่มการค้น customer
- `backend/internal/repository/interfaces.go` · `postgres/customer_repo.go` · `mocks/customer.go`
- `backend/internal/service/auth_service.go` · `handler/auth_handler.go` · `router/router.go`
- `backend/internal/config/config.go` · `backend/go.mod`
- `frontend/src/lib/api.ts` · `stores/auth-store.ts` · `pages/auth/LoginPage.tsx` · `App.tsx`
- `worker/headers.ts` · `wrangler.jsonc`

**ลบ**
- `backend/internal/repository/postgres/refresh_token_repo.go` + mock + interface
- `frontend/src/pages/auth/AuthCallbackPage.tsx`

---

### Task 1: Spike — พิสูจน์ว่า Go ตรวจ JWT ของ Neon ได้จริง

**นี่คือด่านตัดสิน ถ้าไม่ผ่านให้หยุดทั้งแผนแล้วกลับไปคุยกับเจ้าของ ห้ามเดินต่อ**

**Files:**
- Create: `/tmp/neon-spike/main.go` (ของชั่วคราว ลบทิ้งตอนจบ task ห้าม commit)
- Modify: `docs/superpowers/plans/2026-08-12-neon-auth-migration.md` (บันทึกผลท้ายไฟล์)

**Interfaces:**
- Consumes: ไม่มี
- Produces: ค่าจริงของ `NEON_AUTH_URL` · ชื่อ claim ที่ Neon ใส่มาให้ (`sub`, `email`,
  `email_verified` หรือชื่ออื่น) · คำสั่งที่ถูกต้องของ client SDK สำหรับ login ด้วย Google
  → Task 3, 4, 6 ใช้ค่าเหล่านี้

- [ ] **Step 1: สร้าง branch ทดสอบของ Neon (ห้ามแตะ production)**

ใช้ Neon MCP หรือ console สร้าง branch ชื่อ `neon-auth-spike` จาก production
บันทึก endpoint ที่ได้ แล้วยืนยันว่า **ไม่ใช่** `ep-cool-butterfly-ai3qizfi` ซึ่งเป็นของ production

- [ ] **Step 2: เปิด Neon Auth บน branch นั้น แล้วตั้งค่า Google**

ใน Neon Console → Auth → เปิดใช้งาน → เปิด provider `google`
รอบแรกให้ใช้ credential กลางสำหรับทดสอบที่ Neon ให้มา (ยังไม่ต้องสร้าง OAuth app ของตัวเอง)
บันทึกค่า `NEON_AUTH_URL` ที่ console แสดง

- [ ] **Step 3: ให้เจ้าของ login ด้วย Google จริงหนึ่งครั้ง แล้วเก็บ JWT มา**

**ขั้นนี้ Claude ทำแทนไม่ได้ ต้องให้เจ้าของกดเอง** เพราะต้องใช้บัญชี Google จริง
ขอให้เจ้าของเปิดหน้า login ที่ Neon Console เตรียมไว้ให้ → login → copy ค่า session token
มาวางให้ (token มีอายุสั้น ใช้ทดสอบแล้วหมดอายุไปเอง)

- [ ] **Step 4: เขียนโปรแกรม Go ตรวจ JWT ตัวนั้น**

```go
// /tmp/neon-spike/main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	authURL := os.Getenv("NEON_AUTH_URL") // https://ep-xxx.neon.tech/neondb/auth
	tokenStr := os.Getenv("NEON_TOKEN")

	k, err := keyfunc.NewDefaultCtx(context.Background(), []string{authURL + "/.well-known/jwks.json"})
	if err != nil {
		fmt.Println("FAIL ดึง JWKS ไม่ได้:", err)
		os.Exit(1)
	}

	token, err := jwt.Parse(tokenStr, k.Keyfunc)
	if err != nil || !token.Valid {
		fmt.Println("FAIL ตรวจ token ไม่ผ่าน:", err)
		os.Exit(1)
	}

	claims := token.Claims.(jwt.MapClaims)
	fmt.Println("PASS — claim ทั้งหมดที่ Neon ส่งมา:")
	for key, value := range claims {
		fmt.Printf("  %s = %v\n", key, value)
	}
}
```

- [ ] **Step 5: รันแล้วดูผล**

```bash
cd /tmp/neon-spike && go mod init spike && go get github.com/MicahParks/keyfunc/v3 github.com/golang-jwt/jwt/v5
NEON_AUTH_URL='<ค่าจาก Step 2>' NEON_TOKEN='<ค่าจาก Step 3>' go run main.go
```

Expected: ขึ้นคำว่า `PASS` แล้วตามด้วยรายการ claim
**ต้องเช็ค exit code ตรง ๆ ห้ามเขียน `go run main.go | tail && echo ok`** เพราะ exit code
ของคำสั่งจริงจะถูกกลบ (บทเรียนที่เคยพลาดมาแล้วสองรอบตอนย้าย Cloudflare)

- [ ] **Step 6: หาชื่อคำสั่ง login ของ client SDK ให้แน่ชัด**

อ่านเอกสาร `@neondatabase/neon-js` แล้วยืนยันว่าคำสั่ง login ด้วย Google ชื่ออะไรแน่
สมมติฐานตอนเขียนแผนคือ `authClient.signIn.social({ provider: 'google', callbackURL })`
**ยังไม่ได้พิสูจน์** ถ้าของจริงต่างจากนี้ ให้แก้ Task 6 Step 3 ก่อนเริ่มทำ

- [ ] **Step 7: บันทึกผลลงท้ายไฟล์แผนนี้ แล้วลบของชั่วคราวทิ้ง**

เขียนหัวข้อ `## ผล Spike` ท้ายไฟล์ ระบุ: `NEON_AUTH_URL` จริง · ชื่อ claim ที่ได้จริง ·
คำสั่ง SDK ที่ถูกต้อง · อะไรที่ต่างจากที่แผนเดาไว้

```bash
rm -rf /tmp/neon-spike
git add docs/superpowers/plans/2026-08-12-neon-auth-migration.md
git commit -m "docs(plan): บันทึกผล spike Neon Auth — Go ตรวจ JWT ผ่าน JWKS ได้จริง"
```

---

### Task 2: เพิ่มคอลัมน์ `auth_user_id` และวิธีค้นหาด้วยคอลัมน์นี้

**Files:**
- Create: `backend/db/migrations/000028_add_auth_user_id.up.sql`
- Create: `backend/db/migrations/000028_add_auth_user_id.down.sql`
- Modify: `backend/internal/domain/customer.go`
- Modify: `backend/internal/repository/interfaces.go:19-31`
- Modify: `backend/internal/repository/postgres/customer_repo.go`
- Modify: `backend/internal/repository/mocks/customer.go`
- Test: `backend/internal/repository/postgres/customer_repo_integration_test.go`

**Interfaces:**
- Consumes: ไม่มี
- Produces: `domain.Customer.AuthUserID *string` ·
  `CustomerRepository.GetByAuthUserID(ctx context.Context, authUserID string) (*domain.Customer, error)`
  (คืน `nil, nil` เมื่อไม่พบ ตามแบบเดียวกับ `GetByGoogleID` ที่มีอยู่)
  → Task 3 และ Task 4 ใช้

- [ ] **Step 1: เขียน migration**

```sql
-- backend/db/migrations/000028_add_auth_user_id.up.sql
ALTER TABLE customers ADD COLUMN auth_user_id TEXT;
CREATE UNIQUE INDEX idx_customers_auth_user_id ON customers (auth_user_id)
    WHERE auth_user_id IS NOT NULL;
```

```sql
-- backend/db/migrations/000028_add_auth_user_id.down.sql
DROP INDEX IF EXISTS idx_customers_auth_user_id;
ALTER TABLE customers DROP COLUMN IF EXISTS auth_user_id;
```

index เป็นแบบ partial (`WHERE auth_user_id IS NOT NULL`) เพราะ customer ที่ยังไม่ผูกกับ Neon
จะมีค่าเป็น NULL หลายแถว ถ้าใช้ unique index ธรรมดาบน Postgres จะยอมให้ NULL ซ้ำได้อยู่แล้ว
แต่เขียนให้ชัดดีกว่าเพื่อให้คนอ่านรู้เจตนา

- [ ] **Step 2: เพิ่มฟิลด์ลง domain**

ใน `backend/internal/domain/customer.go` เพิ่มบรรทัดถัดจาก `GoogleID`:

```go
	AuthUserID       *string    `json:"-"`
```

- [ ] **Step 3: เขียนเทสต์ที่ล้ม**

**ไฟล์นี้ยังไม่มี ต้องสร้างใหม่** — repo ตัวนี้ยังไม่เคยมี integration test เลย
ให้ทำตามรูปแบบของ `pixel_repo_integration_test.go` ที่มีอยู่แล้ว (build tag + `testutil`)

```go
//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerRepo_GetByAuthUserID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewCustomerRepo(pool)
	ctx := context.Background()

	authUserID := "neon-user-abc123"
	c := &domain.Customer{
		Email:         "authuser@example.com",
		Name:          "Auth User",
		APIKey:        "pk_test_getbyauthuserid",
		Plan:          domain.PlanSandbox,
		RetentionDays: 7,
		AuthUserID:    &authUserID,
	}
	require.NoError(t, repo.Create(ctx, c))

	t.Run("เจอเมื่อมีอยู่จริง", func(t *testing.T) {
		got, err := repo.GetByAuthUserID(ctx, authUserID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, c.ID, got.ID)
		require.NotNil(t, got.AuthUserID, "ถ้า scanCustomer ลืมคอลัมน์ใหม่ ตรงนี้จะจับได้")
		assert.Equal(t, authUserID, *got.AuthUserID)
	})

	t.Run("คืน nil เมื่อไม่มี ไม่ใช่ error", func(t *testing.T) {
		got, err := repo.GetByAuthUserID(ctx, "neon-user-ไม่มีอยู่จริง")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
```

- [ ] **Step 4: รันแล้วต้องเห็นมันล้ม**

```bash
cd backend && go test -tags=integration ./internal/repository/postgres/ -run TestCustomerRepo_GetByAuthUserID -v
```

Expected: FAIL — คอมไพล์ไม่ผ่าน `repo.GetByAuthUserID undefined`
**แนบ output ที่ล้มจริงมาในรายงานด้วย**

- [ ] **Step 5: เพิ่ม method ลง interface**

ใน `backend/internal/repository/interfaces.go` เพิ่มบรรทัดถัดจาก `GetByGoogleID`:

```go
	GetByAuthUserID(ctx context.Context, authUserID string) (*domain.Customer, error)
```

- [ ] **Step 6: เขียน implementation**

ใน `backend/internal/repository/postgres/customer_repo.go` เพิ่มเมธอดใหม่ถัดจาก `GetByGoogleID`
(บรรทัด 67-72) โดยใช้รูปแบบเดียวกันเป๊ะ:

```go
func (r *CustomerRepo) GetByAuthUserID(ctx context.Context, authUserID string) (*domain.Customer, error) {
	return scanCustomer(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, google_id, name, api_key, plan, retention_days, stripe_customer_id, is_admin, suspended_at, auth_user_id, created_at, updated_at
		 FROM customers WHERE auth_user_id = $1`, authUserID,
	))
}
```

**สำคัญ:** ทุก `SELECT` ในไฟล์นี้เขียนรายชื่อคอลัมน์ไว้ตรง ๆ และ `scanCustomer` (บรรทัด 19)
`Scan` ตามลำดับนั้น → ต้องเพิ่ม `auth_user_id` ในตำแหน่ง**เดียวกัน**ทั้ง 3 ที่:
1. รายชื่อคอลัมน์ของ **ทุก** `SELECT` ในไฟล์ (`GetByID` · `GetByEmail` · `GetByGoogleID`
   · `GetByAPIKey` · `GetByStripeCustomerID` · `RegenerateAPIKey` และตัวใหม่นี้)
2. `scanCustomer` — เพิ่ม `&c.AuthUserID` ตามลำดับให้ตรงกับรายชื่อคอลัมน์
3. `Create` และ `Update` — เพิ่มคอลัมน์กับพารามิเตอร์

ถ้าลำดับไม่ตรงกัน pgx จะ scan ค่าผิดช่องโดยไม่มี error ฟ้อง — เทสต์ใน Step 3 ที่เช็ค
`*got.AuthUserID` มีไว้จับกรณีนี้

- [ ] **Step 7: เพิ่ม method ลง mock**

ใน `backend/internal/repository/mocks/customer.go` ทำตามรูปแบบของ `GetByGoogleID` ที่มีอยู่

- [ ] **Step 8: รันเทสต์ต้องผ่าน**

```bash
cd backend && go test -tags=integration ./internal/repository/postgres/ -run TestCustomerRepo_GetByAuthUserID -v
go build ./... && go test ./internal/...
```

Expected: PASS ทั้งหมด และ `go build` exit 0

- [ ] **Step 9: Commit**

```bash
git add backend/db/migrations/000028_* backend/internal/domain/customer.go \
        backend/internal/repository/
git commit -m "feat(auth): เพิ่มคอลัมน์ auth_user_id สำหรับผูกกับ Neon Auth"
```

---

### Task 3: เปลี่ยน middleware ให้ตรวจ JWT ของ Neon

**Files:**
- Modify: `backend/internal/middleware/auth.go` (ทั้งไฟล์ 67 บรรทัด)
- Modify: `backend/internal/config/config.go:14-27`
- Modify: `backend/internal/router/router.go:156`
- Modify: `backend/go.mod`
- Test: `backend/internal/middleware/auth_test.go` (เขียนใหม่ทั้งไฟล์)

**Interfaces:**
- Consumes: `CustomerRepository.GetByAuthUserID` จาก Task 2
- Produces:
  - `type CustomerLookup func(ctx context.Context, authUserID string) (*domain.Customer, error)`
  - `func JWTAuth(keyFn jwt.Keyfunc, issuer string, lookup CustomerLookup) func(http.Handler) http.Handler`
  - `middleware.GetCustomerID(ctx) string` และ `middleware.GetIsAdmin(ctx) bool` ยังใช้ชื่อเดิม
    และคืนค่าแบบเดิม → handler ทั้งหมดไม่ต้องแก้
  - `config.Config.NeonAuthURL string`
  → Task 4 และ Task 7 ใช้

**หมายเหตุการออกแบบ:** `JWTAuth` รับ `jwt.Keyfunc` เข้ามาแทนที่จะสร้าง keyfunc เอง เพื่อให้เทสต์
สร้างกุญแจ RSA ของตัวเองแล้วฉีดเข้าไปได้ ไม่ต้องต่อเน็ตตอนรันเทสต์

- [ ] **Step 1: ติดตั้ง keyfunc**

```bash
cd backend && go get github.com/MicahParks/keyfunc/v3
```

- [ ] **Step 2: เขียนเทสต์ที่ล้ม (เขียนทับไฟล์เดิมทั้งไฟล์)**

```go
package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

const testIssuer = "https://ep-test.neon.tech"

// newTestKey สร้างกุญแจ Ed25519 ในเครื่องสำหรับเซ็นและตรวจ token ในเทสต์
// ไม่ต้องต่อเน็ตและไม่ต้องมี JWKS server จริง
// **Ed25519 ไม่ใช่ RSA** — spike ยืนยันแล้วว่า Neon เซ็นด้วย EdDSA (kty=OKP)
func newTestKey(t *testing.T) (ed25519.PrivateKey, jwt.Keyfunc) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return priv, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return pub, nil
	}
}

func makeToken(t *testing.T, key ed25519.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(key)
	require.NoError(t, err)
	return tokenStr
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "neon-user-1",
		"iss": testIssuer,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

// lookupReturning สร้าง CustomerLookup ที่คืนค่าตามที่กำหนด
func lookupReturning(c *domain.Customer) CustomerLookup {
	return func(_ context.Context, _ string) (*domain.Customer, error) { return c, nil }
}

func runMiddleware(t *testing.T, keyFn jwt.Keyfunc, lookup CustomerLookup, token string) (*httptest.ResponseRecorder, string, bool) {
	t.Helper()
	var gotID string
	var gotAdmin bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetCustomerID(r.Context())
		gotAdmin = GetIsAdmin(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	JWTAuth(keyFn, testIssuer, lookup)(next).ServeHTTP(rec, req)
	return rec, gotID, gotAdmin
}

func TestJWTAuth_TokenถูกและมีCustomer(t *testing.T) {
	key, keyFn := newTestKey(t)
	customer := &domain.Customer{ID: "cust-1", IsAdmin: true}

	rec, gotID, gotAdmin := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "cust-1", gotID, "ต้องใส่ id ของ customer ไม่ใช่ sub ของ Neon")
	assert.True(t, gotAdmin, "ต้องอ่าน is_admin จากฐานข้อมูล ไม่ใช่จาก claim")
}

func TestJWTAuth_ไม่เชื่อis_adminที่ปลอมมาในtoken(t *testing.T) {
	key, keyFn := newTestKey(t)
	claims := validClaims()
	claims["is_admin"] = true // คนร้ายยัดมาเอง
	customer := &domain.Customer{ID: "cust-1", IsAdmin: false}

	rec, _, gotAdmin := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, claims))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotAdmin, "claim ปลอมต้องไม่มีผล")
}

func TestJWTAuth_บัญชีถูกระงับ(t *testing.T) {
	key, keyFn := newTestKey(t)
	suspended := time.Now()
	customer := &domain.Customer{ID: "cust-1", SuspendedAt: &suspended}

	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestJWTAuth_ยังไม่มีCustomer(t *testing.T) {
	key, keyFn := newTestKey(t)

	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(nil),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "customer_not_provisioned",
		"frontend ใช้รหัสนี้แยกว่าต้องเรียก /auth/session ไม่ใช่เตะออกหน้า login")
}

func TestJWTAuth_ปฏิเสธtokenที่ไม่ถูกต้อง(t *testing.T) {
	key, keyFn := newTestKey(t)
	otherKey, _ := newTestKey(t)
	customer := &domain.Customer{ID: "cust-1"}

	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()

	wrongIssuer := validClaims()
	wrongIssuer["iss"] = "https://evil.example.com"

	noSub := validClaims()
	delete(noSub, "sub")

	cases := map[string]string{
		"หมดอายุ":        makeToken(t, key, expired),
		"ผู้ออกผิด":       makeToken(t, key, wrongIssuer),
		"ไม่มี sub":       makeToken(t, key, noSub),
		"เซ็นด้วยกุญแจอื่น": makeToken(t, otherKey, validClaims()),
		"ข้อความมั่ว":      "ไม่ใช่ token",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			rec, _, _ := runMiddleware(t, keyFn, lookupReturning(customer), token)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestJWTAuth_ไม่มีheader(t *testing.T) {
	_, keyFn := newTestKey(t)
	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(nil), "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 3: รันแล้วต้องเห็นมันล้ม**

```bash
cd backend && go test ./internal/middleware/ -run TestJWTAuth -v
```

Expected: FAIL — คอมไพล์ไม่ผ่าน `too many arguments in call to JWTAuth` และ
`undefined: CustomerLookup`
**แนบ output ที่ล้มจริงมาในรายงาน**

- [ ] **Step 4: เขียน middleware ใหม่**

เขียนทับ `backend/internal/middleware/auth.go` ทั้งไฟล์:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

type contextKey string

const (
	CustomerIDKey contextKey = "customer_id"
)

// CustomerLookup แปลง sub ของ Neon Auth เป็น customer ของเรา
// คืน nil, nil เมื่อยังไม่มี customer ผูกกับ user รายนี้
type CustomerLookup func(ctx context.Context, authUserID string) (*domain.Customer, error)

func JWTAuth(keyFn jwt.Keyfunc, issuer string, lookup CustomerLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				writeJSONError(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			token, err := jwt.Parse(tokenStr, keyFn,
				jwt.WithIssuer(issuer),
				jwt.WithExpirationRequired(),
				jwt.WithValidMethods([]string{"EdDSA"}),
			)
			if err != nil || !token.Valid {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "invalid claims")
				return
			}

			authUserID, ok := claims["sub"].(string)
			if !ok || authUserID == "" {
				writeJSONError(w, http.StatusUnauthorized, "invalid subject")
				return
			}

			customer, err := lookup(r.Context(), authUserID)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "lookup failed")
				return
			}
			if customer == nil {
				writeJSONError(w, http.StatusUnauthorized, "customer_not_provisioned")
				return
			}
			if customer.SuspendedAt != nil {
				writeJSONError(w, http.StatusForbidden, "account suspended")
				return
			}

			ctx := context.WithValue(r.Context(), CustomerIDKey, customer.ID)
			ctx = context.WithValue(ctx, IsAdminKey, customer.IsAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetCustomerID(ctx context.Context) string {
	id, _ := ctx.Value(CustomerIDKey).(string)
	return id
}
```

- [ ] **Step 5: รันเทสต์ต้องผ่าน**

```bash
cd backend && go test ./internal/middleware/ -run TestJWTAuth -v
```

Expected: PASS ทุกเคส

- [ ] **Step 6: เพิ่ม config**

ใน `backend/internal/config/config.go` เพิ่มบรรทัดใกล้ ๆ กับ `GoogleClientID`:

```go
	NeonAuthURL string `env:"NEON_AUTH_URL,required"`
```

**ยังไม่ลบ `JWT_SECRET` และตัวอื่นใน task นี้** — ของเก่ายังถูกใช้อยู่ Task 7 ค่อยเก็บกวาด

- [ ] **Step 7: ต่อสายใน router**

ใน `backend/internal/router/router.go` — บริเวณที่สร้าง service (ประมาณบรรทัด 82) เพิ่ม:

```go
	jwks, err := keyfunc.NewDefaultCtx(context.Background(),
		[]string{cfg.NeonAuthURL + "/.well-known/jwks.json"})
	if err != nil {
		return nil, fmt.Errorf("โหลดกุญแจของ Neon Auth ไม่ได้: %w", err)
	}
	issuer := cfg.NeonAuthURL[:strings.Index(cfg.NeonAuthURL, "/", len("https://"))]
```

แล้วเปลี่ยนบรรทัด 156 จาก `r.Use(middleware.JWTAuth(cfg.JWTSecret))` เป็น:

```go
				r.Use(middleware.JWTAuth(jwks.Keyfunc, issuer, customerRepo.GetByAuthUserID))
```

ถ้าฟังก์ชันที่สร้าง router ยังไม่คืน `error` ให้แก้ให้คืน แล้วแก้จุดที่เรียกมันใน
`backend/cmd/server/main.go` ให้จัดการ error ตามรูปแบบที่ไฟล์นั้นใช้อยู่

**ทำไมต้อง fail ตอนเริ่มระบบ:** ถ้าดึงกุญแจไม่ได้แปลว่าไม่มีใคร login ได้เลย ปล่อยให้ระบบ
ขึ้นมาแล้วปฏิเสธทุกคนเงียบ ๆ แย่กว่าไม่ขึ้นเลย

- [ ] **Step 8: ยืนยันว่าทั้งระบบยังคอมไพล์และเทสต์ผ่าน**

```bash
cd backend && go build ./... && go test ./internal/...
```

Expected: exit 0 ทั้งสองคำสั่ง (เช็ค exit code ตรง ๆ)

- [ ] **Step 9: Commit**

```bash
git add backend/internal/middleware/ backend/internal/config/config.go \
        backend/internal/router/router.go backend/cmd/server/main.go \
        backend/go.mod backend/go.sum
git commit -m "feat(auth): middleware ตรวจ JWT ของ Neon ผ่าน JWKS + อ่านสิทธิ์จากฐานข้อมูล"
```

---

### Task 4: endpoint `POST /auth/session` สร้างหรือผูก customer

**Files:**
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/internal/router/router.go:136-146`
- Test: `backend/internal/service/auth_service_test.go`

**Interfaces:**
- Consumes: `CustomerRepository.GetByAuthUserID` (Task 2) ·
  `middleware.CustomerIDKey` ไม่ใช้ใน task นี้เพราะ endpoint นี้อยู่นอกวง middleware
- Produces:
  - `func (s *AuthService) ProvisionCustomer(ctx context.Context, in ProvisionInput) (*domain.Customer, error)`
  - `type ProvisionInput struct { AuthUserID, Email, Name string; EmailVerified bool }`
  - `var ErrEmailNotVerified = errors.New("email not verified")`
  - route `POST /api/v1/auth/session` → คืน `APIResponse{Data: customer}`
  → Task 6 เรียกใช้

**หมายเหตุ:** endpoint นี้ต้องตรวจ JWT เหมือนกันแต่**ห้ามใช้ `JWTAuth` ตัวเดิม** เพราะตัวนั้นจะ
ปฏิเสธด้วย `customer_not_provisioned` ก่อนที่เราจะได้สร้าง customer ให้ — ไก่กับไข่
วิธีแก้: handler ตัวนี้อ่านและตรวจ token เอง โดยรับ `jwt.Keyfunc` กับ issuer เข้ามาทาง
constructor เหมือนที่ router ส่งให้ middleware

- [ ] **Step 1: เขียนเทสต์ที่ล้ม**

เพิ่มลงท้าย `backend/internal/service/auth_service_test.go` (ทำตามรูปแบบ mock ที่ไฟล์นี้ใช้อยู่)

```go
func TestProvisionCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("สร้างบัญชีใหม่เมื่อยังไม่เคยมี", func(t *testing.T) {
		repo := newMockCustomerRepo()
		svc := NewAuthService(repo, newMockRefreshTokenRepo(), testConfig())

		got, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-1", Email: "new@example.com", Name: "คนใหม่", EmailVerified: true,
		})

		require.NoError(t, err)
		require.NotNil(t, got.AuthUserID)
		assert.Equal(t, "neon-1", *got.AuthUserID)
		assert.Equal(t, domain.PlanSandbox, got.Plan)
		assert.True(t, strings.HasPrefix(got.APIKey, "pk_"), "ต้องได้ API key ตั้งแต่แรก")
		assert.Equal(t, 7, got.RetentionDays)
	})

	t.Run("ผูกกับบัญชีเดิมที่ email ตรงกัน ไม่สร้างซ้ำ", func(t *testing.T) {
		repo := newMockCustomerRepo()
		existing := &domain.Customer{
			ID: "cust-เดิม", Email: "old@example.com", APIKey: "pk_ของเดิม", Plan: domain.PlanPaid,
		}
		repo.seed(existing)
		svc := NewAuthService(repo, newMockRefreshTokenRepo(), testConfig())

		got, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-2", Email: "old@example.com", Name: "คนเดิม", EmailVerified: true,
		})

		require.NoError(t, err)
		assert.Equal(t, "cust-เดิม", got.ID, "ต้องเป็นบัญชีเดิม ไม่ใช่บัญชีใหม่")
		assert.Equal(t, "pk_ของเดิม", got.APIKey, "API key เดิมต้องไม่หาย")
		assert.Equal(t, domain.PlanPaid, got.Plan, "แพลนเดิมต้องไม่ถูกรีเซ็ต")
		assert.Equal(t, 1, repo.createCalls, "ห้ามสร้างบัญชีใหม่")
	})

	t.Run("เรียกซ้ำได้ผลเดิม ไม่สร้างซ้ำ", func(t *testing.T) {
		repo := newMockCustomerRepo()
		svc := NewAuthService(repo, newMockRefreshTokenRepo(), testConfig())
		in := ProvisionInput{AuthUserID: "neon-3", Email: "a@example.com", EmailVerified: true}

		first, err := svc.ProvisionCustomer(ctx, in)
		require.NoError(t, err)
		second, err := svc.ProvisionCustomer(ctx, in)
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, 1, repo.createCalls)
	})

	t.Run("email ที่ยังไม่ยืนยันต้องไม่ผูกกับบัญชีเดิม", func(t *testing.T) {
		repo := newMockCustomerRepo()
		repo.seed(&domain.Customer{ID: "cust-เหยื่อ", Email: "victim@example.com"})
		svc := NewAuthService(repo, newMockRefreshTokenRepo(), testConfig())

		_, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-คนร้าย", Email: "victim@example.com", EmailVerified: false,
		})

		assert.ErrorIs(t, err, ErrEmailNotVerified,
			"ไม่งั้นคนร้ายสมัคร email ปลอมแล้วสวมบัญชีคนอื่นได้")
	})

	t.Run("บัญชีถูกระงับต้องเข้าไม่ได้", func(t *testing.T) {
		repo := newMockCustomerRepo()
		suspended := time.Now()
		repo.seed(&domain.Customer{ID: "cust-ระงับ", Email: "s@example.com", SuspendedAt: &suspended})
		svc := NewAuthService(repo, newMockRefreshTokenRepo(), testConfig())

		_, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-4", Email: "s@example.com", EmailVerified: true,
		})

		assert.ErrorIs(t, err, ErrAccountSuspended)
	})
}
```

ถ้าไฟล์เทสต์เดิมยังไม่มี `newMockCustomerRepo` / `seed` / `createCalls` / `testConfig` /
`newMockRefreshTokenRepo` ให้เขียนเพิ่มโดยดูรูปแบบ mock ที่ไฟล์นี้ใช้อยู่แล้ว
อย่าสร้างรูปแบบใหม่ขึ้นมาเอง

**หมายเหตุ signature:** ตอนทำ task นี้ `NewAuthService` ยังรับ 3 พารามิเตอร์
(`customerRepo, refreshTokenRepo, cfg`) เพราะระบบเก่ายังไม่ถูกลบ — Task 6 จะลดเหลือ 2 ตัว
แล้วต้องกลับมาแก้บรรทัดที่เรียกในเทสต์ชุดนี้ด้วย

- [ ] **Step 2: รันแล้วต้องเห็นมันล้ม**

```bash
cd backend && go test ./internal/service/ -run TestProvisionCustomer -v
```

Expected: FAIL — `undefined: ProvisionInput` และ `svc.ProvisionCustomer undefined`
**แนบ output ที่ล้มจริง**

- [ ] **Step 3: เขียน implementation ใน service**

เพิ่มลงใน `backend/internal/service/auth_service.go`:

```go
var ErrEmailNotVerified = errors.New("email not verified")

type ProvisionInput struct {
	AuthUserID    string
	Email         string
	Name          string
	EmailVerified bool
}

// ProvisionCustomer หา customer ที่ผูกกับ user ของ Neon อยู่แล้ว
// ถ้ายังไม่มี ให้ผูกกับบัญชีเดิมที่ email ตรงกัน หรือสร้างใหม่
func (s *AuthService) ProvisionCustomer(ctx context.Context, in ProvisionInput) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByAuthUserID(ctx, in.AuthUserID)
	if err != nil {
		return nil, fmt.Errorf("get by auth user id: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		return customer, nil
	}

	if !in.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	customer, err = s.customerRepo.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("get by email: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		customer.AuthUserID = &in.AuthUserID
		if err := s.customerRepo.Update(ctx, customer); err != nil {
			return nil, fmt.Errorf("link auth user id: %w", err)
		}
		return customer, nil
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	customer = &domain.Customer{
		Email:         in.Email,
		AuthUserID:    &in.AuthUserID,
		Name:          in.Name,
		APIKey:        apiKey,
		Plan:          domain.PlanSandbox,
		RetentionDays: 7,
	}
	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}
	return customer, nil
}
```

- [ ] **Step 4: รันเทสต์ต้องผ่าน**

```bash
cd backend && go test ./internal/service/ -run TestProvisionCustomer -v
```

Expected: PASS ทุกเคส

- [ ] **Step 5: เขียน handler**

เพิ่มลงใน `backend/internal/handler/auth_handler.go` และเพิ่มฟิลด์
`jwks jwt.Keyfunc` กับ `issuer string` เข้า struct `AuthHandler` พร้อมรับเข้ามาทาง
`NewAuthHandler` (แก้จุดที่เรียกใน router ให้ส่งค่าเดียวกับที่ส่งให้ middleware):

```go
func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == r.Header.Get("Authorization") {
		ErrorJSON(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	token, err := jwt.Parse(tokenStr, h.jwks,
		jwt.WithIssuer(h.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"EdDSA"}),
	)
	if err != nil || !token.Valid {
		ErrorJSON(w, http.StatusUnauthorized, "invalid token")
		return
	}

	claims, _ := token.Claims.(jwt.MapClaims)
	authUserID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	emailVerified, _ := claims["emailVerified"].(bool) // camelCase — spike ยืนยันแล้ว

	if authUserID == "" || email == "" {
		ErrorJSON(w, http.StatusUnauthorized, "invalid claims")
		return
	}

	customer, err := h.authService.ProvisionCustomer(r.Context(), service.ProvisionInput{
		AuthUserID: authUserID, Email: email, Name: name, EmailVerified: emailVerified,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailNotVerified) {
			ErrorJSON(w, http.StatusForbidden, "email not verified")
			return
		}
		if errors.Is(err, service.ErrAccountSuspended) {
			ErrorJSON(w, http.StatusForbidden, "account suspended")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "provision failed", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: customer})
}
```

**ชื่อ claim ยืนยันจาก token จริงแล้ว** (ดูหัวข้อ "ผล Spike" ท้ายไฟล์): `sub` `email` `name`
`emailVerified` — **`emailVerified` เป็น camelCase ไม่ใช่ `email_verified`** ถ้าเขียนผิดจะได้
`false` เสมอแล้วไม่มีใคร provision ได้เลย โดยไม่มี error ฟ้อง

- [ ] **Step 6: เพิ่ม route**

ใน `backend/internal/router/router.go` ในบล็อก `r.Route("/auth", ...)` เพิ่ม:

```go
				r.Post("/session", authHandler.Session)
```

- [ ] **Step 7: ยืนยันทั้งระบบ**

```bash
cd backend && go build ./... && go test ./internal/...
```

Expected: exit 0 ทั้งสองคำสั่ง

- [ ] **Step 8: Commit**

```bash
git add backend/internal/service/ backend/internal/handler/ backend/internal/router/router.go
git commit -m "feat(auth): เพิ่ม POST /auth/session สร้างหรือผูก customer กับ user ของ Neon"
```

---

### Task 5: frontend เปลี่ยนไปใช้ Neon Auth

**Files:**
- Create: `frontend/src/lib/neon-auth.ts`
- Modify: `frontend/src/pages/auth/LoginPage.tsx:165-200`
- Modify: `frontend/src/lib/api.ts` (เขียนใหม่ทั้งไฟล์ 177 → ~55 บรรทัด)
- Modify: `frontend/src/stores/auth-store.ts:20-24`
- Modify: `frontend/src/App.tsx` (เอา route `/auth/callback` ออก)
- Delete: `frontend/src/pages/auth/AuthCallbackPage.tsx`
- Modify: `frontend/package.json`

**Interfaces:**
- Consumes: `POST /api/v1/auth/session` จาก Task 4
- Produces: `authClient` จาก `@/lib/neon-auth` → ใช้ในหน้า login และใน `api.ts`

- [ ] **Step 1: ติดตั้ง SDK และถอดของเก่า**

```bash
cd frontend && npm install @neondatabase/neon-js && npm uninstall @react-oauth/google
```

- [ ] **Step 2: สร้าง client**

```typescript
// frontend/src/lib/neon-auth.ts
import { createAuthClient } from '@neondatabase/neon-js/auth'

const authURL = import.meta.env.VITE_NEON_AUTH_URL
if (!authURL) {
  throw new Error('ไม่ได้ตั้งค่า VITE_NEON_AUTH_URL')
}

export const authClient = createAuthClient(authURL)

// ---------------------------------------------------------------------------
// JWT ที่ backend ตรวจได้ **ไม่ใช่** ค่าที่ getSession() คืนมา
// getSession().session.token เป็น session token ทึบ ๆ ที่ตรวจกับ JWKS ไม่ได้
// ตัว JWT จริงต้องขอจาก endpoint /token แยกต่างหาก (spike ยืนยันแล้ว)
// และมันอายุแค่ 15 นาที จึงต้อง cache ไว้แล้วขอใหม่ก่อนหมดอายุ
// ---------------------------------------------------------------------------
let cached: { token: string; expiresAt: number } | null = null

export async function getAccessToken(): Promise<string | null> {
  // เผื่อเวลาไว้ 60 วินาที กัน token หมดอายุระหว่างทางไปถึง backend
  if (cached && Date.now() < cached.expiresAt - 60_000) {
    return cached.token
  }

  const res = await fetch(`${authURL}/token`, { credentials: 'include' })
  if (!res.ok) {
    cached = null
    return null
  }

  const { token } = (await res.json()) as { token: string }
  const payload = JSON.parse(atob(token.split('.')[1])) as { exp: number }
  cached = { token, expiresAt: payload.exp * 1000 }
  return token
}

export function clearAccessToken() {
  cached = null
}
```

**`credentials: 'include'` จำเป็น** เพราะ session อยู่ในคุกกี้ของโดเมน Neon ซึ่งเป็นคนละโดเมนกับเรา
→ โดเมนของเราต้องถูกใส่ใน `trusted_origins` ฝั่ง Neon ก่อน ไม่งั้น CORS บล็อก (ทำใน Task 8)

- [ ] **Step 3: เปลี่ยนปุ่มในหน้า login**

ใน `frontend/src/pages/auth/LoginPage.tsx` ลบ `import { GoogleLogin } from '@react-oauth/google'`
กับตัวแปร `googleClientId` แล้วแทนบล็อก `{googleClientId ? ... : ...}` (บรรทัด 165-185) ด้วย:

```tsx
            <div className="flex justify-center">
              <button
                type="button"
                onClick={() =>
                  authClient.signIn.social({
                    provider: 'google',
                    callbackURL: window.location.origin + '/dashboard',
                  })
                }
                className="flex h-11 w-[320px] items-center justify-center gap-2 rounded-md bg-foreground text-sm font-medium text-background transition-opacity hover:opacity-90"
              >
                เข้าสู่ระบบด้วย Google
              </button>
            </div>
```

พร้อม `import { authClient } from '@/lib/neon-auth'`

**คำสั่ง `authClient.signIn.social` ต้องตรงกับที่ Spike Step 6 ยืนยันไว้ ถ้าต่างให้แก้ตรงนี้**

- [ ] **Step 4: เขียน `api.ts` ใหม่**

เขียนทับทั้งไฟล์ — ตัด logic ต่ออายุ token ทั้งชุดทิ้ง เพราะ Neon จัดการให้แล้ว:

```typescript
import axios from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { authClient, getAccessToken, clearAccessToken } from '@/lib/neon-auth'

const API_BASE = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: `${API_BASE}/api/v1`,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use(async (config) => {
  const token = await getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status
    const message = error.response?.data?.error

    // ยังไม่มี customer ในระบบเรา — ลงทะเบียนแล้วลองใหม่หนึ่งครั้ง
    if (status === 401 && message === 'customer_not_provisioned' && !error.config._retried) {
      error.config._retried = true
      const { data } = await api.post('/auth/session')
      useAuthStore.getState().setCustomer(data.data)
      return api.request(error.config)
    }

    if (status === 401) {
      clearAccessToken()
      await authClient.signOut()
      useAuthStore.getState().logout()
      window.location.href = '/login'
      return Promise.reject(error)
    }

    if (status === 403 && message === 'account suspended') {
      toast.error('บัญชีนี้ถูกระงับการใช้งาน')
    }

    return Promise.reject(error)
  }
)

export default api
```

**ถ้าไฟล์เดิม export อย่างอื่นนอกจาก `default` ให้เก็บ export เหล่านั้นไว้ด้วย**
ตรวจด้วย `grep -rn "from '@/lib/api'" frontend/src` ก่อนเขียนทับ

- [ ] **Step 5: แก้ auth-store**

ใน `frontend/src/stores/auth-store.ts` เปลี่ยน `logout` เป็น:

```typescript
      logout: () => set({ customer: null, isAuthenticated: false }),
```

(Neon เก็บ session เอง ไม่มี token ของเราให้ลบแล้ว)

- [ ] **Step 6: ลบหน้า callback**

```bash
rm frontend/src/pages/auth/AuthCallbackPage.tsx
```

แล้วเอา route `/auth/callback` กับ import ที่เกี่ยวข้องออกจาก `frontend/src/App.tsx`

- [ ] **Step 7: ยืนยันว่า build ผ่าน**

```bash
cd frontend && npx tsc --noEmit; echo "typecheck exit=$?"
npm run build; echo "build exit=$?"
```

Expected: exit=0 ทั้งคู่ · **ต้องอ่านตัวเลข exit code จริง ห้ามดูแค่ข้อความสุดท้าย**

- [ ] **Step 8: ยืนยันว่าไม่เหลือร่องรอยของเก่า**

```bash
grep -rn "react-oauth\|access_token\|refresh_token\|VITE_GOOGLE_CLIENT_ID" frontend/src frontend/package.json
```

Expected: ไม่มีผลลัพธ์ ถ้ามีให้เก็บให้หมดก่อนไปต่อ

- [ ] **Step 9: Commit**

```bash
git add frontend/
git commit -m "feat(auth): frontend ใช้ Neon Auth แทน Google OAuth ตรง ๆ"
```

---

### Task 6: ลบระบบ token เดิมออกจาก Go

**Files:**
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/internal/router/router.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/repository/interfaces.go`
- Delete: `backend/internal/repository/postgres/refresh_token_repo.go`
- Delete: `backend/internal/repository/mocks/refresh_token.go`
- Create: `backend/db/migrations/000029_drop_legacy_auth.{up,down}.sql`
- Modify: `backend/internal/service/auth_service_test.go` · `handler/auth_handler_test.go`
- Modify: `backend/go.mod`

**Interfaces:**
- Consumes: ทุกอย่างจาก Task 3 และ 4 ต้องเสร็จก่อน
- Produces: ไม่มีของใหม่ — task นี้ลบอย่างเดียว

- [ ] **Step 1: ลบ method ออกจาก service**

ใน `backend/internal/service/auth_service.go` ลบ `GoogleAuth()` · `GoogleAuthInput` ·
`DevLogin()` · `RefreshTokens()` · `generateTokens()` · `hashToken()` · `AuthTokens` ·
`ErrInvalidRefreshToken` · `ErrInvalidGoogleToken` · ฟิลด์ `refreshTokenRepo` ใน struct
และพารามิเตอร์ตัวนั้นใน `NewAuthService` · import `github.com/golang-jwt/jwt/v5`,
`google.golang.org/api/idtoken`, `crypto/sha256`, `time`

เหลือ: `GetCustomerByID` · `ProvisionCustomer` · `RegenerateAPIKey` · `Logout`
→ **`Logout` ลบด้วย** เพราะมันลบ refresh token ซึ่งไม่มีแล้ว

- [ ] **Step 2: ลบ handler และ route**

ใน `handler/auth_handler.go` ลบ `GoogleAuth` · `GoogleAuthCallback` · `DevLogin` ·
`Refresh` · `Logout` · ฟิลด์ `cfg` ถ้าไม่มีใครใช้แล้ว

ใน `router/router.go` ลบ 5 route: `/auth/google` · `/auth/google/callback` ·
`/auth/refresh` · `/auth/dev-login` · `/auth/logout` และตัวแปร `refreshTokenRepo`

**เรื่อง `/auth/dev-login`:** ทางลัด login ตอนพัฒนาหายไปพร้อมกัน ต่อจากนี้การ login ในเครื่อง
ให้ใช้ Google จริงผ่าน Neon Auth (branch ของฐานข้อมูลแต่ละอันมี auth environment แยกกัน
จึงทดสอบได้โดยไม่แตะผู้ใช้จริง) ตามที่ spec หัวข้อ "การ login ตอนพัฒนาในเครื่อง" ระบุไว้
**ห้ามสร้างทางลัดใหม่ขึ้นมาแทน** — ทางลัดที่ข้ามการยืนยันตัวตนคือช่องโหว่ถ้าหลุดขึ้น production

- [ ] **Step 3: ลบ repository**

```bash
rm backend/internal/repository/postgres/refresh_token_repo.go \
   backend/internal/repository/mocks/refresh_token.go
```

แล้วลบ `RefreshTokenRepository` ออกจาก `backend/internal/repository/interfaces.go`

- [ ] **Step 4: ลบ config ที่ไม่ใช้แล้ว**

ใน `backend/internal/config/config.go` ลบ `JWTSecret` · `JWTAccessTTL` · `JWTRefreshTTL` ·
`GoogleClientID`

- [ ] **Step 5: ลบเทสต์ของฟังก์ชันที่ไม่มีแล้ว**

ใน `auth_service_test.go` และ `auth_handler_test.go` ลบเฉพาะเทสต์ของ `GoogleAuth` ·
`DevLogin` · `RefreshTokens` · `Logout` · `generateTokens`
**ห้ามลบเทสต์ของ `ProvisionCustomer` · `GetCustomerByID` · `RegenerateAPIKey`**

- [ ] **Step 6: เก็บ dependency ที่ไม่ใช้**

```bash
cd backend && go mod tidy
```

- [ ] **Step 7: ยืนยันว่าคอมไพล์และเทสต์ผ่าน**

```bash
cd backend && go build ./...; echo "build exit=$?"
go test ./internal/...; echo "test exit=$?"
```

Expected: exit=0 ทั้งคู่

- [ ] **Step 8: ยืนยันว่าไม่เหลือร่องรอย**

```bash
grep -rn "JWT_SECRET\|JWTSecret\|idtoken\|refreshTokenRepo\|GoogleClientID\|dev-login" backend/ --include="*.go"
grep -rn "google.golang.org/api" backend/go.mod
```

Expected: ไม่มีผลลัพธ์เลยทั้งสองคำสั่ง

- [ ] **Step 9: เขียน migration ทิ้งของเก่าในฐานข้อมูล**

```sql
-- backend/db/migrations/000029_drop_legacy_auth.up.sql
DROP TABLE IF EXISTS refresh_tokens;
ALTER TABLE customers DROP COLUMN IF EXISTS google_id;
```

```sql
-- backend/db/migrations/000029_drop_legacy_auth.down.sql
ALTER TABLE customers ADD COLUMN IF NOT EXISTS google_id VARCHAR(255) UNIQUE;
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

โครงตารางข้างบนคัดมาจาก `000001_init_schema.up.sql:78-84` ตรงตัว
**ส่วนนิยามของ `google_id` ให้เปิด `000001_init_schema.up.sql` ยืนยันชนิดและ constraint จริง
ก่อนเขียน** ตรงนี้ยังไม่ได้ตรวจ

จากนั้นลบ `GoogleID` ออกจาก `domain/customer.go` · `GetByGoogleID` ออกจาก interface,
postgres repo, mock · และทุก `SELECT`/`INSERT`/`UPDATE` ที่อ้าง `google_id`

- [ ] **Step 10: ยืนยันอีกรอบแล้ว commit**

```bash
cd backend && go build ./...; echo "build exit=$?"
go test ./internal/...; echo "test exit=$?"
grep -rn "google_id\|GoogleID" backend/ --include="*.go"
```

Expected: exit=0 ทั้งคู่ · grep ไม่มีผลลัพธ์

```bash
git add -A backend/
git commit -m "refactor(auth): รื้อระบบ JWT + refresh token ของเราทิ้งทั้งชุด"
```

---

### Task 7: ตั้งค่า Cloudflare และ CSP

**Files:**
- Modify: `wrangler.jsonc:18-22`
- Modify: `worker/index.ts` (รายการ `envVars`)
- Modify: `worker/headers.ts:3-7`
- Test: `worker/headers.test.ts`

**Interfaces:**
- Consumes: `NEON_AUTH_URL` จาก Task 1
- Produces: `NEON_AUTH_URL` ส่งถึง container จริง · CSP ที่ยอมให้เบราว์เซอร์ต่อ Neon Auth

- [ ] **Step 1: เขียนเทสต์ CSP ที่ล้ม**

เพิ่มลงใน `worker/headers.test.ts` (ทำตามรูปแบบ 5 เคสที่มีอยู่แล้วในไฟล์):

```typescript
  it("CSP ยอมให้ต่อ Neon Auth และไม่เหลือ Google Sign-In", () => {
    const csp = headersFor("/dashboard").get("Content-Security-Policy")!;
    expect(csp).toContain("*.neon.tech");
    expect(csp).not.toContain("accounts.google.com");
  });
```

(`headersFor` เป็น helper ที่มีอยู่แล้วบรรทัด 4 ของไฟล์ ห่อ `applySecurityHeaders`)

- [ ] **Step 2: รันแล้วต้องเห็นมันล้ม**

```bash
npm test -- worker/headers.test.ts
```

Expected: FAIL — ยังไม่มี `*.neon.tech` ใน CSP

- [ ] **Step 3: แก้ CSP**

ใน `worker/headers.ts` บรรทัด 3-7 — เอา `accounts.google.com` ออกจาก `script-src`,
`style-src`, `connect-src`, `frame-src` แล้วเพิ่ม `*.neon.tech` เข้า `connect-src`

**ห้ามแตะ `connect.facebook.net` และ `js.stripe.com`** — เป็นของ sale page กับระบบชำระเงิน
คนละเรื่องกับ login

- [ ] **Step 4: รันเทสต์ต้องผ่านครบ**

```bash
npm test -- worker/headers.test.ts
```

Expected: PASS ทั้ง 6 เคส (5 เคสเดิม + เคสใหม่)

- [ ] **Step 5: แก้ wrangler.jsonc**

ใน `vars` ลบ `JWT_REFRESH_TTL` แล้วเพิ่ม `NEON_AUTH_URL` ด้วยค่าจริงจาก Spike:

```jsonc
  "vars": {
    "NEON_AUTH_URL": "<ค่าจาก Task 1 Step 7>",
    "DB_MAX_CONNS": "10",
    "DB_MIN_CONNS": "2"
  },
```

- [ ] **Step 6: เพิ่มเข้ารายการที่ส่งต่อถึง container**

ใน `worker/index.ts` เพิ่ม `'NEON_AUTH_URL'` เข้า `envVars` แล้วเอา `'JWT_SECRET'` ·
`'GOOGLE_CLIENT_ID'` · `'JWT_REFRESH_TTL'` ออก

- [ ] **Step 7: regenerate types แล้วยืนยัน**

```bash
npx wrangler types; echo "types exit=$?"
npm run typecheck; echo "typecheck exit=$?"
npm test; echo "test exit=$?"
npx wrangler deploy --dry-run; echo "dryrun exit=$?"
```

Expected: exit=0 ทุกคำสั่ง

- [ ] **Step 8: Commit**

```bash
git add wrangler.jsonc worker/ worker-configuration.d.ts
git commit -m "chore(worker): ส่ง NEON_AUTH_URL ถึง container + CSP รับ Neon Auth"
```

---

### Task 8: ขึ้น production และทดสอบจริง

**Files:** ไม่มีการแก้โค้ด — งานตั้งค่าและยืนยันผล

**Interfaces:**
- Consumes: ทุก task ก่อนหน้าต้องเสร็จ

**⚠️ ขั้นตอนนี้แตะ production ต้องได้รับอนุญาตจากเจ้าของก่อนทุก step ที่เขียนข้อมูล**

- [ ] **Step 1: เปิด Neon Auth บนฐานข้อมูล production**

ทำแบบเดียวกับ Task 1 Step 2 แต่บน branch production
คราวนี้**ต้องใช้ OAuth app ของเราเอง ไม่ใช่ credential กลางสำหรับทดสอบ** — สร้างใน Google Cloud
Console แบบ Web application แล้วใส่ redirect URI `{NEON_AUTH_URL}/callback/google`
Google ต้องการให้ publish หน้า consent screen สำหรับใช้งานจริง ซึ่งอาจใช้เวลาตรวจสอบ 2-3 วันทำการ
**ถ้าติดขั้นนี้ให้แจ้งเจ้าของทันที อย่ารอเงียบ ๆ**

- [ ] **Step 1b: ตั้ง `trusted_origins` และปิด email/password**

ใช้ `configure_neon_auth` (Neon MCP) บน branch production:

- `trusted_origins` = `["https://pixlinks.xyz"]` — **ถ้าไม่ตั้ง เบราว์เซอร์จะถูก CORS บล็อก
  ตอน `fetch(/token)` แล้ว login ไม่ผ่านทั้งที่ทุกอย่างถูก** (ค่าตั้งต้นเป็น `[]`)
- `auth_methods.email_password.enabled` = `false` — spec บอกว่าใช้ Google อย่างเดียว
  ปล่อยไว้เท่ากับเปิดทางให้ใครก็ได้สมัครบัญชีด้วย email/password โดยไม่ต้องยืนยัน email
  (`require_email_verification: false` เป็นค่าตั้งต้น) ซึ่งไม่มีใครขอ

ยืนยันด้วย `get_neon_auth_config` ว่าทั้งสองค่าเปลี่ยนจริง

- [ ] **Step 2: อัปเดต `NEON_AUTH_URL` ใน `wrangler.jsonc` เป็นค่าของ production**

- [ ] **Step 3: ลบ secret ที่ไม่ใช้แล้ว**

```bash
npx wrangler secret delete JWT_SECRET
npx wrangler secret delete GOOGLE_CLIENT_ID
npx wrangler secret list
```

Expected: เหลือ 16 ตัว

- [ ] **Step 4: รัน migration บน production**

ขออนุญาตเจ้าของก่อน แล้วรัน `000028` และ `000029` ตามลำดับ
ยืนยันหลังรัน: `\d customers` ต้องมี `auth_user_id` และไม่มี `google_id` ·
`\dt refresh_tokens` ต้องไม่พบ

- [ ] **Step 5: merge เข้า main ให้ CI deploy**

CI จะ deploy ให้เองเมื่อ push เข้า main (`.github/workflows/ci.yml` job `deploy`)

- [ ] **Step 6: ยืนยัน security header ยังครบ**

```bash
curl -sI https://pixlinks.xyz/ | grep -ci "content-security-policy\|strict-transport-security\|x-frame-options\|x-content-type-options\|referrer-policy\|permissions-policy\|cross-origin-opener-policy"
```

Expected: `7`

- [ ] **Step 7: ให้เจ้าของ login จริงหนึ่งครั้ง**

**Claude ทำแทนไม่ได้ ต้องใช้บัญชี Google ของเจ้าของ**
ขอให้เจ้าของเปิด `https://pixlinks.xyz/login` → กด "เข้าสู่ระบบด้วย Google" → ต้องเข้า
dashboard ได้ และเห็นข้อมูลบัญชีตัวเองครบ

หลังจากนั้นยืนยันจากฝั่งเรา:
- `SELECT id, email, auth_user_id FROM customers WHERE auth_user_id IS NOT NULL;`
  ต้องมีแถวของเจ้าของ และ **`api_key` กับ `plan` ต้องเป็นค่าเดิม ไม่ถูกรีเซ็ต**
- ดู log ของ Worker ว่าไม่มี error

- [ ] **Step 8: ทดสอบว่าการระงับบัญชีมีผลทันที**

ขออนุญาตเจ้าของ แล้วตั้ง `suspended_at` ของบัญชีทดสอบ (ไม่ใช่บัญชีเจ้าของ) → เรียก API →
ต้องได้ 403 ทันทีโดยไม่ต้องรอ token หมดอายุ → แล้วคืนค่าเป็น NULL

- [ ] **Step 9: บันทึกผลลง ledger**

เขียนสรุปลง `.superpowers/sdd/2026-08-12-neon-auth-migration/progress.md`:
อะไรที่ยืนยันด้วยการรันจริง อะไรที่ยังไม่ได้ยืนยัน และค่าจริงทั้งหมดที่ใช้

---

## ผล Spike (รันจริง 2026-08-12 · Claude)

**สรุป: ผ่าน — Go ตรวจ JWT ของ Neon ได้จริง** เดินหน้าทั้งแผนต่อได้

สภาพแวดล้อมทดสอบ: Neon project `pixlinks` (`shy-hall-58617360`) branch `neon-auth-spike`
(`br-snowy-unit-aiesq0jb`) endpoint `ep-ancient-brook-ain4f1ec` — **ไม่ใช่ production
(`ep-cool-butterfly-ai3qizfi`)** · ตั้งให้หมดอายุเอง 2026-08-26

`NEON_AUTH_URL` ของ branch ทดสอบ:
`https://ep-ancient-brook-ain4f1ec.neonauth.c-4.us-east-1.aws.neon.tech/neondb/auth`

### สิ่งที่พิสูจน์แล้วด้วยการรันจริง

1. `keyfunc.NewDefaultCtx` ดึงและแปลงกุญแจของ Neon ได้ → ได้ `ed25519.PublicKey` 1 ดอก
2. `golang-jwt/v5` เซ็นและตรวจ EdDSA ได้
3. JWT จริงจาก Neon ผ่านการตรวจกับ JWKS ครบ (exit code 0)

### 5 อย่างที่ต่างจากที่แผนเดาไว้ — แก้ในแผนแล้วทั้งหมด

| # | แผนเดิมเดาว่า | ของจริง | แก้ที่ |
|---|---|---|---|
| 1 | ลายเซ็น RS256 | **EdDSA / Ed25519** (`kty=OKP`) | Task 3 (เทสต์ + `WithValidMethods`) · Task 4 |
| 2 | claim ชื่อ `email_verified` | **`emailVerified`** (camelCase) | Task 4 Step 5 |
| 3 | JWT มากับ `getSession()` | **ต้องขอจาก `GET {base}/token` แยก** — ค่าใน `getSession()` เป็น session token ทึบ ตรวจกับ JWKS ไม่ได้ | Task 5 (เพิ่ม `getAccessToken()`) |
| 4 | ไม่ได้คิดเรื่องอายุ token | **JWT อายุ 900 วินาที (15 นาที)** | Task 5 (cache + ขอใหม่ก่อนหมด 60 วิ) |
| 5 | ไม่ได้คิดเรื่อง CORS | ทุก request ต้องมี `Origin` และโดเมนต้องอยู่ใน `trusted_origins` ซึ่งตั้งต้นเป็น `[]` | Task 8 Step 1b |

**ข้อ 3 คือข้อที่อันตรายที่สุด** — คู่มือของ Neon เองแนะนำ `data.session.token` ซึ่งใช้ไม่ได้กับ
backend แยก ถ้าเดินตามคู่มือจะได้ 401 ทุก request แล้วไล่หาสาเหตุผิดทางนาน

### claim ทั้งหมดที่ Neon ส่งมาจริง

`sub` (= `id`) · `email` · `emailVerified` · `name` · `role` (`"authenticated"`) ·
`banned` · `banReason` · `banExpires` · `createdAt` · `updatedAt` · `iat` · `exp` ·
`iss` · `aud`

- `iss` = `https://ep-ancient-brook-ain4f1ec.neonauth.c-4.us-east-1.aws.neon.tech`
  → **ยืนยันว่าการคำนวณ issuer จาก origin ของ `NEON_AUTH_URL` ถูกต้อง**
- `aud` มีค่าเท่ากับ `iss`
- **ไม่มี claim ใดบอกสิทธิ์แอดมิน** → ยืนยันว่าการอ่าน `is_admin` จากฐานข้อมูลเป็นทางเดียวที่ทำได้

### สิ่งที่ยังไม่ได้พิสูจน์ (บอกตรง ๆ)

- **ยังไม่ได้ login ด้วย Google จริง** — token ที่ใช้ทดสอบมาจากการสมัคร email/password
  (`spike-test@example.com`) ซึ่งเดินเส้นทางการออก JWT เดียวกัน แต่ **ไม่ได้พิสูจน์ว่า OAuth
  กับ Google สำเร็จ** · จะพิสูจน์จริงตอน Task 5 (ในเครื่อง) และ Task 8 Step 7 (production)
- **ยังไม่ได้ยืนยันชื่อคำสั่ง `authClient.signIn.social`** ของ `@neondatabase/neon-js`
  → Task 5 Step 3 ต้องตรวจจากเอกสารของแพ็กเกจก่อนเขียน
- user ทดสอบ `spike-test@example.com` ยังค้างอยู่บน branch ทดสอบ — หายไปเองพร้อม branch
  วันที่ 2026-08-26
