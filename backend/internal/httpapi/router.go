package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/config"
)

func NewRouter(cfg config.Config) (*gin.Engine, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Belum ada proxy yang dipercaya. Daftar IP Nginx diisi di Sprint 2 bersama rate limit,
	// karena rate limit per IP butuh X-Forwarded-For yang hanya boleh dibaca dari Nginx.
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("atur trusted proxies: %w", err)
	}

	api := r.Group("/api")
	api.GET("/health", health)

	return r, nil
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}