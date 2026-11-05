package registration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const validKey = "7b0c2f3e-9a41-4d2b-8c55-0f6e1a2b3c4d"

func validRequest() Request {
	return Request{
		Attendees:      []AttendeeRequest{{FullName: "Budi", Email: "budi@email.com", AccountName: "budi"}},
		Consent:        true,
		IdempotencyKey: validKey,
	}
}

func errorMap(errs []FieldError) map[string]string {
	m := make(map[string]string, len(errs))
	for _, e := range errs {
		m[e.Field] = e.Message
	}
	return m
}

type sharedValidationCase struct {
	Nama      string            `json:"nama"`
	Attendees []AttendeeRequest `json:"attendees"`
	Consent   bool              `json:"consent"`
	Errors    map[string]string `json:"errors"`
}

// File yang sama dibaca Vitest, jadi aturan dan pesan browser dijamin sama dengan backend.
func TestValidate_KasusBersama(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "validasi-pendaftaran.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Kasus []sharedValidationCase `json:"kasus"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Kasus) == 0 {
		t.Fatal("file kasus kosong")
	}

	for _, c := range file.Kasus {
		t.Run(c.Nama, func(t *testing.T) {
			req := Request{Attendees: c.Attendees, Consent: c.Consent, IdempotencyKey: validKey}
			_, errs := Validate(req)
			got := errorMap(errs)
			if len(got) != len(c.Errors) {
				t.Fatalf("error = %v, mau %v", got, c.Errors)
			}
			for field, want := range c.Errors {
				if got[field] != want {
					t.Errorf("%s = %q, mau %q", field, got[field], want)
				}
			}
		})
	}
}

func TestValidate_HasilDibersihkan(t *testing.T) {
	req := validRequest()
	req.Attendees = []AttendeeRequest{
		{FullName: "  Budi Santoso ", Email: " Budi@Email.COM ", AccountName: " @Budi "},
		{FullName: "Sari", Email: "sari@email.com", AccountName: "sari"},
	}
	in, errs := Validate(req)
	if len(errs) > 0 {
		t.Fatalf("error tidak diharapkan: %v", errs)
	}
	want := Attendee{FullName: "Budi Santoso", Email: "budi@email.com", AccountName: "@Budi"}
	if in.Attendees[0] != want {
		t.Errorf("attendees[0] = %+v, mau %+v", in.Attendees[0], want)
	}
	if in.IdempotencyKey != uuid.MustParse(validKey) {
		t.Errorf("idempotency key = %s", in.IdempotencyKey)
	}
}

func TestValidate_Honeypot(t *testing.T) {
	req := validRequest()
	req.Website = "http://spam.example"
	_, errs := Validate(req)
	if len(errs) != 1 || errs[0].Field != "" || errs[0].Message != "Pendaftaran tidak bisa diproses." {
		t.Fatalf("errs = %+v", errs)
	}

	// Spasi saja bukan isian bot, misalnya dari isi otomatis browser.
	req.Website = "  "
	if _, errs := Validate(req); len(errs) > 0 {
		t.Fatalf("honeypot berisi spasi ditolak: %+v", errs)
	}
}

func TestValidate_IdempotencyKey(t *testing.T) {
	for _, key := range []string{"", "bukan-uuid", "7b0c2f3e-9a41-4d2b-8c55"} {
		req := validRequest()
		req.IdempotencyKey = key
		_, errs := Validate(req)
		if got := errorMap(errs)["idempotency_key"]; got != "Muat ulang halaman, lalu coba kirim lagi." {
			t.Errorf("key %q: pesan = %q", key, got)
		}
	}
}

