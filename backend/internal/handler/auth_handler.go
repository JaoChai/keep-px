package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	logger      *slog.Logger
	jwks        jwt.Keyfunc
	issuer      string
}

func NewAuthHandler(authService *service.AuthService, logger *slog.Logger, jwks jwt.Keyfunc, issuer string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
		jwks:        jwks,
		issuer:      issuer,
	}
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	customer, err := h.authService.GetCustomerByID(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusUnauthorized, "invalid token")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: customer})
}

// Session สร้างหรือผูก customer กับ user ของ Neon Auth
// ตรวจ JWT เอง (ไม่ใช้ middleware JWTAuth) เพราะ middleware ตัวนั้นจะปฏิเสธ
// ด้วย customer_not_provisioned ก่อนที่เราจะได้สร้าง customer — ไก่กับไข่
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
		if errors.Is(err, service.ErrEmailAlreadyLinked) {
			ErrorJSON(w, http.StatusForbidden, "email already linked to a different account")
			return
		}
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "provision failed", err)
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: customer})
}

func (h *AuthHandler) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	customer, err := h.authService.RegenerateAPIKey(r.Context(), customerID)
	if err != nil {
		ErrorJSONWithLog(w, r, h.logger, http.StatusInternalServerError, "failed to regenerate api key", err)
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: customer})
}
