package registration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

// ErrAccountTaken: nama akun VIP sudah dipakai registrasi lain yang belum batal.
var ErrAccountTaken = errors.New("nama akun VIP sudah dipakai")

// Dipakai API dan nanti frontend. Kontak admin menyusul setelah NVM memutuskannya.
const MsgAccountTaken = "Nama akun ini sudah terdaftar. Hubungi admin NVM."

// NormalizeAccount menghasilkan kunci pencocokan daftar VIP. Harus selaras dengan CHECK
// vip_accounts.account_key di migrasi 0004, karena index unik di sana bergantung pada bentuk ini.
func NormalizeAccount(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "@")
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateAccount untuk langkah pertama formulir: aturan yang sama dengan kolom nama akun pendaftar.
func ValidateAccount(raw string) (string, []FieldError) {
	var errs []FieldError
	add := func(field, msg string) {
		errs = append(errs, FieldError{Field: field, Message: msg})
	}

	account := strings.TrimSpace(raw)
	checkText(account, 50, "account_name", "Nama akun", add)
	// "@" saja lolos checkText, tapi kuncinya kosong dan tidak bisa dicocokkan.
	if len(errs) == 0 && NormalizeAccount(account) == "" {
		add("account_name", "Nama akun wajib diisi.")
	}
	return account, errs
}

// CheckAccount hanya petunjuk untuk tampilan formulir. Register mengambil keputusan ulang di bawah
// kunci event, karena di antara cek dan kirim nama itu bisa dinonaktifkan atau dipakai orang lain.
func (s *Service) CheckAccount(ctx context.Context, account string) (sqlc.RegistrationType, error) {
	vip, err := sqlc.New(s.pool).FindVipAccount(ctx, NormalizeAccount(account))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return sqlc.RegistrationTypeSolo, nil
	case err != nil:
		return "", fmt.Errorf("cari akun VIP: %w", err)
	case vip.Used:
		return "", ErrAccountTaken
	}
	return sqlc.RegistrationTypeVip, nil
}
