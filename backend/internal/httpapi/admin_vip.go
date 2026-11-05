package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/vip"
)

// 500 nama x 50 karakter, ditambah tanda kutip dan koma, masih di bawah batas ini.
const maxVipBody = 64 << 10

type vipAccountResponse struct {
	ID               string    `json:"id"`
	AccountName      string    `json:"account_name"`
	IsActive         bool      `json:"is_active"`
	RegistrationCode *string   `json:"registration_code"`
	CreatedAt        time.Time `json:"created_at"`
}

type vipAddRequest struct {
	Names []string `json:"names"`
}

type vipRejectedResponse struct {
	AccountName string `json:"account_name"`
	Message     string `json:"message"`
}

type vipAddResponse struct {
	Added       []string              `json:"added"`
	Reactivated []string              `json:"reactivated"`
	Existing    []string              `json:"existing"`
	Rejected    []vipRejectedResponse `json:"rejected"`
}

type vipUpdateRequest struct {
	IsActive *bool `json:"is_active"`
}

func vipListHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		accounts, err := deps.Vip.List(c.Request.Context())
		if err != nil {
			internalError(c, deps.Log, "ambil daftar VIP", err)
			return
		}
		out := make([]vipAccountResponse, len(accounts))
		for i, a := range accounts {
			out[i] = vipAccountResponse{
				ID:          a.ID.String(),
				AccountName: a.Name,
				IsActive:    a.Active,
				CreatedAt:   a.CreatedAt.UTC(),
			}
			if a.RegistrationCode != "" {
				code := a.RegistrationCode
				out[i].RegistrationCode = &code
			}
		}
		c.JSON(http.StatusOK, gin.H{"accounts": out})
	}
}

func vipAddHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			writeError(c, http.StatusUnsupportedMediaType, CodeValidation, "Format kiriman tidak didukung.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxVipBody)

		var req vipAddRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation,
				"Daftar nama tidak bisa dibaca. Muat ulang halaman, lalu coba lagi.", nil)
			return
		}
		if !hasNonBlank(req.Names) {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation, "Ada isian yang perlu dicek.",
				[]FieldError{{Field: "names", Message: "Isi minimal satu nama akun."}})
			return
		}

		rep, err := deps.Vip.Add(c.Request.Context(), req.Names)
		if errors.Is(err, vip.ErrTooMany) {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation, "Ada isian yang perlu dicek.",
				[]FieldError{{Field: "names", Message: "Maksimal 500 nama sekali tambah. Bagi jadi beberapa kali."}})
			return
		}
		if err != nil {
			internalError(c, deps.Log, "tambah nama VIP", err)
			return
		}

		rejected := make([]vipRejectedResponse, len(rep.Rejected))
		for i, r := range rep.Rejected {
			rejected[i] = vipRejectedResponse{AccountName: r.Name, Message: r.Message}
		}
		c.JSON(http.StatusOK, vipAddResponse{
			Added:       rep.Added,
			Reactivated: rep.Reactivated,
			Existing:    rep.Existing,
			Rejected:    rejected,
		})
	}
}

func vipUpdateHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			writeError(c, http.StatusUnsupportedMediaType, CodeValidation, "Format kiriman tidak didukung.", nil)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			writeError(c, http.StatusNotFound, CodeNotFound, "Nama akun tidak ditemukan.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<10)

		var req vipUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation, "Ada isian yang perlu dicek.",
				[]FieldError{{Field: "is_active", Message: "Pilih aktif atau nonaktif."}})
			return
		}

		_, err = deps.Vip.SetActive(c.Request.Context(), id, *req.IsActive)
		switch {
		case errors.Is(err, vip.ErrNotFound):
			writeError(c, http.StatusNotFound, CodeNotFound, "Nama akun tidak ditemukan.", nil)
		case err != nil:
			internalError(c, deps.Log, "ubah status VIP", err)
		default:
			c.Status(http.StatusNoContent)
		}
	}
}

func hasNonBlank(names []string) bool {
	for _, n := range names {
		if strings.TrimSpace(n) != "" {
			return true
		}
	}
	return false
}
