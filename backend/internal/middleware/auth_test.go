package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-jwt"

// makeToken creates a signed JWT token using the given claims and secret.
func makeToken(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func TestJWTAuth_ValidToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "customer-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := makeToken(t, claims, testSecret)

	var gotCustomerID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCustomerID = GetCustomerID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "customer-123", gotCustomerID)
}

func TestJWTAuth_MissingAuthorizationHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "missing authorization header", body["error"])
}

func TestJWTAuth_NoBearerPrefix(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "customer-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := makeToken(t, claims, testSecret)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid authorization format", body["error"])
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "customer-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tokenStr := makeToken(t, claims, testSecret)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid token", body["error"])
}

func TestJWTAuth_WrongSigningSecret(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "customer-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := makeToken(t, claims, "wrong-secret")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid token", body["error"])
}

func TestJWTAuth_NonHMACSigningMethod(t *testing.T) {
	// Create a token with the "none" algorithm. jwt.UnsafeAllowNoneSignatureType
	// lets us sign it without a key. The middleware should reject it because
	// the signing method is not *jwt.SigningMethodHMAC.
	claims := jwt.MapClaims{
		"sub": "customer-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err = json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid token", body["error"])
}

func TestJWTAuth_MissingSubClaim(t *testing.T) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		// deliberately no "sub" claim
	}
	tokenStr := makeToken(t, claims, testSecret)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := JWTAuth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid subject", body["error"])
}

func TestJWTAuth_IsAdminPropagated(t *testing.T) {
	tests := []struct {
		name     string
		isAdmin  interface{}
		expected bool
	}{
		{"admin true", true, true},
		{"admin false", false, false},
		{"admin not set", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"sub": "customer-123",
				"exp": time.Now().Add(time.Hour).Unix(),
			}
			if tt.isAdmin != nil {
				claims["is_admin"] = tt.isAdmin
			}
			tokenStr := makeToken(t, claims, testSecret)

			var gotIsAdmin bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotIsAdmin = GetIsAdmin(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			handler := JWTAuth(testSecret)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tokenStr)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expected, gotIsAdmin)
		})
	}
}

func TestGetCustomerID_NotInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := GetCustomerID(req.Context())
	assert.Equal(t, "", id)
}
