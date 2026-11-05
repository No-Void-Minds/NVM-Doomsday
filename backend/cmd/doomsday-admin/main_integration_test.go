//go:build integration

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/testdb"
)

// runCLI memanggil run seperti main, dengan stdout ditangkap. Perintah CLI mencetak langsung ke os.Stdout.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	out := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		out <- string(b)
	}()
	runErr := run(context.Background(), args)
	w.Close()
	os.Stdout = old
	return <-out, runErr
}

func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		io.WriteString(w, input)
		w.Close()
	}()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
}

type cliEnv struct {
	*testdb.Env
	svc *registration.Service
}

func setupCLI(t *testing.T) cliEnv {
	t.Helper()
	env := testdb.Setup(t)
	env.Reset(t)
	// CLI login sebagai doomsday_app, sama seperti di container produksi.
	t.Setenv("DATABASE_URL", env.AppURL)
	return cliEnv{Env: env, svc: registration.NewService(env.AppPool(t))}
}

func (e cliEnv) register(t *testing.T, people ...registration.AttendeeRequest) registration.Result {
	t.Helper()
	in, errs := registration.Validate(registration.Request{Attendees: people, Consent: true, IdempotencyKey: uuid.NewString()})
	if len(errs) > 0 {
		t.Fatalf("data uji tidak valid: %v", errs)
	}
	res, err := e.svc.Register(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func person(name string) registration.AttendeeRequest {
	return registration.AttendeeRequest{FullName: name, Email: name + "@email.com", AccountName: name}
}

// Lebar kolom tabwriter bergantung isi terpanjang, jadi spasi dirapatkan dulu sebelum dicocokkan.
func mustContain(t *testing.T, out string, parts ...string) {
	t.Helper()
	flat := strings.Join(strings.Fields(out), " ")
	for _, p := range parts {
		if !strings.Contains(flat, strings.Join(strings.Fields(p), " ")) {
			t.Errorf("keluaran tidak memuat %q:\n%s", p, out)
		}
	}
}

func wantErr(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Errorf("error = %v, mau memuat %q", err, contains)
	}
}

func TestCLI_PemakaianSalah(t *testing.T) {
	setupCLI(t)
	for _, args := range [][]string{
		nil, {"tidak-ada"}, {"event"}, {"event", "set", "capacity"}, {"vip"}, {"vip", "list", "x"},
		{"vip", "add"}, {"vip", "off"}, {"promote"}, {"cancel", "a", "b"}, {"rename", "NVM-DD-ABC234", "0"},
	} {
		if _, err := runCLI(t, args...); !errors.Is(err, errUsage) {
			t.Errorf("%v: error = %v, mau errUsage", args, err)
		}
	}
}

func TestCLI_TanpaDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := runCLI(t, "event", "show")
	wantErr(t, err, "DATABASE_URL belum diisi")
}

func TestCLI_HashPassword(t *testing.T) {
	withStdin(t, "sandi-admin-panjang\r\n")
	out, err := runCLI(t, "hash-password")
	if err != nil {
		t.Fatal(err)
	}
	hash := []byte(strings.TrimSpace(out))
	if cost, err := bcrypt.Cost(hash); err != nil || cost != 12 {
		t.Errorf("cost = %d, %v", cost, err)
	}
	// CRLF dari terminal Windows tidak ikut jadi bagian sandi.
	if bcrypt.CompareHashAndPassword(hash, []byte("sandi-admin-panjang")) != nil {
		t.Error("hash tidak cocok dengan sandi tanpa akhir baris")
	}

	withStdin(t, "pendek\n")
	_, err = runCLI(t, "hash-password")
	wantErr(t, err, "minimal 10 karakter")

	withStdin(t, strings.Repeat("a", 73)+"\n")
	_, err = runCLI(t, "hash-password")
	wantErr(t, err, "maksimal 72 byte")
}

func TestCLI_EventShow(t *testing.T) {
	e := setupCLI(t)
	e.register(t, person("budi"))
	out, err := runCLI(t, "event", "show")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "No Void Minds Doomsday", "Avengers: Doomsday", "(kosong)", "terkonfirmasi  1 dari 200 peserta",
		"daftar tunggu  0 peserta", "batal          0 registrasi")
}

