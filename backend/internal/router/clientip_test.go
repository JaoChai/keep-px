package router

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/config"
)

// newTestRouter builds the real router with a nil DB pool. The rate limiter is
// registered on the /api/v1 route and runs before route matching, so a request
// to an unmatched path like /api/v1/zzz still traverses the full middleware
// chain and returns 404 (or 429 when the limiter trips) without touching the
// database — exactly what these guards need to observe.
func newTestRouter(t *testing.T) (http.Handler, func()) {
	t.Helper()
	// JWKS stub: New() ดึงกุญแจตอน startup และ (หลังแก้ NoErrorReturnFirstHTTPReq=false)
	// จะ fail จริงถ้าดึงไม่ได้ — เทสต์นี้ไม่ได้ลอง JWT จึงให้ server ตอบ JWKS ว่างๆ พอให้ New ผ่าน
	jwks := startDummyJWKS(http.StatusOK, `{"keys":[]}`)
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		Env:                  "test",
		NeonAuthURL:          jwks.URL,
		RateLimitRPS:         1,
		RateLimitAPIKeyRPS:   100,
		RateLimitAPIKeyBurst: 100,
		DBQueryTimeout:       time.Second,
		SalePageCacheTTL:     time.Minute,
	}
	h, cleanup, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, ctx)
	if err != nil {
		jwks.Close()
		cancel()
		t.Fatalf("failed to build router: %v", err)
	}
	return h, func() {
		cleanup()
		cancel()
		jwks.Close()
	}
}

// do fires one GET /api/v1/zzz through h from the given RemoteAddr with the
// given headers and returns the response status code.
func do(t *testing.T, h http.Handler, remoteAddr string, headers map[string]string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/zzz", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

// TestRouter_ForgedHeadersCannotBuyFreshBuckets is the router-level guard that
// replaces the brittle CI grep: it proves router.New actually registers
// chimiddleware.ClientIPFromHeader("CF-Connecting-IP"). Six requests share one real
// identity — the same CF-Connecting-IP, the one header Cloudflare's edge overwrites
// and the only one ClientIPFromHeader reads — but vary everything a client CAN
// control: a unique forged X-Real-IP, True-Client-IP and X-Forwarded-For per
// request, AND a distinct RemoteAddr per request.
//
// Correct wiring collapses all six into the single 203.0.113.1 bucket
// (RateLimitWithContext uses burst = rps*2 = 2), so at least one request must
// be 429.
//
// The distinct per-request RemoteAddr is what lets this catch a missing
// middleware: if someone comments out ClientIPFromHeader (or swaps RealIP back
// in), the limiter keys on the client-controlled RemoteAddr / forged header
// instead, every request lands in a fresh bucket, and no 429 ever appears.
// With a constant RemoteAddr that regression would be invisible, because the
// RemoteAddr fallback alone would still collide and trip 429.
//
// Two couplings to be aware of before changing anything here:
//
//   - The request count must stay above the limiter's burst. RateLimitWithContext
//     derives burst as rps*2, so rps=1 gives burst 2 and six requests leave
//     headroom. Widen that formula in ratelimit.go without raising the count here
//     and this test goes red on correct code.
//   - This test alone would pass vacuously if RateLimitRPS were ever 0, because
//     rate.NewLimiter(0, 0) rejects everything and the first request already
//     returns 429. TestRouter_DistinctRealIPsKeepSeparateBuckets is what rules
//     that out — it fails when the limiter blocks traffic it should allow. Keep
//     the pair together; neither is sufficient on its own.
func TestRouter_ForgedHeadersCannotBuyFreshBuckets(t *testing.T) {
	h, cleanup := newTestRouter(t)
	defer cleanup()

	for i := 1; i <= 6; i++ {
		code := do(t, h,
			fmt.Sprintf("10.0.0.%d:1234", i),
			map[string]string{
				"CF-Connecting-IP": "203.0.113.1",
				"X-Real-IP":        fmt.Sprintf("198.51.100.%d", i),
				"True-Client-IP":   fmt.Sprintf("198.51.100.%d", i),
				"X-Forwarded-For":  fmt.Sprintf("192.0.2.%d", i),
			},
		)
		if code == http.StatusTooManyRequests {
			return // bucket collision observed — the guard holds
		}
	}

	t.Fatal("router does not resolve the client IP from CF-Connecting-IP — forged " +
		"X-Real-IP / True-Client-IP / X-Forwarded-For bought a fresh rate-limit " +
		"bucket on every request. chimiddleware.ClientIPFromHeader(\"CF-Connecting-IP\") " +
		"is not registered in router.New. See " +
		"docs/superpowers/specs/2026-08-12-cloudflare-migration-design.md")
}

// TestRouter_DistinctRealIPsKeepSeparateBuckets is the complement: five
// genuine visitors behind the SAME RemoteAddr (as all traffic is behind Cloudflare
// in production) but each carrying a distinct, edge-resolved CF-Connecting-IP. Each
// must land in its own bucket, so none is rate-limited — distinct real
// customers must not be collapsed into one bucket.
func TestRouter_DistinctRealIPsKeepSeparateBuckets(t *testing.T) {
	h, cleanup := newTestRouter(t)
	defer cleanup()

	for i := 1; i <= 5; i++ {
		code := do(t, h,
			"10.0.0.1:1234",
			map[string]string{
				"CF-Connecting-IP": fmt.Sprintf("203.0.113.%d", i),
			},
		)
		if code == http.StatusTooManyRequests {
			t.Fatalf("request %d with distinct CF-Connecting-IP 203.0.113.%d was "+
				"rate-limited (429) — distinct real client IPs must keep "+
				"separate buckets", i, i)
		}
	}
}
