package middleware

// หมายเหตุ: ชื่อฟังก์ชันเทสต์เป็น ASCII เพราะ Go ไม่ยอมรับสระ/วรรณยุกต์ไทย
// (Unicode category Mn เช่น ู ่ ิ) ใน identifier — คอมไพล์ไม่ผ่าน
// เจตนาของแต่ละเคสรักษาไว้ในคอมเมนต์ภาษาไทยเหมือน brief ตั้งใจ

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

// TokenถูกและมีCustomer
func TestJWTAuth_validTokenWithCustomer(t *testing.T) {
	key, keyFn := newTestKey(t)
	customer := &domain.Customer{ID: "cust-1", IsAdmin: true}

	rec, gotID, gotAdmin := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "cust-1", gotID, "ต้องใส่ id ของ customer ไม่ใช่ sub ของ Neon")
	assert.True(t, gotAdmin, "ต้องอ่าน is_admin จากฐานข้อมูล ไม่ใช่จาก claim")
}

// ไม่เชื่อ is_admin ที่ปลอมมาใน token
func TestJWTAuth_ignoresForgedIsAdmin(t *testing.T) {
	key, keyFn := newTestKey(t)
	claims := validClaims()
	claims["is_admin"] = true // คนร้ายยัดมาเอง
	customer := &domain.Customer{ID: "cust-1", IsAdmin: false}

	rec, _, gotAdmin := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, claims))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotAdmin, "claim ปลอมต้องไม่มีผล")
}

// บัญชีถูกระงับ
func TestJWTAuth_suspendedAccountForbidden(t *testing.T) {
	key, keyFn := newTestKey(t)
	suspended := time.Now()
	customer := &domain.Customer{ID: "cust-1", SuspendedAt: &suspended}

	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(customer),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ยังไม่มี Customer
func TestJWTAuth_customerNotProvisioned(t *testing.T) {
	key, keyFn := newTestKey(t)

	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(nil),
		makeToken(t, key, validClaims()))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "customer_not_provisioned",
		"frontend ใช้รหัสนี้แยกว่าต้องเรียก /auth/session ไม่ใช่เตะออกหน้า login")
}

// ปฏิเสธ token ที่ไม่ถูกต้อง
func TestJWTAuth_rejectsInvalidTokens(t *testing.T) {
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

// ไม่มี header
func TestJWTAuth_missingAuthorizationHeader(t *testing.T) {
	_, keyFn := newTestKey(t)
	rec, _, _ := runMiddleware(t, keyFn, lookupReturning(nil), "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
