package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/config"
)

// Pinger dipisah sebagai interface supaya health check bisa diuji tanpa database sungguhan di Sprint 9.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Log *slog.Logger
	DB  Pinger
}

func NewRouter(cfg config.Config, deps Deps) (*gin.Engine, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Urutan penting: request ID dulu supaya tersedia untuk log, log membungkus recovery
	// supaya request yang panic tetap tercatat dengan status 500.
	r.Use(requestID(), accessLog(deps.Log), recoverJSON(deps.Log))

	// Belum ada proxy yang dipercaya. Daftar IP Nginx diisi di langkah 7 bersama rate limit,
	// karena rate limit per IP butuh X-Forwarded-For yang hanya boleh dibaca dari Nginx.
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("atur trusted proxies: %w", err)
	}

	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, CodeNotFound, "Alamat tidak ditemukan.", nil)
	})

	api := r.Group("/api")
	api.GET("/health", healthHandler(deps))

	return r, nil
}

func healthHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := deps.DB.Ping(ctx); err != nil {
			// Endpoint ini publik. Detail error (alamat, nama database) cukup di log server.
			deps.Log.Error("health: database tidak bisa dijangkau",
				"request_id", c.GetString(requestIDKey), "err", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "up"})
	}
}
