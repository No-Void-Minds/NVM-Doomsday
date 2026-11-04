package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/adminauth"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/config"
)

const adminCookie = "doomsday_admin"

type loginRequest struct {
	Password string `json:"password"`
}

func loginHandler(cfg config.Config, deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			writeError(c, http.StatusUnsupportedMediaType, CodeValidation, "Format kiriman tidak didukung.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<10)

		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" ||
			bcrypt.CompareHashAndPassword(cfg.AdminPasswordHash, []byte(req.Password)) != nil {
			deps.Log.Warn("login admin gagal", "request_id", c.GetString(requestIDKey))
			writeError(c, http.StatusUnauthorized, CodeInvalidPassword, "Sandi salah. Coba lagi.", nil)
			return
		}

		token := adminauth.Sign(cfg.SessionSecret, time.Now().Add(adminauth.TTL))
		setAdminCookie(c, cfg, token, int(adminauth.TTL.Seconds()))
		deps.Log.Info("login admin berhasil", "request_id", c.GetString(requestIDKey))
		c.Status(http.StatusNoContent)
	}
}

// Hanya menghapus cookie di browser ini. Token yang sempat disalin tetap berlaku sampai kedaluwarsa;
// untuk membatalkan semuanya, ganti SESSION_SECRET.
func logoutHandler(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		setAdminCookie(c, cfg, "", -1)
		c.Status(http.StatusNoContent)
	}
}

func requireAdmin(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(adminCookie)
		if err != nil || !adminauth.Valid(cfg.SessionSecret, token, time.Now()) {
			writeError(c, http.StatusUnauthorized, CodeUnauthorized, "Sesi admin berakhir. Masuk lagi.", nil)
			return
		}
		// Data peserta tidak boleh tersimpan di cache browser atau proxy.
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

func setAdminCookie(c *gin.Context, cfg config.Config, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     adminCookie,
		Value:    value,
		Path:     "/api/admin",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}
