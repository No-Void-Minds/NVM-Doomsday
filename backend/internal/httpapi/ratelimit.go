package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

type ipLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	every    rate.Limit
	burst    int
}

func newIPLimiter(every time.Duration, burst int) *ipLimiter {
	return &ipLimiter{visitors: map[string]*visitor{}, every: rate.Every(every), burst: burst}
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, ok := l.visitors[ip]
	if !ok {
		// Dibersihkan saat peta membesar, supaya banjir IP berbeda tidak menghabiskan memori.
		if len(l.visitors) >= 10000 {
			for k, old := range l.visitors {
				if now.Sub(old.seen) > 10*time.Minute {
					delete(l.visitors, k)
				}
			}
		}
		v = &visitor{limiter: rate.NewLimiter(l.every, l.burst)}
		l.visitors[ip] = v
	}
	v.seen = now
	return v.limiter.Allow()
}

// IP dari c.ClientIP() hanya membaca X-Forwarded-For bila request datang dari proxy di TRUSTED_PROXIES.
// Tanpa itu, siapa pun bisa memalsukan header untuk lolos dari batas.
func rateLimit(l *ipLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP()) {
			c.Header("Retry-After", "10")
			writeError(c, http.StatusTooManyRequests, CodeRateLimited,
				"Terlalu banyak percobaan. Tunggu sebentar, lalu coba lagi.", nil)
			return
		}
		c.Next()
	}
}
