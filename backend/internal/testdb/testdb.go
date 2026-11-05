// Package testdb menyiapkan database uji untuk integration test (build tag integration).
// Setiap fungsi menolak jalan kalau nama database di URL tidak berakhiran _test, karena isinya
// dikosongkan di antara test dan nvm_doomsday adalah data asli peserta.
package testdb

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db"
)

const (
	AppURLEnv      = "TEST_DATABASE_URL"
	MigratorURLEnv = "TEST_MIGRATE_DATABASE_URL"
	// Angka bebas, hanya perlu sama di semua proses test yang memakai database uji.
	lockKey = 20261110
)

// RequireTestDatabase hanya membaca nama database, tidak pernah mencetak URL yang berisi sandi.
func RequireTestDatabase(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL database uji kosong")
	}
	cfg, err := pgx.ParseConfig(rawURL)
	if err != nil {
		return errors.New("URL database uji tidak bisa dibaca")
	}
	if !strings.HasSuffix(cfg.Database, "_test") {
		return fmt.Errorf("database %q tidak berakhiran _test, test menolak jalan supaya data asli tidak terhapus", cfg.Database)
	}
	return nil
}

func backendDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// Nilai dari env didahulukan. Kalau kosong, dibaca dari backend/.env supaya go test bisa langsung dijalankan
// tanpa source manual. Hanya dua kunci ini yang diambil.
func lookup(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	f, err := os.Open(filepath.Join(backendDir(), ".env"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if ok && k == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

type Env struct {
	AppURL      string
	MigratorURL string
}

var (
	setupOnce sync.Once
	setupEnv  *Env
	setupErr  error
	// Ditahan sampai proses test selesai. Paket lain yang ikut go test ./... menunggu di kunci ini,
	// supaya dua paket tidak mengosongkan database yang sama bersamaan.
	lockConn *pgx.Conn
)

// Setup dipanggil di awal setiap integration test. Migrasi dan kunci hanya dijalankan sekali per proses.
func Setup(t testing.TB) *Env {
	t.Helper()
	setupOnce.Do(func() { setupEnv, setupErr = setup() })
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	return setupEnv
}

func setup() (*Env, error) {
	env := &Env{AppURL: lookup(AppURLEnv), MigratorURL: lookup(MigratorURLEnv)}
	if env.AppURL == "" || env.MigratorURL == "" {
		return nil, fmt.Errorf("%s dan %s belum diisi di env atau backend/.env", AppURLEnv, MigratorURLEnv)
	}
	if err := RequireTestDatabase(env.AppURL); err != nil {
		return nil, fmt.Errorf("%s: %w", AppURLEnv, err)
	}
	if err := RequireTestDatabase(env.MigratorURL); err != nil {
		return nil, fmt.Errorf("%s: %w", MigratorURLEnv, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, env.MigratorURL)
	if err != nil {
		return nil, fmt.Errorf("konek ke database uji gagal, cek tunnel 127.0.0.1:15432: %s", redact(err.Error(), env.MigratorURL))
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return nil, fmt.Errorf("kunci database uji: %w", err)
	}
	lockConn = conn

	if err := migrateUp(env.MigratorURL); err != nil {
		return nil, err
	}
	return env, nil
}

// Memakai CLI migrate yang sama dengan deploy, supaya schema_migrations terbentuk persis seperti di produksi.
func migrateUp(rawURL string) error {
	bin, err := exec.LookPath("migrate")
	if err != nil {
		return errors.New("perintah migrate tidak ditemukan di PATH, pasang golang-migrate v4 dengan tag postgres")
	}
	cmd := exec.Command(bin, "-path", "migrations", "-database", rawURL, "up")
	cmd.Dir = backendDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("migrate up gagal: %s", redact(string(out), rawURL))
	}
	return nil
}

// Pesan error dari driver bisa memuat URL. Sandinya diganti sebelum dicetak.
func redact(s, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return s
	}
	if pw, ok := u.User.Password(); ok && pw != "" {
		s = strings.ReplaceAll(s, pw, "***")
		s = strings.ReplaceAll(s, url.QueryEscape(pw), "***")
	}
	return s
}

// Reset mengosongkan data peserta dan daftar VIP, lalu mengembalikan event_config ke isi seed migrasi 0003.
// Dijalankan sebagai doomsday_migrator karena doomsday_app tidak punya izin DELETE atau TRUNCATE.
func (e *Env) Reset(t testing.TB) {
	t.Helper()
	// Kalau koneksi kunci putus (misalnya tunnel tersambung ulang), proses test lain bisa sedang memakai
	// database yang sama. Lebih baik berhenti daripada mengosongkan data test itu.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := lockConn.Ping(ctx); err != nil {
		t.Fatalf("koneksi kunci database uji terputus: %v", err)
	}
	e.Exec(t, `TRUNCATE attendees, registrations, vip_accounts`)
	e.Exec(t, `
		INSERT INTO event_config (id, title, film_title, capacity, max_guests_vip, is_open)
		VALUES (1, 'No Void Minds Doomsday', 'Avengers: Doomsday', 200, 3, true)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title, film_title = EXCLUDED.film_title,
			cinema_name = NULL, city = NULL, studio = NULL, gather_at = NULL, show_at = NULL, map_url = NULL,
			capacity = EXCLUDED.capacity, max_guests_vip = EXCLUDED.max_guests_vip, is_open = EXCLUDED.is_open,
			updated_at = now()`)
}

// Exec menjalankan SQL sebagai doomsday_migrator, untuk menyiapkan keadaan yang tidak bisa dibuat lewat API.
func (e *Env) Exec(t testing.TB, sql string, args ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, e.MigratorURL)
	if err != nil {
		t.Fatalf("konek sebagai migrator: %s", redact(err.Error(), e.MigratorURL))
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec %q: %v", firstLine(sql), err)
	}
}

// QueryInt untuk memeriksa isi tabel langsung, tanpa lewat kode yang sedang diuji.
func (e *Env) QueryInt(t testing.TB, sql string, args ...any) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, e.MigratorURL)
	if err != nil {
		t.Fatalf("konek sebagai migrator: %s", redact(err.Error(), e.MigratorURL))
	}
	defer conn.Close(ctx)
	var n int64
	if err := conn.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", firstLine(sql), err)
	}
	return n
}

// AppPool login sebagai doomsday_app, sama seperti server dan CLI di produksi.
func (e *Env) AppPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool, err := db.NewPool(context.Background(), e.AppURL)
	if err != nil {
		t.Fatalf("pool doomsday_app: %s", redact(err.Error(), e.AppURL))
	}
	t.Cleanup(pool.Close)
	return pool
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + "..."
	}
	return s
}
