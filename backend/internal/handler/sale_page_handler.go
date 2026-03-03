package handler

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
	"github.com/jaochai/pixlinks/backend/internal/templates"
)

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

type SalePageHandler struct {
	salePageService *service.SalePageService
	validate        *validator.Validate
	templates       map[string]*template.Template
	logger          *slog.Logger
	baseURL         string
}

func NewSalePageHandler(salePageService *service.SalePageService, baseURL string, logger *slog.Logger) *SalePageHandler {
	funcMap := template.FuncMap{
		"deref": func(s *string) string {
			if s != nil {
				return *s
			}
			return ""
		},
		"firstImageURL": func(blocks []domain.Block) string {
			for _, b := range blocks {
				if b.Type == "image" && b.ImageURL != "" {
					return b.ImageURL
				}
			}
			return ""
		},
		"hasPixels": func(pixels []service.PixelPublishInfo) bool {
			return len(pixels) > 0
		},
	}

	tmplMap := make(map[string]*template.Template)
	tmplMap["simple"] = template.Must(
		template.New("simple.html").Funcs(funcMap).ParseFS(templates.SalePageTemplates, "sale_pages/simple.html", "sale_pages/tracking.html"),
	)
	tmplMap["blocks"] = template.Must(
		template.New("blocks.html").Funcs(funcMap).ParseFS(templates.SalePageTemplates, "sale_pages/blocks.html", "sale_pages/tracking.html"),
	)

	return &SalePageHandler{
		salePageService: salePageService,
		validate:        validator.New(),
		templates:       tmplMap,
		logger:          logger,
		baseURL:         strings.TrimRight(baseURL, "/"),
	}
}

type salePageTemplateData struct {
	Page          *domain.SalePage
	Content       *domain.SimpleContent
	BlocksContent *domain.BlocksContent
	Style         domain.PageStyle
	Tracking      domain.TrackingConfig
	APIKey        string
	Pixels        []service.PixelPublishInfo
	BaseURL       string
}

func (h *SalePageHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pages, err := h.salePageService.List(r.Context(), customerID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to list sale pages")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: pages})
}

func (h *SalePageHandler) Create(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())

	var input service.CreateSalePageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := h.salePageService.Create(r.Context(), customerID, input)
	if err != nil {
		if errors.Is(err, service.ErrQuotaSalePagesExceeded) {
			ErrorJSON(w, http.StatusPaymentRequired, "sale page limit exceeded")
			return
		}
		if errors.Is(err, service.ErrInvalidSlug) {
			ErrorJSON(w, http.StatusBadRequest, "slug must contain only lowercase letters, numbers, and hyphens")
			return
		}
		if errors.Is(err, service.ErrInvalidContent) {
			ErrorJSON(w, http.StatusBadRequest, "invalid page content structure")
			return
		}
		if errors.Is(err, service.ErrSlugTaken) {
			ErrorJSON(w, http.StatusConflict, "slug is already taken")
			return
		}
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusBadRequest, "one or more pixels not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "one or more pixels not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to create sale page")
		return
	}
	JSON(w, http.StatusCreated, APIResponse{Data: page})
}

func (h *SalePageHandler) Update(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pageID := chi.URLParam(r, "id")

	var input service.UpdateSalePageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	page, err := h.salePageService.Update(r.Context(), customerID, pageID, input)
	if err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		if errors.Is(err, service.ErrSalePageNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "sale page not owned by you")
			return
		}
		if errors.Is(err, service.ErrInvalidContent) {
			ErrorJSON(w, http.StatusBadRequest, "invalid page content structure")
			return
		}
		if errors.Is(err, service.ErrInvalidSlug) {
			ErrorJSON(w, http.StatusBadRequest, "slug must contain only lowercase letters, numbers, and hyphens")
			return
		}
		if errors.Is(err, service.ErrSlugTaken) {
			ErrorJSON(w, http.StatusConflict, "slug is already taken")
			return
		}
		if errors.Is(err, service.ErrPixelNotFound) {
			ErrorJSON(w, http.StatusBadRequest, "one or more pixels not found")
			return
		}
		if errors.Is(err, service.ErrPixelNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "one or more pixels not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to update sale page")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Data: page})
}

