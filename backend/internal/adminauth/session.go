package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

const TTL = 8 * time.Hour

// Token berbentuk "<unix kedaluwarsa>.<hmac>". Server tidak menyimpan sesi; mengganti SESSION_SECRET
// otomatis membatalkan semua sesi yang beredar.
func Sign(secret []byte, expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	return exp + "." + mac(secret, exp)
}

func Valid(secret []byte, token string, now time.Time) bool {
	exp, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	// Dibandingkan dalam waktu konstan supaya tanda tangan tidak bisa ditebak dari lamanya respons.
	if !hmac.Equal([]byte(sig), []byte(mac(secret, exp))) {
		return false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return false
	}
	return now.Before(time.Unix(unix, 0))
}

func mac(secret []byte, exp string) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte("doomsday-admin|" + exp))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
