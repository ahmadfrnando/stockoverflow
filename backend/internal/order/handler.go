package order

import (
	// "errors"
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateOrderInput
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

	if ve, ok := errors.AsType[*InsufficientStockError](err); ok {
		httpx.ErrorWith(w, http.StatusConflict, httpx.ErrorBody{
			Code:      httpx.CodeInsufficientStock,
			Message:   ve.Error(),
			SKU:       ve.SKU,
			Available: &ve.Available,
		})
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "create product")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"data": p})
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
		httpx.InternalError(w, err, "show order")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidID, "invalid id")
		return
	}

	p, err := h.service.Cancel(r.Context(), id)

	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}

	if errors.Is(err, ErrAlreadyCancelled) {
		httpx.Error(w, http.StatusConflict, httpx.CodeAlreadyCancelled, err.Error())
		return
	}

	if err != nil {
		httpx.InternalError(w, err, "cancel order")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": p})
}
