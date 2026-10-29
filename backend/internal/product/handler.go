package product

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"stockoverflow/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func queryInt(r *http.Request, key string, fallback int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return v
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 10)

	result, err := h.service.List(r.Context(), search, page, limit)

	if err != nil {
		httpx.InternalError(w, err, "list product")
		return
	}

	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	p, err := h.service.GetByID(r.Context(), id)

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "show product")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateProductInput
	err := json.NewDecoder(r.Body).Decode(&in)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidJSON, "invalid json body")
		return
	}

	p, err := h.service.Create(r.Context(), in)

	if ve, ok := errors.AsType[*ValidationError](err); ok {
		httpx.ErrorWith(w, http.StatusBadRequest, httpx.ErrorBody{
			Code:    httpx.CodeValidationFailed,
			Message: "validation failed",
			Fields:  ve.Fields,
		})
		return
	}

	if errors.Is(err, ErrDuplicateSKU) {
		httpx.Error(w, http.StatusConflict, httpx.CodeDuplicateSKU, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "create product")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"data": p})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	var in UpdateProductInput

	err = json.NewDecoder(r.Body).Decode(&in)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidJSON, "invalid json body")
		return
	}

	p, err := h.service.Update(r.Context(), id, in)

	if ve, ok := errors.AsType[*ValidationError](err); ok {
		httpx.ErrorWith(w, http.StatusBadRequest, httpx.ErrorBody{
			Code:    httpx.CodeValidationFailed,
			Message: "validation failed",
			Fields:  ve.Fields,
		})
		return
	}

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "update product")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	err = h.service.Delete(r.Context(), id)

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if errors.Is(err, ErrProductHasHistory) {
		httpx.Error(w, http.StatusConflict, httpx.CodeProductHasHistory, "product has history")
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	var in AdjustStockInput
	err = json.NewDecoder(r.Body).Decode(&in)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidJSON, "invalid json body")
		return
	}

	p, err := h.service.AdjustStock(r.Context(), id, in)

	if ve, ok := errors.AsType[*ValidationError](err); ok {
		httpx.ErrorWith(w, http.StatusBadRequest, httpx.ErrorBody{
			Code:    httpx.CodeValidationFailed,
			Message: "validation failed",
			Fields:  ve.Fields,
		})
		return
	}

	if ve, ok := errors.AsType[*InsufficientStockError](err); ok {
		httpx.ErrorWith(w, http.StatusConflict, httpx.ErrorBody{
			Code:      httpx.CodeInsufficientStock,
			Message:   ve.Error(),
			Available: &ve.Available,
		})
		return
	}

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "adjust stock product")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"data": p})
}

func (h *Handler) Movements(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	p, err := h.service.ListMovements(r.Context(), id)

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "list movements")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": p})
}
