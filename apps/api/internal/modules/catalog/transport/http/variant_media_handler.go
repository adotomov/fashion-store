package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type variantResponse struct {
	ID                string                   `json:"id"`
	ProductID         string                   `json:"product_id"`
	PriceOverride     *moneyResponse           `json:"price_override,omitempty"`
	AttributeValueIDs []string                 `json:"attribute_value_ids"`
	Attributes        []attributeValueResponse `json:"attributes"`
	InventoryItemID   *string                  `json:"inventory_item_id,omitempty"`
	QuantityAvailable *int                     `json:"quantity_available,omitempty"`
	CreatedAt         string                   `json:"created_at"`
	UpdatedAt         string                   `json:"updated_at"`
}

func toVariantResponse(v domain.ProductVariant) variantResponse {
	resp := variantResponse{
		ID:                v.ID.String(),
		ProductID:         v.ProductID.String(),
		AttributeValueIDs: []string{},
		QuantityAvailable: v.QuantityAvailable,
		CreatedAt:         v.CreatedAt.Format(timeFormat),
		UpdatedAt:         v.UpdatedAt.Format(timeFormat),
	}
	if v.PriceOverride != nil {
		m := toMoneyResponse(*v.PriceOverride)
		resp.PriceOverride = &m
	}
	if v.InventoryItemID != nil {
		id := v.InventoryItemID.String()
		resp.InventoryItemID = &id
	}
	for _, a := range v.Attributes {
		resp.AttributeValueIDs = append(resp.AttributeValueIDs, a.ID.String())
		resp.Attributes = append(resp.Attributes, toAttributeValueResponse(a))
	}
	return resp
}

type variantRequest struct {
	AttributeValueIDs  []string       `json:"attribute_value_ids"`
	PriceOverride      *moneyResponse `json:"price_override,omitempty"`
	ClearPriceOverride bool           `json:"clear_price_override,omitempty"`
}

func parseUUIDList(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (h *ProductHandler) createVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	var req variantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	attributeValueIDs, err := parseUUIDList(req.AttributeValueIDs)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "attribute_value_ids is invalid")
		return
	}

	input := application.CreateVariantInput{AttributeValueIDs: attributeValueIDs}
	if req.PriceOverride != nil {
		input.PriceOverride = &money.Money{AmountMinor: req.PriceOverride.AmountMinor, Currency: req.PriceOverride.Currency}
	}

	variant, err := h.service.AddVariant(r.Context(), productID, input)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toVariantResponse(*variant))
}

func (h *ProductHandler) updateVariant(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(chi.URLParam(r, "variantId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "variant id is invalid")
		return
	}

	var req variantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	attributeValueIDs, err := parseUUIDList(req.AttributeValueIDs)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "attribute_value_ids is invalid")
		return
	}

	input := application.UpdateVariantInput{
		AttributeValueIDs:  attributeValueIDs,
		ClearPriceOverride: req.ClearPriceOverride,
	}
	if req.PriceOverride != nil {
		input.PriceOverride = &money.Money{AmountMinor: req.PriceOverride.AmountMinor, Currency: req.PriceOverride.Currency}
	}

	variant, err := h.service.UpdateVariant(r.Context(), variantID, input)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toVariantResponse(*variant))
}

func (h *ProductHandler) deleteVariant(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(chi.URLParam(r, "variantId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "variant id is invalid")
		return
	}

	if err := h.service.DeleteVariant(r.Context(), variantID); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type mediaResponse struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"object_key"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Position    int    `json:"position"`
	AltText     string `json:"alt_text"`
	CreatedAt   string `json:"created_at"`
}

func toMediaResponse(m domain.ProductMedia) mediaResponse {
	return mediaResponse{
		ID:          m.ID.String(),
		ProductID:   m.ProductID.String(),
		Bucket:      m.Bucket,
		ObjectKey:   m.ObjectKey,
		ContentType: m.ContentType,
		SizeBytes:   m.SizeBytes,
		Position:    m.Position,
		AltText:     m.AltText,
		CreatedAt:   m.CreatedAt.Format(timeFormat),
	}
}

func (h *ProductHandler) createMedia(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	const maxUploadBytes = 10 << 20 // 10 MiB
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}
	defer file.Close()

	position, _ := strconv.Atoi(r.FormValue("position"))
	altText := r.FormValue("alt_text")
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	media, err := h.service.UploadMedia(r.Context(), productID, header.Filename, contentType, file, position, altText)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toMediaResponse(*media))
}

// serveMedia proxies the stored object's bytes back to the client. The
// admin UI's <img src> points here rather than at the storage backend
// directly, since FakeGCS's self-signed local cert would otherwise make
// the browser silently refuse to load the image.
func (h *ProductHandler) serveMedia(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "media id is invalid")
		return
	}

	reader, contentType, err := h.service.OpenMedia(r.Context(), mediaID)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	defer reader.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, reader)
}

func (h *ProductHandler) updateMedia(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "media id is invalid")
		return
	}

	var req struct {
		Position *int    `json:"position,omitempty"`
		AltText  *string `json:"alt_text,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	media, err := h.service.UpdateMedia(r.Context(), mediaID, application.UpdateMediaInput{
		Position: req.Position,
		AltText:  req.AltText,
	})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toMediaResponse(*media))
}

func (h *ProductHandler) deleteMedia(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "media id is invalid")
		return
	}

	if err := h.service.DeleteMedia(r.Context(), mediaID); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
