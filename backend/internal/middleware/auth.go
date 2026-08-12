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
