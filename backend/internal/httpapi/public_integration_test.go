//go:build integration

package httpapi

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var codePattern = regexp.MustCompile(`^NVM-DD-[A-HJKMNP-Z2-9]{6}$`)

func TestEvent_IsiBawaan(t *testing.T) {
	a := newAPI(t)
	w := a.send(http.MethodGet, "/api/event", nil)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("event = %d, Cache-Control %q", w.Code, w.Header().Get("Cache-Control"))
	}
	body := w.Body.String()
	// Field yang belum diisi admin dikirim null, bukan string kosong, supaya frontend menampilkan "Menyusul".
	for _, f := range []string{"cinema_name", "city", "studio", "gather_at", "show_at", "map_url"} {
		if !strings.Contains(body, `"`+f+`":null`) {
			t.Errorf("%s bukan null: %s", f, body)
		}
	}
	ev := decode[map[string]any](t, w)
	want := map[string]any{
		"title": "No Void Minds Doomsday", "film_title": "Avengers: Doomsday", "capacity": 200.0,
		"max_guests_vip": 3.0, "is_open": true, "seats_left": 200.0, "waitlisted_attendees": 0.0,
	}
	for k, v := range want {
		if ev[k] != v {
			t.Errorf("%s = %v, mau %v", k, ev[k], v)
		}
	}
}

func TestEvent_WaktuUTCDanSisaKursi(t *testing.T) {
	a := newAPI(t)
	a.env.Exec(t, `UPDATE event_config SET capacity = 2, show_at = '2026-12-16 19:00+07', gather_at = '2026-12-16 18:15+07'`)
	a.mustRegister(p("andi", "andi"))
	a.mustRegister(p("budi", "budi"))
	a.mustRegister(p("citra", "citra"))

	ev := decode[map[string]any](t, a.send(http.MethodGet, "/api/event", nil))
	if ev["show_at"] != "2026-12-16T12:00:00Z" || ev["gather_at"] != "2026-12-16T11:15:00Z" {
		t.Errorf("waktu bukan UTC: show_at %v, gather_at %v", ev["show_at"], ev["gather_at"])
	}
	if ev["seats_left"] != 0.0 || ev["waitlisted_attendees"] != 1.0 {
		t.Errorf("seats_left %v, waitlisted %v", ev["seats_left"], ev["waitlisted_attendees"])
	}

	// Kapasitas diturunkan di bawah jumlah terkonfirmasi lewat CLI: tetap 0, tidak negatif.
	a.env.Exec(t, `UPDATE event_config SET capacity = 1`)
	ev = decode[map[string]any](t, a.send(http.MethodGet, "/api/event", nil))
	if ev["seats_left"] != 0.0 {
		t.Errorf("seats_left = %v setelah kapasitas diturunkan", ev["seats_left"])
	}
}

func TestRegistrasi_SoloBaru201(t *testing.T) {
	a := newAPI(t)
	body := newReg(person{FullName: "  Budi Santoso ", Email: " Budi@Email.COM ", AccountName: "budi_nvm"})
	w := a.register(body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d %s", w.Code, w.Body)
	}
	res := decode[regResult](t, w)
	if !codePattern.MatchString(res.Code) || res.Type != "solo" || res.Status != "confirmed" || res.WaitlistPosition != nil {
		t.Errorf("hasil = %+v", res)
	}
	if strings.Join(res.Attendees, ",") != "Budi Santoso" {
		t.Errorf("peserta = %v", res.Attendees)
	}
	if !strings.Contains(w.Body.String(), `"waitlist_position":null`) {
		t.Errorf("waitlist_position bukan null: %s", w.Body)
	}
	n := a.env.QueryInt(t, `SELECT count(*) FROM registrations r JOIN attendees x ON x.registration_id = r.id
		WHERE r.code = $1 AND r.type = 'solo' AND r.status = 'confirmed' AND r.consent_at IS NOT NULL
		AND x.email = 'budi@email.com' AND x.full_name = 'Budi Santoso' AND x.role = 'primary' AND x.position = 0`, res.Code)
	if n != 1 {
		t.Errorf("baris di database tidak sesuai (%d)", n)
	}
}