func TestValidate_JumlahPeserta(t *testing.T) {
	req := validRequest()
	req.Attendees = nil
	_, errs := Validate(req)
	if got := errorMap(errs)["attendees"]; got != "Isi data kamu." {
		t.Errorf("tanpa peserta: %q", got)
	}

	person := AttendeeRequest{FullName: "A", Email: "a@email.com", AccountName: "a"}
	req.Attendees = make([]AttendeeRequest, 4)
	for i := range req.Attendees {
		p := person
		p.Email = string(rune('a'+i)) + "@email.com"
		req.Attendees[i] = p
	}
	if _, errs := Validate(req); len(errs) > 0 {
		t.Errorf("pendaftar dan 3 tamu ditolak: %v", errs)
	}

	// Lima peserta kosong: hanya satu pesan, karena pemeriksaan berhenti sebelum memeriksa tiap baris.
	req.Attendees = make([]AttendeeRequest, 5)
	_, errs = Validate(req)
	if len(errs) != 1 || errs[0].Field != "attendees" || errs[0].Message != "Maksimal 3 tamu." {
		t.Errorf("lima peserta: %+v", errs)
	}
}

func TestValidate_PanjangDihitungPerKarakter(t *testing.T) {
	cases := []struct {
		name, field, value, want string
		apply                    func(*AttendeeRequest, string)
	}{
		{"nama 100 huruf beraksen", "attendees[0].full_name", strings.Repeat("é", 100), "",
			func(a *AttendeeRequest, v string) { a.FullName = v }},
		{"nama 101", "attendees[0].full_name", strings.Repeat("a", 101), "Nama lengkap maksimal 100 karakter.",
			func(a *AttendeeRequest, v string) { a.FullName = v }},
		{"nama akun 50", "attendees[0].account_name", strings.Repeat("é", 50), "",
			func(a *AttendeeRequest, v string) { a.AccountName = v }},
		{"nama akun 51", "attendees[0].account_name", strings.Repeat("a", 51), "Nama akun maksimal 50 karakter.",
			func(a *AttendeeRequest, v string) { a.AccountName = v }},
		{"email 255", "attendees[0].email", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 186) + ".com", "Email terlalu panjang.",
			func(a *AttendeeRequest, v string) { a.Email = v }},
		{"nama UTF-8 rusak", "attendees[0].full_name", "Budi\xff", "Nama lengkap berisi karakter yang tidak valid.",
			func(a *AttendeeRequest, v string) { a.FullName = v }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validRequest()
			c.apply(&req.Attendees[0], c.value)
			_, errs := Validate(req)
			if got := errorMap(errs)[c.field]; got != c.want {
				t.Errorf("%s = %q, mau %q", c.field, got, c.want)
			}
		})
	}
}

// Nama akun pendaftar dicocokkan dengan daftar VIP, jadi "@" saja ditolak. Nama akun tamu tidak dicocokkan.
func TestValidate_NamaAkunHanyaAt(t *testing.T) {
	req := validRequest()
	req.Attendees[0].AccountName = "@@"
	req.Attendees = append(req.Attendees, AttendeeRequest{FullName: "Sari", Email: "sari@email.com", AccountName: "@"})
	_, errs := Validate(req)
	got := errorMap(errs)
	if got["attendees[0].account_name"] != "Nama akun wajib diisi." {
		t.Errorf("pendaftar: %q", got["attendees[0].account_name"])
	}
	if _, ok := got["attendees[1].account_name"]; ok {
		t.Errorf("tamu dengan nama akun @ ikut ditolak: %v", got)
	}
}

func TestValidateAttendee(t *testing.T) {
	out, errs := ValidateAttendee(AttendeeRequest{FullName: " Sari ", Email: "SARI@Email.com", AccountName: " sari "})
	if len(errs) > 0 {
		t.Fatalf("errs = %v", errs)
	}
	if out != (Attendee{FullName: "Sari", Email: "sari@email.com", AccountName: "sari"}) {
		t.Errorf("out = %+v", out)
	}

	_, errs = ValidateAttendee(AttendeeRequest{})
	got := errorMap(errs)
	want := map[string]string{
		"full_name":    "Nama lengkap wajib diisi.",
		"email":        "Email wajib diisi.",
		"account_name": "Nama akun wajib diisi.",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, mau %q", k, got[k], v)
		}
	}
}
