package registration

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

// Batas yang diizinkan skema database (CHECK posisi 0 sampai 3). Batas yang sedang berlaku,
// event_config.max_guests_vip, dicek lagi di dalam transaksi pendaftaran.
const maxGuestsHard = 3

type AttendeeRequest struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	AccountName string `json:"account_name"`
}

type Request struct {
	Type           string            `json:"type"`
	Attendees      []AttendeeRequest `json:"attendees"`
	Consent        bool              `json:"consent"`
	IdempotencyKey string            `json:"idempotency_key"`
	// Honeypot: kolom tersembunyi di formulir. Bot pengisi otomatis biasanya ikut mengisinya.
	Website string `json:"website"`
}

type Attendee struct {
	FullName    string
	Email       string
	AccountName string
}

// Input adalah hasil validasi yang sudah bersih: spasi dipangkas, email huruf kecil.
type Input struct {
	Type           sqlc.RegistrationType
	Attendees      []Attendee
	IdempotencyKey uuid.UUID
}

// Field memakai path yang sama dengan body JSON, misalnya "attendees[1].email",
// supaya frontend bisa menaruh pesan tepat di bawah kolom yang salah.
type FieldError struct {
	Field   string
	Message string
}

func Validate(req Request) (Input, []FieldError) {
	var errs []FieldError
	add := func(field, msg string) {
		errs = append(errs, FieldError{Field: field, Message: msg})
	}

	// Pesan sengaja umum, supaya pembuat bot tidak tahu kolom mana yang menjebaknya.
	if strings.TrimSpace(req.Website) != "" {
		return Input{}, []FieldError{{Message: "Pendaftaran tidak bisa diproses."}}
	}

	var in Input
	switch req.Type {
	case string(sqlc.RegistrationTypeSolo):
		in.Type = sqlc.RegistrationTypeSolo
	case string(sqlc.RegistrationTypeVip):
		in.Type = sqlc.RegistrationTypeVip
	default:
		add("type", "Pilih Solo atau VIP.")
	}

	key, err := uuid.Parse(req.IdempotencyKey)
	if err != nil {
		add("idempotency_key", "Muat ulang halaman, lalu coba kirim lagi.")
	} else {
		in.IdempotencyKey = key
	}

	if !req.Consent {
		add("consent", "Centang persetujuan penggunaan data untuk melanjutkan.")
	}

	n := len(req.Attendees)
	switch {
	case n == 0:
		add("attendees", "Isi data kamu.")
	case n > 1+maxGuestsHard:
		// Berhenti di sini supaya body dengan ribuan baris tidak ikut diperiksa satu per satu.
		add("attendees", fmt.Sprintf("Maksimal %d tamu.", maxGuestsHard))
		return Input{}, errs
	case in.Type == sqlc.RegistrationTypeSolo && n != 1:
		add("attendees", "Solo hanya untuk satu orang. Pilih VIP kalau mau membawa tamu.")
	case in.Type == sqlc.RegistrationTypeVip && n < 2:
		add("attendees", "VIP butuh minimal satu tamu. Pilih Solo kalau datang sendiri.")
	}

	seen := make(map[string]int, n)
	for i, a := range req.Attendees {
		prefix := fmt.Sprintf("attendees[%d]", i)
		name := strings.TrimSpace(a.FullName)
		email := strings.ToLower(strings.TrimSpace(a.Email))
		account := strings.TrimSpace(a.AccountName)

		checkText(name, 100, prefix+".full_name", "Nama lengkap", add)
		checkText(account, 50, prefix+".account_name", "Nama akun", add)

		if msg := emailProblem(email); msg != "" {
			add(prefix+".email", msg)
		} else if first, dup := seen[email]; dup {
			add(prefix+".email", fmt.Sprintf("Email ini sudah dipakai di %s.", rowLabel(first)))
		} else {
			seen[email] = i
		}

		in.Attendees = append(in.Attendees, Attendee{FullName: name, Email: email, AccountName: account})
	}

	if len(errs) > 0 {
		return Input{}, errs
	}
	return in, nil
}

func checkText(v string, max int, field, label string, add func(string, string)) {
	switch {
	case v == "":
		add(field, label+" wajib diisi.")
	case !utf8.ValidString(v) || strings.ContainsFunc(v, unicode.IsControl):
		add(field, label+" berisi karakter yang tidak valid.")
	// Dihitung per karakter, bukan per byte, sama seperti varchar di Postgres.
	case utf8.RuneCountInString(v) > max:
		add(field, fmt.Sprintf("%s maksimal %d karakter.", label, max))
	}
}

func emailProblem(email string) string {
	const format = "Format email belum benar, contoh: nama@email.com."
	switch {
	case email == "":
		return "Email wajib diisi."
	case len(email) > 254:
		return "Email terlalu panjang."
	}

	addr, err := mail.ParseAddress(email)
	// ParseAddress juga menerima bentuk "Nama <email>". Yang diterima hanya alamat polos.
	if err != nil || addr.Address != email {
		return format
	}
	// Selaras dengan CHECK di database: domain harus punya titik.
	domain := email[strings.LastIndexByte(email, '@')+1:]
	if !strings.Contains(domain, ".") {
		return format
	}
	return ""
}

func rowLabel(i int) string {
	if i == 0 {
		return "data kamu"
	}
	return fmt.Sprintf("Tamu %d", i)
}
