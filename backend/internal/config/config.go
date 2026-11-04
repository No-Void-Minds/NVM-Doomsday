package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Env               string
	Addr              string
	DatabaseURL       string
	TrustedProxies    []string
	AdminPasswordHash []byte
	SessionSecret     []byte
	CookieSecure      bool
}

func Load() (Config, error) {
	cfg := Config{
		Env:            getenv("APP_ENV", "development"),
		Addr:           getenv("HTTP_ADDR", "127.0.0.1:8081"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV harus development atau production, didapat %q", cfg.Env)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL wajib diisi")
	}

	cfg.AdminPasswordHash = []byte(os.Getenv("ADMIN_PASSWORD_HASH"))
	if _, err := bcrypt.Cost(cfg.AdminPasswordHash); err != nil {
		// Penyebab paling umum: hash ditulis tanpa kutip tunggal di .env, sehingga shell mengartikan $ di dalamnya.
		return Config{}, errors.New("ADMIN_PASSWORD_HASH bukan hash bcrypt yang valid, pastikan diapit kutip tunggal di .env")
	}

	cfg.SessionSecret = []byte(os.Getenv("SESSION_SECRET"))
	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("SESSION_SECRET minimal 32 karakter, buat dengan: openssl rand -hex 32")
	}

	secure, err := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE harus true atau false: %w", err)
	}
	cfg.CookieSecure = secure

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
