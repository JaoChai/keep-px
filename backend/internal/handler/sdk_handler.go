package handler

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/jaochai/pixlinks/backend/internal/static"
)

type SDKHandler struct {
	etag string
}

func NewSDKHandler() *SDKHandler {
	hash := sha256.Sum256(static.SDKBundle)
	etag := fmt.Sprintf(`"%x"`, hash[:8])
	return &SDKHandler{etag: etag}
}

func (h *SDKHandler) ServeSDK(w http.ResponseWriter, r *http.Request) {
	if match := r.Header.Get("If-None-Match"); match == h.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
	w.Header().Set("ETag", h.etag)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(static.SDKBundle)
}
