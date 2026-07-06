package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

const maxDocumentUploadBytes = 25 << 20 // 25 MiB — terms/privacy are PDFs/DOCX, larger than a thumbnail

type StoreDocumentHandler struct {
	service *application.StoreDocumentService
}

func NewStoreDocumentHandler(service *application.StoreDocumentService) *StoreDocumentHandler {
	return &StoreDocumentHandler{service: service}
}

func (h *StoreDocumentHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/store-settings/documents/{type}", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.upload)
			r.Get("/file", h.serve)
			r.Delete("/", h.delete)
			r.Get("/content", h.getContent)
			r.Put("/content", h.saveContent)
		})
	})
}

func (h *StoreDocumentHandler) RegisterStorefrontRoutes(r chi.Router) {
	r.Get("/storefront/store-settings/documents/{type}/file", h.serve)
	r.Get("/storefront/store-settings/documents/{type}/content", h.getContent)
}

func parseDocumentType(r *http.Request) (domain.DocumentType, error) {
	switch chi.URLParam(r, "type") {
	case string(domain.DocumentTypeTerms):
		return domain.DocumentTypeTerms, nil
	case string(domain.DocumentTypePrivacy):
		return domain.DocumentTypePrivacy, nil
	default:
		return "", domain.ErrInvalidDocumentType
	}
}

func localeFromQuery(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		return "en"
	}
	return locale
}

type storeDocumentResponse struct {
	Locale   string `json:"locale"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

func (h *StoreDocumentHandler) list(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	docs, err := h.service.List(r.Context(), docType)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	resp := make([]storeDocumentResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, storeDocumentResponse{
			Locale:   d.Locale,
			Filename: d.Filename,
			URL:      "/api/v1/admin/store-settings/documents/" + string(d.Type) + "/file?locale=" + d.Locale,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StoreDocumentHandler) upload(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}

	if err := r.ParseMultipartForm(maxDocumentUploadBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	locale := r.FormValue("locale")
	if locale == "" {
		locale = "en"
	}

	doc, err := h.service.Upload(r.Context(), docType, locale, header.Filename, contentType, file)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, storeDocumentResponse{
		Locale:   doc.Locale,
		Filename: doc.Filename,
		URL:      "/api/v1/admin/store-settings/documents/" + string(doc.Type) + "/file?locale=" + doc.Locale,
	})
}

func (h *StoreDocumentHandler) serve(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}

	reader, contentType, err := h.service.Open(r.Context(), docType, localeFromQuery(r))
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	defer reader.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, reader)
}

func (h *StoreDocumentHandler) delete(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), docType, localeFromQuery(r)); err != nil {
		writeAdminModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type legalContentResponse struct {
	Locale    string `json:"locale"`
	ContentMD string `json:"content_md"`
}

func (h *StoreDocumentHandler) getContent(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	locale := localeFromQuery(r)
	content, err := h.service.GetContent(r.Context(), docType, locale)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, legalContentResponse{Locale: locale, ContentMD: content})
}

func (h *StoreDocumentHandler) saveContent(w http.ResponseWriter, r *http.Request) {
	docType, err := parseDocumentType(r)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	var body struct {
		Locale    string `json:"locale"`
		ContentMD string `json:"content_md"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not parse request body")
		return
	}
	if body.Locale == "" {
		body.Locale = "en"
	}
	doc, err := h.service.SaveContent(r.Context(), docType, body.Locale, body.ContentMD)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	content := ""
	if doc.ContentMD != nil {
		content = *doc.ContentMD
	}
	httpx.WriteJSON(w, http.StatusOK, legalContentResponse{Locale: doc.Locale, ContentMD: content})
}
