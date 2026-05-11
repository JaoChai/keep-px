package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReady_ReturnsOK(t *testing.T) {
	h := &HealthHandler{pool: nil}
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"status":"ready"}` {
		t.Fatalf("unexpected body: %q", got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("want Content-Type application/json, got %q", ct)
	}
}
