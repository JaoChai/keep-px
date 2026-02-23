package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/service"
	"github.com/jaochai/pixlinks/backend/internal/templates"
)

type SalePageHandler struct {
	salePageService *service.SalePageService
	validate        *validator.Validate
	templates       map[string]*template.Template
	logger          *slog.Logger
}

func NewSalePageHandler(salePageService *service.SalePageService, logger *slog.Logger) *SalePageHandler {
	funcMap := template.FuncMap{
		"deref": func(s *string) string {
			if s != nil {
				return *s
			}
			return ""
		},
	}

	tmplMap := make(map[string]*template.Template)
	tmplMap["simple"] = template.Must(
		template.New("simple.html").Funcs(funcMap).ParseFS(templates.SalePageTemplates, "sale_pages/simple.html"),
	)

	return &SalePageHandler{
		salePageService: salePageService,
		validate:        validator.New(),
		templates:       tmplMap,
		logger:          logger,
	}
}

type salePageTemplateData struct {
	Page    *domain.SalePage
	Content *domain.SimpleContent
	APIKey  string
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
		if errors.Is(err, service.ErrInvalidSlug) {
			ErrorJSON(w, http.StatusBadRequest, "slug must contain only lowercase letters, numbers, and hyphens")
			return
		}
		if errors.Is(err, service.ErrSlugTaken) {
			ErrorJSON(w, http.StatusConflict, "slug is already taken")
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

	h.renderTemplate(w, data.Page, data.APIKey)
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

	h.renderTemplate(w, page, "")
}

func (h *SalePageHandler) renderTemplate(w http.ResponseWriter, page *domain.SalePage, apiKey string) {
	var content domain.SimpleContent
	if err := json.Unmarshal(page.Content, &content); err != nil {
		http.Error(w, "invalid page content", http.StatusInternalServerError)
		return
	}

	tmpl, ok := h.templates[page.TemplateName]
	if !ok {
		h.logger.Warn("unknown template, falling back to simple", "template_name", page.TemplateName, "page_id", page.ID)
		tmpl = h.templates["simple"]
	}

	td := salePageTemplateData{
		Page:    page,
		Content: &content,
		APIKey:  apiKey,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		h.logger.Error("template render failed", "error", err, "page_id", page.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func (h *SalePageHandler) renderNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>Not Found</title></head><body style="display:flex;align-items:center;justify-content:center;min-height:100vh;font-family:sans-serif;color:#666"><h1>Page Not Found</h1></body></html>`))
}
