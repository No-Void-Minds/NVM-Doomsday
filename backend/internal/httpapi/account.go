package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
)

// Satu nama akun jauh di bawah 1 KB.
const maxAccountCheckBody = 1 << 10

type accountCheckRequest struct {
	AccountName string `json:"account_name"`
}

type accountCheckResponse struct {
	Type string `json:"type"`
}

// POST, bukan GET, supaya nama akun tidak tercatat di URL. JSON wajib dengan alasan yang sama
// seperti pendaftaran: kiriman dari situs lain ditahan pemeriksaan CORS browser.
func accountCheckHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			writeError(c, http.StatusUnsupportedMediaType, CodeValidation,
				"Format kiriman tidak didukung.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAccountCheckBody)

		var req accountCheckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Data formulir tidak bisa dibaca. Muat ulang halaman, lalu coba lagi.", nil)
			return
		}

		account, fieldErrs := registration.ValidateAccount(req.AccountName)
		if len(fieldErrs) > 0 {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Ada isian yang perlu dicek.", toFieldErrors(fieldErrs))
			return
		}

		t, err := deps.Registrations.CheckAccount(c.Request.Context(), account)
		switch {
		case errors.Is(err, registration.ErrAccountTaken):
			writeError(c, http.StatusConflict, CodeAccountTaken, registration.MsgAccountTaken,
				[]FieldError{{Field: "account_name", Message: registration.MsgAccountTaken}})
			return
		case err != nil:
			internalError(c, deps.Log, "cek nama akun", err)
			return
		}

		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, accountCheckResponse{Type: string(t)})
	}
}
