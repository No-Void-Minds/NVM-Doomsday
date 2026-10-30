package httpapi

import (
	"crypto/rand"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

// ID dari klien sengaja tidak dipakai, supaya log tidak bisa diisi ID palsu.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := rand.Text()
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Tanpa IP, query string, atau body (NFR-08). Route dicatat sebagai pola, karena URL mentah
// bisa berisi nama orang dari parameter ?to=.
func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		log.Info("request",
			"request_id", c.GetString(requestIDKey),
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

// Pengganti gin.Recovery: stack trace masuk log, klien hanya menerima error JSON yang seragam.
func recoverJSON(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		log.Error("panic",
			"request_id", c.GetString(requestIDKey),
			"panic", recovered,
			"stack", string(debug.Stack()),
		)
		writeError(c, http.StatusInternalServerError, CodeInternal,
			"Terjadi kesalahan di server. Coba lagi sebentar.", nil)
	})
}
