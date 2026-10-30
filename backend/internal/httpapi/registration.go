package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
)

// Empat peserta dengan isian terpanjang masih jauh di bawah 16 KB. Body yang lebih besar pasti bukan dari formulir.
const maxRegistrationBody = 16 << 10

type registrationResponse struct {
	Code             string   `json:"code"`
	Type             string   `json:"type"`
	Status           string   `json:"status"`
	WaitlistPosition *int64   `json:"waitlist_position"`
	Attendees        []string `json:"attendees"`
}

func registerHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRegistrationBody)

		var req registration.Request
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Data formulir tidak bisa dibaca. Muat ulang halaman, lalu coba lagi.", nil)
			return
		}

		in, fieldErrs := registration.Validate(req)
		if len(fieldErrs) > 0 {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Ada isian yang perlu dicek.", toFieldErrors(fieldErrs))
			return
		}

		res, err := deps.Registrations.Register(c.Request.Context(), in)
		var vErr *registration.ValidationError
		var dErr *registration.DuplicateEmailError
		switch {
		case errors.Is(err, registration.ErrClosed):
			writeError(c, http.StatusConflict, CodeClosed, "Pendaftaran sudah ditutup.", nil)
			return
		case errors.As(err, &vErr):
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Ada isian yang perlu dicek.", toFieldErrors(vErr.Fields))
			return
		case errors.As(err, &dErr):
			writeError(c, http.StatusConflict, CodeDuplicateEmail,
				"Ada email yang sudah terdaftar.", toFieldErrors(dErr.Fields))
			return
		case err != nil:
			internalError(c, deps.Log, "simpan pendaftaran", err)
			return
		}

		// 200 untuk kiriman ulang dengan idempotency key yang sama, supaya frontend bisa membedakan
		// tapi tetap menampilkan kartu akses yang sama.
		status := http.StatusCreated
		if res.Replayed {
			status = http.StatusOK
		}

		var pos *int64
		if res.Status == sqlc.RegistrationStatusWaitlisted {
			p := res.WaitlistPosition
			pos = &p
		}

		c.JSON(status, registrationResponse{
			Code:             res.Code,
			Type:             string(res.Type),
			Status:           string(res.Status),
			WaitlistPosition: pos,
			Attendees:        res.Names,
		})
	}
}

func toFieldErrors(in []registration.FieldError) []FieldError {
	out := make([]FieldError, len(in))
	for i, f := range in {
		out[i] = FieldError{Field: f.Field, Message: f.Message}
	}
	return out
}
