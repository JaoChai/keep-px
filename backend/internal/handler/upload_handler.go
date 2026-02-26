package handler

import (
	"net/http"
	"strings"

	"github.com/jaochai/pixlinks/backend/internal/service"
)

const maxUploadSize = 5 << 20 // 5MB

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type UploadHandler struct {
	storageService *service.StorageService
}

func NewUploadHandler(storageService *service.StorageService) *UploadHandler {
	return &UploadHandler{storageService: storageService}
}

func (h *UploadHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if !h.storageService.IsConfigured() {
		ErrorJSON(w, http.StatusServiceUnavailable, "image upload is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	file, header, err := r.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			ErrorJSON(w, http.StatusRequestEntityTooLarge, "file too large (max 5MB)")
			return
		}
		ErrorJSON(w, http.StatusBadRequest, "invalid file upload")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		ErrorJSON(w, http.StatusBadRequest, "only JPEG, PNG, WebP, and GIF images are allowed")
		return
	}

	url, err := h.storageService.Upload(r.Context(), file, header.Filename, contentType)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to upload image")
		return
	}

	JSON(w, http.StatusOK, APIResponse{Data: map[string]string{"url": url}})
}