func TestCLI_EventSet(t *testing.T) {
	e := setupCLI(t)
	ok := []struct {
		field, value, out, check string
	}{
		{"capacity", "5", "capacity: 200  ->  5", `SELECT count(*) FROM event_config WHERE capacity = 5`},
		{"max_guests_vip", "2", "max_guests_vip: 3  ->  2", `SELECT count(*) FROM event_config WHERE max_guests_vip = 2`},
		// Jam dibaca sebagai WIB: 19.00 WIB sama dengan 12.00 UTC.
		{"show_at", "2026-12-16 19:00", "show_at: (kosong)  ->  2026-12-16 19:00 WIB", `SELECT count(*) FROM event_config WHERE show_at = '2026-12-16 12:00Z'`},
		{"gather_at", "2026-12-16 18:15", "->  2026-12-16 18:15 WIB", `SELECT count(*) FROM event_config WHERE gather_at = '2026-12-16 11:15Z'`},
		{"cinema_name", "CGV Grand Indonesia", "->  CGV Grand Indonesia", `SELECT count(*) FROM event_config WHERE cinema_name = 'CGV Grand Indonesia'`},
		{"cinema_name", "-", "->  (kosong)", `SELECT count(*) FROM event_config WHERE cinema_name IS NULL`},
		{"map_url", "https://maps.example/cgv", "->  https://maps.example/cgv", `SELECT count(*) FROM event_config WHERE map_url = 'https://maps.example/cgv'`},
		{"map_url", "-", "->  (kosong)", `SELECT count(*) FROM event_config WHERE map_url IS NULL`},
		{"is_open", "false", "is_open: true  ->  false", `SELECT count(*) FROM event_config WHERE NOT is_open`},
		{"title", "  Nobar NVM  ", "->  Nobar NVM", `SELECT count(*) FROM event_config WHERE title = 'Nobar NVM'`},
	}
	for _, c := range ok {
		out, err := runCLI(t, "event", "set", c.field, c.value)
		if err != nil {
			t.Errorf("set %s %q: %v", c.field, c.value, err)
			continue
		}
		mustContain(t, out, c.out)
		if n := e.QueryInt(t, c.check); n != 1 {
			t.Errorf("set %s %q tidak tersimpan", c.field, c.value)
		}
	}

	bad := []struct{ field, value, msg string }{
		{"capacity", "0", "harus angka 1 sampai 10000"},
		{"capacity", "10001", "harus angka 1 sampai 10000"},
		{"capacity", "dua ratus", "harus angka 1 sampai 10000"},
		{"max_guests_vip", "4", "harus angka 1 sampai 3"},
		{"max_guests_vip", "0", "harus angka 1 sampai 3"},
		{"show_at", "16-12-2026 19:00", "YYYY-MM-DD HH:MM"},
		// Jam kumpul setelah jam tayang ditolak CHECK tabel.
		{"gather_at", "2026-12-16 20:00", "nilai ditolak aturan tabel"},
		{"map_url", "http://maps.example/cgv", "https://"},
		{"map_url", "maps.example", "https://"},
		{"is_open", "mungkin", "is_open"},
		{"title", "-", "tidak boleh kosong"},
		{"film_title", "", "tidak boleh kosong"},
		{"lokasi", "x", "tidak dikenal"},
	}
	for _, c := range bad {
		_, err := runCLI(t, "event", "set", c.field, c.value)
		wantErr(t, err, c.msg)
	}
	if n := e.QueryInt(t, `SELECT count(*) FROM event_config WHERE capacity = 5 AND max_guests_vip = 2 AND title = 'Nobar NVM'`); n != 1 {
		t.Error("nilai yang ditolak tetap mengubah event_config")
	}
}

func TestCLI_KapasitasDiBawahTerkonfirmasi(t *testing.T) {
	e := setupCLI(t)
	e.register(t, person("andi"))
	e.register(t, person("budi"))
	out, err := runCLI(t, "event", "set", "capacity", "1")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "Perhatian: sudah ada 2 peserta terkonfirmasi")
	if n := e.QueryInt(t, `SELECT count(*) FROM registrations WHERE status = 'confirmed'`); n != 2 {
		t.Error("registrasi dipindah otomatis padahal keputusan itu milik admin")
	}
}