func TestRegistrasi_Idempotency201Lalu200(t *testing.T) {
	a := newAPI(t)
	body := newReg(p("budi", "budi"))
	first := a.register(body)
	second := a.register(body)
	if first.Code != http.StatusCreated || second.Code != http.StatusOK {
		t.Fatalf("status = %d lalu %d, mau 201 lalu 200", first.Code, second.Code)
	}
	r1, r2 := decode[regResult](t, first), decode[regResult](t, second)
	if r1.Code != r2.Code || r1.Status != r2.Status {
		t.Errorf("kiriman ulang memberi hasil berbeda: %+v vs %+v", r1, r2)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM registrations`); n != 1 {
		t.Errorf("%d registrasi tersimpan, mau 1", n)
	}

	// Key yang sama tetap mengembalikan hasil lama walau pendaftaran sudah ditutup, supaya kiriman ulang
	// setelah sinyal putus tetap mendapat kartu yang sama.
	a.env.Exec(t, `UPDATE event_config SET is_open = false`)
	if w := a.register(body); w.Code != http.StatusOK || decode[regResult](t, w).Code != r1.Code {
		t.Errorf("kiriman ulang saat ditutup = %d %s", w.Code, w.Body)
	}
}

func TestRegistrasi_EmailUnik409(t *testing.T) {
	a := newAPI(t)
	a.mustRegister(person{FullName: "Budi", Email: "Budi@Email.com", AccountName: "budi"})

	e := expectError(t, a.register(newReg(person{FullName: "Budi Dua", Email: "budi@EMAIL.com", AccountName: "budi2"})),
		http.StatusConflict, CodeDuplicateEmail)
	if fieldMessage(e, "attendees[0].email") != "Email ini sudah terdaftar." {
		t.Errorf("pesan kolom = %+v", e.Error.Fields)
	}

	// Email tamu yang sudah terdaftar ditunjuk di kolom tamu itu, dan tidak ada baris baru tersimpan.
	a.addVip("vipsatu")
	e = expectError(t, a.register(newReg(p("sari", "vipsatu"), person{FullName: "Tamu", Email: "BUDI@email.com", AccountName: "tamu"})),
		http.StatusConflict, CodeDuplicateEmail)
	if fieldMessage(e, "attendees[1].email") != "Email ini sudah terdaftar." || fieldMessage(e, "attendees[0].email") != "" {
		t.Errorf("pesan kolom = %+v", e.Error.Fields)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM registrations`); n != 1 {
		t.Errorf("%d registrasi tersimpan, mau 1", n)
	}
}

func TestRegistrasi_BatalMembebaskanEmail(t *testing.T) {
	a := newAPI(t)
	first := a.mustRegister(p("budi", "budi"))
	a.cancel(first.Code)
	again := a.register(newReg(p("budi", "budi")))
	if again.Code != http.StatusCreated {
		t.Fatalf("daftar ulang setelah batal = %d %s", again.Code, again.Body)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM attendees WHERE email = 'budi@email.com' AND is_active`); n != 1 {
		t.Errorf("%d peserta aktif dengan email itu, mau 1", n)
	}
}

func TestRegistrasi_Ditutup409(t *testing.T) {
	a := newAPI(t)
	a.env.Exec(t, `UPDATE event_config SET is_open = false`)
	e := expectError(t, a.register(newReg(p("budi", "budi"))), http.StatusConflict, CodeClosed)
	if e.Error.Message != "Pendaftaran sudah ditutup." {
		t.Errorf("pesan = %q", e.Error.Message)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM registrations`); n != 0 {
		t.Errorf("%d registrasi tersimpan saat ditutup", n)
	}
}

