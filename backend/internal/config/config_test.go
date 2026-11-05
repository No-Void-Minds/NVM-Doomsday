package config

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Cost minimum supaya test cepat. Load hanya memeriksa bentuk hash, bukan cost-nya.
func validHash(t *testing.T) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte("sandi-uji-config"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

// Semua variabel diisi eksplisit, supaya isi env mesin yang menjalankan test tidak ikut terbaca.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/x_test")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("ADMIN_PASSWORD_HASH", validHash(t))
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 32))
	t.Setenv("COOKIE_SECURE", "")
}

func TestLoad_Valid(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "development" || cfg.Addr != "127.0.0.1:8081" || cfg.CookieSecure {
		t.Errorf("nilai bawaan salah: %+v", cfg)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("TRUSTED_PROXIES kosong terbaca %v", cfg.TrustedProxies)
	}
}

func TestLoad_AppEnv(t *testing.T) {
	for _, env := range []string{"development", "production"} {
		setValidEnv(t)
		t.Setenv("APP_ENV", env)
		if cfg, err := Load(); err != nil || cfg.Env != env {
			t.Errorf("APP_ENV=%s: %v", env, err)
		}
	}
	for _, env := range []string{"prod", "Production", "dev", "test"} {
		setValidEnv(t)
		t.Setenv("APP_ENV", env)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "APP_ENV") {
			t.Errorf("APP_ENV=%s diterima: %v", env, err)
		}
	}
	// Kosong berarti development, sama dengan tidak diisi.
	setValidEnv(t)
	t.Setenv("APP_ENV", "")
	if cfg, err := Load(); err != nil || cfg.Env != "development" {
		t.Errorf("APP_ENV kosong: %v %q", err, cfg.Env)
	}
}

func TestLoad_DatabaseURLWajib(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("DATABASE_URL kosong diterima: %v", err)
	}
}

func TestLoad_HashBcrypt(t *testing.T) {
	hash := validHash(t)
	bad := map[string]string{
		"kosong":            "",
		"placeholder":       "$2a$12$GANTI_DENGAN_HASH",
		"teks biasa":        "sandi-admin",
		"terpotong":         hash[:30],
		"dimakan shell":     strings.SplitN(hash, "$", 4)[3],
		"diapit kutip ikut": "'" + hash + "'",
	}
	for name, v := range bad {
		setValidEnv(t)
		t.Setenv("ADMIN_PASSWORD_HASH", v)
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "ADMIN_PASSWORD_HASH") {
			t.Errorf("%s: hash %q diterima: %v", name, v, err)
		}
	}
}

func TestLoad_PanjangSessionSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 31))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Errorf("secret 31 karakter diterima: %v", err)
	}
	t.Setenv("SESSION_SECRET", "")
	if _, err := Load(); err == nil {
		t.Error("secret kosong diterima")
	}
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 32))
	if cfg, err := Load(); err != nil || len(cfg.SessionSecret) != 32 {
		t.Errorf("secret 32 karakter ditolak: %v", err)
	}
}

func TestLoad_CookieSecure(t *testing.T) {
	setValidEnv(t)
	t.Setenv("COOKIE_SECURE", "true")
	if cfg, err := Load(); err != nil || !cfg.CookieSecure {
		t.Errorf("COOKIE_SECURE=true: %v %v", err, cfg.CookieSecure)
	}
	t.Setenv("COOKIE_SECURE", "ya")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Errorf("COOKIE_SECURE=ya diterima: %v", err)
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	setValidEnv(t)
	t.Setenv("TRUSTED_PROXIES", " 172.30.93.0/24, ,10.0.0.1 ,")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(cfg.TrustedProxies, "|") != "172.30.93.0/24|10.0.0.1" {
		t.Errorf("TRUSTED_PROXIES = %q", cfg.TrustedProxies)
	}
}
