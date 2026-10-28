package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

const (
	CodeInternalError     = "INTERNAL_ERROR"
	CodeInvalidID         = "INVALID_ID"
	CodeInvalidJSON       = "INVALID_JSON"
	CodeDuplicateSKU      = "DUPLICATE_SKU"
	CodeValidationFailed  = "VALIDATION_FAILED"
	CodeNotFound          = "NOT_FOUND"
	CodeProductHasHistory = "PRODUCT_HAS_HISTORY"
)

type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	SKU       string            `json:"sku,omitempty"`
	Available *int              `json:"available,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	ErrorWith(w, status, ErrorBody{Code: code, Message: message})
}

func InternalError(w http.ResponseWriter, err error, context string) {
	log.Printf("%s: %v", context, err)
	Error(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

func ErrorWith(w http.ResponseWriter, status int, body ErrorBody) {
	JSON(w, status, map[string]ErrorBody{"error": body})
}
