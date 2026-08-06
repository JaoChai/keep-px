package middleware

import (
	"net"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// ClientIP returns the client IP resolved by chi's ClientIPFromHeader
// middleware (registered in router.go), falling back to the TCP peer address
// when no trusted header was present. Client-supplied headers are never
// trusted — see docs/superpowers/specs/2026-08-06-dependency-update-followups.md
func ClientIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
