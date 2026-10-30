package httpapi

import "github.com/gin-gonic/gin"

// Kode error dibaca frontend untuk memilih tampilan. Teksnya stabil, jangan diganti tanpa mengubah frontend.
const (
	CodeValidation     = "validation_failed"
	CodeNotFound       = "not_found"
	CodeClosed         = "registration_closed"
	CodeDuplicateEmail = "duplicate_email"
	CodeRateLimited    = "rate_limited"
	CodeInternal       = "internal_error"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

func writeError(c *gin.Context, status int, code, message string, fields []FieldError) {
	c.AbortWithStatusJSON(status, gin.H{"error": errorBody{Code: code, Message: message, Fields: fields}})
}
