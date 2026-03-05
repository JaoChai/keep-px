package middleware

import (
	"context"
	"net/http"
)

const IsAdminKey contextKey = "is_admin"

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := GetIsAdmin(r.Context())
		if !isAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetIsAdmin(ctx context.Context) bool {
	isAdmin, _ := ctx.Value(IsAdminKey).(bool)
	return isAdmin
}