func TestCLI_PromoteDanCancel(t *testing.T) {
	e := setupCLI(t)
	e.Exec(t, `UPDATE event_config SET capacity = 2`)
	e.Exec(t, `INSERT INTO vip_accounts (account_name, account_key) VALUES ('vipb', 'vipb')`)
	a := e.register(t, person("andi"))
	b := e.register(t, registration.AttendeeRequest{FullName: "Bayu", Email: "bayu@email.com", AccountName: "vipb"}, person("bella"))
	if a.Status != "confirmed" || b.Status != "waitlisted" {
		t.Fatalf("susunan awal = %s, %s", a.Status, b.Status)
	}

	_, err := runCLI(t, "promote", b.Code)
	wantErr(t, err, "kursi tidak cukup: rombongan 2 orang, sisa 1 kursi")
	_, err = runCLI(t, "promote", a.Code)
	wantErr(t, err, "hanya registrasi daftar tunggu")

	out, err := runCLI(t, "cancel", a.Code)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "status         confirmed      cancelled", "terkonfirmasi  1 dari 2", "0 dari 2",
		"Daftar tunggu teratas: "+b.Code+" (2 orang)", "doomsday-admin promote "+b.Code)

	// Kode boleh diketik huruf kecil dan berspasi, seperti saat dibacakan peserta.
	out, err = runCLI(t, "promote", "  "+strings.ToLower(b.Code)+" ")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "waitlisted", "confirmed", "2 dari 2")
	if n := e.QueryInt(t, `SELECT count(*) FROM registrations WHERE code = $1 AND status = 'confirmed'`, b.Code); n != 1 {
		t.Error("promote tidak tersimpan")
	}

	_, err = runCLI(t, "cancel", a.Code)
	wantErr(t, err, "sudah batal")
	_, err = runCLI(t, "cancel", "NVM-DD-ZZZZZZ")
	if !errors.Is(err, registration.ErrNotFound) {
		t.Errorf("kode tidak ada: %v", err)
	}
	_, err = runCLI(t, "promote", "NVM-DD-0000")
	wantErr(t, err, "format kode tidak valid")
}

func TestCLI_Rename(t *testing.T) {
	e := setupCLI(t)
	a := e.register(t, person("budi"))
	e.register(t, person("sari"))

	out, err := runCLI(t, "rename", a.Code, "0", "--name", " Budi Baru ", "--account", " @budi_baru ")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "budi", "Budi Baru", "@budi_baru", "budi@email.com")
	if n := e.QueryInt(t, `SELECT count(*) FROM attendees WHERE full_name = 'Budi Baru' AND account_name = '@budi_baru' AND email = 'budi@email.com'`); n != 1 {
		t.Error("rename tidak tersimpan, atau email ikut berubah")
	}

	bad := []struct {
		args []string
		msg  string
	}{
		{[]string{a.Code, "0", "--email", "SARI@email.com"}, "sudah dipakai peserta aktif lain"},
		{[]string{a.Code, "0", "--email", "bukan-email"}, "Format email belum benar"},
		{[]string{a.Code, "0", "--name", ""}, "Nama lengkap wajib diisi."},
		{[]string{a.Code, "2", "--name", "X"}, "tidak punya peserta di posisi 2"},
		{[]string{a.Code, "4", "--name", "X"}, "posisi harus 0"},
		{[]string{a.Code, "nol", "--name", "X"}, "posisi harus angka"},
		{[]string{a.Code, "0", "--name", "Budi Baru"}, "tidak ada yang berubah"},
		{[]string{"NVM-DD-ZZZZZZ", "0", "--name", "X"}, "tidak ditemukan"},
	}
	for _, c := range bad {
		_, err := runCLI(t, append([]string{"rename"}, c.args...)...)
		wantErr(t, err, c.msg)
	}
	if _, err := runCLI(t, "rename", a.Code, "0", "--usia", "20"); !errors.Is(err, errUsage) {
		t.Errorf("flag tidak dikenal: %v", err)
	}
}

func TestCLI_VIP(t *testing.T) {
	e := setupCLI(t)
	out, err := runCLI(t, "vip", "add", "Satu", "@dua", "@", "SATU")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "ditambah: Satu, @dua", `ditolak: "@", Nama akun wajib diisi.`)

	reg := e.register(t, registration.AttendeeRequest{FullName: "Dua", Email: "dua@email.com", AccountName: "dua"})
	out, err = runCLI(t, "vip", "list")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "@dua", reg.Code, "Satu", "2 nama, 2 aktif.")

	out, err = runCLI(t, "vip", "off", " @SATU ")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "Satu sekarang nonaktif.", "registrasinya tetap VIP")
	out, _ = runCLI(t, "vip", "list")
	mustContain(t, out, "2 nama, 1 aktif.")

	out, err = runCLI(t, "vip", "on", "satu")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "Satu sekarang aktif.")

	out, err = runCLI(t, "vip", "add", "satu")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "sudah ada: Satu")

	_, err = runCLI(t, "vip", "off", "tidak-ada")
	wantErr(t, err, "tidak ditemukan")
}
