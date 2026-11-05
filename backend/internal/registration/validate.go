package registration

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Batas yang diizinkan skema database (CHECK posisi 0 sampai 3). Batas yang sedang berlaku,
// event_config.max_guests_vip, dicek lagi di dalam transaksi pendaftaran.
const maxGuestsHard = 3

type AttendeeRequest struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	AccountName string `json:"account_name"`
}

// Tipe Solo atau VIP tidak dikirim browser. Server menentukannya dari daftar VIP saat mendaftar.
type Request struct {
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
	}

	seen := make(map[string]int, n)
	for i, a := range req.Attendees {
		prefix := fmt.Sprintf("attendees[%d]", i)
		name := strings.TrimSpace(a.FullName)
		email := strings.ToLower(strings.TrimSpace(a.Email))
		account := strings.TrimSpace(a.AccountName)

		checkText(name, 100, prefix+".full_name", "Nama lengkap", add)
		checkText(account, 50, prefix+".account_name", "Nama akun", add)
		// Nama akun pendaftar dicocokkan dengan daftar VIP, jadi kuncinya tidak boleh kosong.
		if i == 0 && account != "" && NormalizeAccount(account) == "" {
			add(prefix+".account_name", "Nama akun wajib diisi.")
		}

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

// ValidateAttendee memakai aturan yang sama dengan formulir, untuk jalur lain seperti CLI admin.
func ValidateAttendee(a AttendeeRequest) (Attendee, []FieldError) {
	var errs []FieldError
	add := func(field, msg string) {
		errs = append(errs, FieldError{Field: field, Message: msg})
	}

	out := Attendee{
		FullName:    strings.TrimSpace(a.FullName),
		Email:       strings.ToLower(strings.TrimSpace(a.Email)),
		AccountName: strings.TrimSpace(a.AccountName),
	}
	checkText(out.FullName, 100, "full_name", "Nama lengkap", add)
	checkText(out.AccountName, 50, "account_name", "Nama akun", add)
	if msg := emailProblem(out.Email); msg != "" {
		add("email", msg)
	}
	return out, errs
}