func TestRegistrasi_Honeypot(t *testing.T) {
	a := newAPI(t)
	body := newReg(p("budi", "budi"))
	body.Website = "http://spam.example"
	e := expectError(t, a.register(body), http.StatusUnprocessableEntity, CodeValidation)
	if len(e.Error.Fields) != 1 || e.Error.Fields[0].Message != "Pendaftaran tidak bisa diproses." {
		t.Errorf("fields = %+v", e.Error.Fields)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM registrations`); n != 0 {
		t.Errorf("isian bot tersimpan")
	}
}

func TestRegistrasi_TipeDitentukanServer(t *testing.T) {
	a := newAPI(t)
	a.addVip("UjiVIP")

	// Nama ternormalisasi cocok dengan daftar VIP.
	res := a.mustRegister(person{FullName: "Vina", Email: "vina@email.com", AccountName: " @ujivip "}, p("tamu1", "tamu1"))
	if res.Type != "vip" || len(res.Attendees) != 2 {
		t.Errorf("hasil = %+v", res)
	}

	// Field type dari browser diabaikan.
	body := `{"type":"vip","attendees":[{"full_name":"Lain","email":"lain@email.com","account_name":"lainnya"}],"consent":true,"idempotency_key":"` + newReg().IdempotencyKey + `"}`
	if w := a.send(http.MethodPost, "/api/registrations", body); w.Code != http.StatusCreated || decode[regResult](t, w).Type != "solo" {
		t.Errorf("type dari browser dipakai: %d %s", w.Code, w.Body)
	}

	// Bukan VIP tapi membawa tamu.
	e := expectError(t, a.register(newReg(p("solo", "bukanvip"), p("tamu2", "tamu2"))), http.StatusUnprocessableEntity, CodeValidation)
	if fieldMessage(e, "attendees") != "Nama akun ini tidak ada di daftar VIP, jadi pendaftaran hanya untuk kamu sendiri." {
		t.Errorf("fields = %+v", e.Error.Fields)
	}
}

func TestRegistrasi_BatasTamuVIP(t *testing.T) {
	a := newAPI(t)
	a.addVip("vipa", "vipb", "vipc")
	a.env.Exec(t, `UPDATE event_config SET max_guests_vip = 2`)

	e := expectError(t, a.register(newReg(p("a0", "vipa"), p("a1", "x"), p("a2", "x"), p("a3", "x"))), http.StatusUnprocessableEntity, CodeValidation)
	if fieldMessage(e, "attendees") != "Maksimal 2 tamu." {
		t.Errorf("fields = %+v", e.Error.Fields)
	}
	if res := a.mustRegister(p("b0", "vipb"), p("b1", "x"), p("b2", "x")); res.Type != "vip" {
		t.Errorf("VIP dua tamu = %+v", res)
	}
	// VIP boleh datang tanpa tamu (keputusan Sprint 9).
	if res := a.mustRegister(p("c0", "vipc")); res.Type != "vip" {
		t.Errorf("VIP tanpa tamu = %+v", res)
	}
	// Batas mutlak 3 tamu dicek sebelum database.
	e = expectError(t, a.register(newReg(p("d0", "vipa"), p("d1", "x"), p("d2", "x"), p("d3", "x"), p("d4", "x"))), http.StatusUnprocessableEntity, CodeValidation)
	if fieldMessage(e, "attendees") != "Maksimal 3 tamu." {
		t.Errorf("fields = %+v", e.Error.Fields)
	}
}

// FR-17: rombongan masuk semua atau semua ke daftar tunggu, nomor urut per registrasi.
func TestRegistrasi_RombonganTidakDipecah(t *testing.T) {
	a := newAPI(t)
	a.addVip("vipa", "vipb")
	a.env.Exec(t, `UPDATE event_config SET capacity = 3`)

	if r := a.mustRegister(p("solo1", "solo1")); r.Status != "confirmed" {
		t.Fatalf("solo pertama = %+v", r)
	}
	big := a.mustRegister(p("a0", "vipa"), p("a1", "x"), p("a2", "x"))
	if big.Status != "waitlisted" || big.WaitlistPosition == nil || *big.WaitlistPosition != 1 || len(big.Attendees) != 3 {
		t.Errorf("rombongan 3 orang di sisa 2 kursi = %+v", big)
	}
	if r := a.mustRegister(p("solo2", "solo2")); r.Status != "confirmed" {
		t.Errorf("solo kedua masih muat = %+v", r)
	}
	pair := a.mustRegister(p("b0", "vipb"), p("b1", "x"))
	if pair.Status != "waitlisted" || *pair.WaitlistPosition != 2 {
		t.Errorf("rombongan 2 orang di sisa 1 kursi = %+v", pair)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM attendees x JOIN registrations r ON r.id = x.registration_id WHERE r.status = 'confirmed'`); n != 2 {
		t.Errorf("%d peserta terkonfirmasi, mau 2", n)
	}

	full := a.mustRegister(p("solo3", "solo3"))
	if full.Status != "confirmed" {
		t.Errorf("solo ketiga mengisi kursi terakhir = %+v", full)
	}
	last := a.mustRegister(p("solo4", "solo4"))
	if last.Status != "waitlisted" || *last.WaitlistPosition != 3 {
		t.Errorf("solo saat penuh = %+v", last)
	}
}

