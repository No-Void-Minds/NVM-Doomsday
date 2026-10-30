package registration

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	codePrefix = "NVM-DD-"
	codeLength = 6
	// Tanpa O, I, L, 0, 1 karena mudah tertukar saat dibacakan atau diketik ulang (FR-21).
	// Harus sama dengan CHECK kolom registrations.code di migrasi 0001.
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

func NewCode() (string, error) {
	max := big.NewInt(int64(len(codeAlphabet)))
	b := make([]byte, codeLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("acak kode registrasi: %w", err)
		}
		b[i] = codeAlphabet[n.Int64()]
	}
	return codePrefix + string(b), nil
}

// NormalizeCode merapikan kode yang diketik manusia, misalnya di CLI admin atau saat check-in:
// spasi dibuang dan huruf dijadikan kapital. Mengembalikan false kalau bentuknya bukan kode yang sah.
func NormalizeCode(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(s, codePrefix) || len(s) != len(codePrefix)+codeLength {
		return "", false
	}
	for _, r := range s[len(codePrefix):] {
		if !strings.ContainsRune(codeAlphabet, r) {
			return "", false
		}
	}
	return s, true
}