func (h *SalePageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pageID := chi.URLParam(r, "id")

	err := h.salePageService.Delete(r.Context(), customerID, pageID)
	if err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			ErrorJSON(w, http.StatusNotFound, "sale page not found")
			return
		}
		if errors.Is(err, service.ErrSalePageNotOwned) {
			ErrorJSON(w, http.StatusForbidden, "sale page not owned by you")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "failed to delete sale page")
		return
	}
	JSON(w, http.StatusOK, APIResponse{Message: "sale page deleted"})
}

func (h *SalePageHandler) Serve(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	data, err := h.salePageService.GetPublishData(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			h.renderNotFound(w)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, r, data.Page, data.APIKey, data.Pixels)
}

func (h *SalePageHandler) Preview(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerID(r.Context())
	pageID := chi.URLParam(r, "id")

	page, err := h.salePageService.GetByID(r.Context(), customerID, pageID)
	if err != nil {
		if errors.Is(err, service.ErrSalePageNotFound) {
			h.renderNotFound(w)
			return
		}
		if errors.Is(err, service.ErrSalePageNotOwned) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, r, page, "", nil)
}

func (h *SalePageHandler) renderTemplate(w http.ResponseWriter, r *http.Request, page *domain.SalePage, apiKey string, pixels []service.PixelPublishInfo) {
	td := salePageTemplateData{
		Page:    page,
		APIKey:  apiKey,
		Pixels:  pixels,
		BaseURL: h.baseURL,
	}

	// Content-aware template detection
	templateName := page.TemplateName
	var peek struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(page.Content, &peek); err != nil {
		h.logger.Error("failed to peek content version", "error", err, "page_id", page.ID)
		http.Error(w, "invalid page content", http.StatusInternalServerError)
		return
	}
	if peek.Version == 2 {
		templateName = "blocks"
	} else if templateName == "blocks" {
		// template_name says blocks but content has no version:2
		templateName = "simple"
		h.logger.Warn("template_name/content version mismatch",
			"page_id", page.ID, "template_name", page.TemplateName)
	}

	if templateName == "blocks" {
		var blocks domain.BlocksContent
		if err := json.Unmarshal(page.Content, &blocks); err != nil {
			h.logger.Error("failed to unmarshal blocks content", "error", err, "page_id", page.ID)
			http.Error(w, "invalid page content", http.StatusInternalServerError)
			return
		}
		td.BlocksContent = &blocks
		td.Style = blocks.Style
		td.Tracking = blocks.Tracking
	} else {
		var content domain.SimpleContent
		if err := json.Unmarshal(page.Content, &content); err != nil {
			h.logger.Error("failed to unmarshal simple content", "error", err, "page_id", page.ID)
			http.Error(w, "invalid page content", http.StatusInternalServerError)
			return
		}
		td.Content = &content
		td.Style = content.Style
		td.Tracking = content.Tracking
	}

	tmpl, ok := h.templates[templateName]
	if !ok {
		h.logger.Warn("unknown template, falling back to simple", "template_name", templateName, "page_id", page.ID)
		tmpl = h.templates["simple"]
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := tmpl.Execute(buf, td); err != nil {
		h.logger.Error("template render failed", "error", err, "page_id", page.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	body := buf.Bytes()
	etag := fmt.Sprintf(`"%x"`, md5.Sum(body))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=30, s-maxage=60")
	w.Header().Set("ETag", etag)

	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if _, err := w.Write(body); err != nil {
		h.logger.Error("failed to write response", "error", err, "page_id", page.ID)
	}
}

func (h *SalePageHandler) renderNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>Not Found</title></head><body style="display:flex;align-items:center;justify-content:center;min-height:100vh;font-family:sans-serif;color:#666"><h1>Page Not Found</h1></body></html>`))
}
