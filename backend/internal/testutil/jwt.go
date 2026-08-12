// Package testutil ให้ helper สำหรับเทสต์ข้าม package.
// ไฟล์นี้ (jwt.go) ถูกคอมไพล์เสมอ — ต่างจาก pgtest.go/fixtures.go ที่ใช้
// build tag `integration` เพราะต้องต่อ Postgres จริง ส่วน helper ในไฟล์นี้
// เป็น in-memory ไม่ต้องต่อเน็ตหรือฐานข้อมูล
package testutil

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// TestAuthIssuer คือ issuer สำหรับเทสต์ — ต้องตรงกับที่ฝั่ง middleware ใช้ตรวจ
const TestAuthIssuer = "https://ep-test.neon.tech"

// TestAuth เก็บกุญแจ Ed25519 ในหน่วยความจำพร้อม registry ที่ผูก authUserID
// เข้ากับ customer ให้ handler test ทุกไฟล์ใช้ตัวเดียวกันได้ โดย map ของ customer
// โตขึ้นเรื่อย ๆ (ไม่ทับกันเมื่อ authUserID ต่างกัน) แก้ปัญหา "lookup ตัวเดียว
// ไม่รู้ isAdmin ของแต่ละเคส" เพราะแต่ละเคสเรียก MintToken ลงทะเบียน customer ของตัวเองก่อนยิง request
type TestAuth struct {
	priv      ed25519.PrivateKey
	pub       ed25519.PublicKey
	customers map[string]*domain.Customer
}

// NewTestAuth สุ่มกุญแจ Ed25519 ใหม่และสร้าง registry ว่าง
func NewTestAuth() *TestAuth {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("testutil: failed to generate Ed25519 key: " + err.Error())
	}
	return &TestAuth{priv: priv, pub: pub, customers: map[string]*domain.Customer{}}
}

// KeyFunc คืน jwt.Keyfunc ที่ยอมรับเฉพาะ token ที่เซ็นด้วยกุญแจส่วนตัวของ TestAuth นี้ (EdDSA)
func (ta *TestAuth) KeyFunc() jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return ta.pub, nil
	}
}

// MintToken เซ็น JWT (sub=authUserID, iss=TestAuthIssuer, exp=+1h) ด้วย EdDSA
// และผูก authUserID เข้ากับ customer ใน registry ในขั้นตอนเดียวกัน — ทำให้ token
// ที่สร้างกับ customer ที่ lookup จะคืนนั้นผูกกันแน่นอน
func (ta *TestAuth) MintToken(authUserID string, customer *domain.Customer) string {
	ta.customers[authUserID] = customer
	claims := jwt.MapClaims{
		"sub": authUserID,
		"iss": TestAuthIssuer,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(ta.priv)
	if err != nil {
		panic("testutil: failed to sign test JWT: " + err.Error())
	}
	return tokenStr
}

// MintNeonToken เซ็น JWT เหมือนที่ Neon Auth ออกให้จริง — caller ใส่ claim เองทั้งหมด
// (sub/email/name/emailVerified) เพื่อทดสอบ handler ที่อ่าน claim เอง เช่น Session
// ต่างจาก MintToken ที่สร้างแค่ sub/iss/exp (พอสำหรับ middleware ที่อ่านสิทธิ์จาก lookup)
// ใช้กุญแจเดียวกัน (ta.priv) จึงผ่าน KeyFunc ของ testAuth ตัวเดียวกัน
func (ta *TestAuth) MintNeonToken(claims jwt.MapClaims) string {
	if claims["iss"] == nil {
		claims["iss"] = TestAuthIssuer
	}
	if claims["exp"] == nil {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(ta.priv)
	if err != nil {
		panic("testutil: failed to sign test JWT: " + err.Error())
	}
	return tokenStr
}

// Lookup คืน customer ที่ผูกกับ authUserID (nil ถ้ายังไม่มี) signature ตรงกับ
// middleware.CustomerLookup ทำให้ส่ง ta.Lookup เข้า middleware.JWTAuth ได้ตรง ๆ
// โดย testutil ไม่ต้อง import middleware (Go ยอมรับ method value ที่ underlying type ตรงกัน)
func (ta *TestAuth) Lookup(_ context.Context, authUserID string) (*domain.Customer, error) {
	return ta.customers[authUserID], nil
}
