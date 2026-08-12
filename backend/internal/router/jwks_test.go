package router

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/config"
)

// startDummyJWKS ตั้ง httptest server ที่ตอบทุก path ด้วย status/body ที่กำหนด
// ใช้เป็น config.NeonAuthURL ในเทสต์ router โดยไม่ต้องพึ่ง endpoint ของ Neon Auth จริง
func startDummyJWKS(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// TestRouter_FailsWhenJWKSUnreachable พิสูจน์กลไก fail-at-startup:
// ถ้าดึง JWKS ไม่สำเร็จตอน New (ที่นี่ server ตอบ 404) New ต้องคืน error จริง ไม่ใช่กลืนเงียบๆ
// แล้วปล่อยให้ login พังทุกคนจนกว่า background refresh จะสำเร็จ (นานถึง 1 ชม.)
// ซึ่งเป็นสิ่งที่จะเกิดขึ้นถ้าใช้ default ของ library (NoErrorReturnFirstHTTPReq=true)
func TestRouter_FailsWhenJWKSUnreachable(t *testing.T) {
	bad := startDummyJWKS(http.StatusNotFound, `{}`)
	defer bad.Close()

	cfg := &config.Config{
		Env:              "test",
		NeonAuthURL:      bad.URL,
		RateLimitRPS:     1,
		DBQueryTimeout:   time.Second,
		SalePageCacheTTL: time.Minute,
	}
	_, _, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, context.Background())
	if err == nil {
		t.Fatal("New ต้องคืน error เมื่อดึง JWKS ไม่ได้ตอน startup (NoErrorReturnFirstHTTPReq=false) — " +
			"ไม่ใช่กลืนเงียบๆ แล้วปล่อยให้ทุก token fail จนกว่า refresh goroutine จะสำเร็จ")
	}
	if !strings.Contains(err.Error(), "Neon Auth") {
		t.Fatalf("error ต้องมาจากการโหลดกุญแจ Neon Auth ได้: %q", err.Error())
	}
}