func TestRegistrasi_NamaVIPSekaliPakai(t *testing.T) {
	a := newAPI(t)
	a.addVip("ujivip")
	first := a.mustRegister(p("vina", "ujivip"))
	if first.Type != "vip" {
		t.Fatalf("pendaftaran pertama = %+v", first)
	}

	e := expectError(t, a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "@UjiVIP"}), http.StatusConflict, CodeAccountTaken)
	if fieldMessage(e, "account_name") != "Nama akun ini sudah terdaftar. Hubungi admin NVM." {
		t.Errorf("cek kedua = %+v", e.Error)
	}
	e = expectError(t, a.register(newReg(p("lain", " ujivip"))), http.StatusConflict, CodeAccountTaken)
	if fieldMessage(e, "attendees[0].account_name") != "Nama akun ini sudah terdaftar. Hubungi admin NVM." {
		t.Errorf("daftar kedua = %+v", e.Error)
	}

	a.cancel(first.Code)
	if w := a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "ujivip"}); w.Code != http.StatusOK || decode[map[string]string](t, w)["type"] != "vip" {
		t.Errorf("cek setelah batal = %d %s", w.Code, w.Body)
	}
	if r := a.mustRegister(p("lain", "ujivip")); r.Type != "vip" {
		t.Errorf("daftar setelah batal = %+v", r)
	}
}

func TestRegistrasi_RateLimit(t *testing.T) {
	a := newAPI(t)
	ip := fromIP("192.0.2.10:1234")
	for i := 1; i <= 10; i++ {
		if w := a.send(http.MethodPost, "/api/registrations", `{}`, ip); w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("kiriman %d = %d", i, w.Code)
		}
	}
	w := a.send(http.MethodPost, "/api/registrations", newReg(p("budi", "budi")), ip)
	expectError(t, w, http.StatusTooManyRequests, CodeRateLimited)
	if w.Header().Get("Retry-After") == "" {
		t.Error("tanpa Retry-After")
	}
	// IP lain tidak terpengaruh.
	if w := a.register(newReg(p("sari", "sari"))); w.Code != http.StatusCreated {
		t.Errorf("IP lain = %d", w.Code)
	}
}

func TestAccountCheck(t *testing.T) {
	a := newAPI(t)
	a.addVip("UjiVIP", "nonaktif")
	a.login()
	list := decode[struct {
		Accounts []vipAccountResponse `json:"accounts"`
	}](t, a.send(http.MethodGet, "/api/admin/vip", nil))
	for _, acc := range list.Accounts {
		if acc.AccountName == "nonaktif" {
			if w := a.send(http.MethodPatch, "/api/admin/vip/"+acc.ID, map[string]bool{"is_active": false}); w.Code != http.StatusNoContent {
				t.Fatalf("nonaktifkan = %d", w.Code)
			}
		}
	}
	a.cookie = nil

	cases := map[string]string{"ujivip": "vip", " @UjiVIP ": "vip", "@@ujivip": "vip", "orang lain": "solo", "nonaktif": "solo"}
	for in, want := range cases {
		w := a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": in})
		if w.Code != http.StatusOK || decode[map[string]string](t, w)["type"] != want {
			t.Errorf("%q = %d %s, mau %s", in, w.Code, w.Body, want)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%q tanpa no-store", in)
		}
	}

	for _, in := range []string{"", "  ", "@", "@@", strings.Repeat("a", 51)} {
		expectError(t, a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": in}), http.StatusUnprocessableEntity, CodeValidation)
	}
	if w := a.send(http.MethodPost, "/api/account-check", "account_name=budi", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("bukan JSON = %d", w.Code)
	}
}

func TestAccountCheck_RateLimit(t *testing.T) {
	a := newAPI(t)
	ip := fromIP("192.0.2.20:1234")
	for i := 1; i <= 20; i++ {
		if w := a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "budi"}, ip); w.Code != http.StatusOK {
			t.Fatalf("cek %d = %d", i, w.Code)
		}
	}
	expectError(t, a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "budi"}, ip), http.StatusTooManyRequests, CodeRateLimited)
}
