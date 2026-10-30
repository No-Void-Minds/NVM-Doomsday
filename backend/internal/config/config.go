package config

import (
	"errors"
	"fmt"
	"os"
)

type Config struct {
	Env         string
	Addr        string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		Env:         getenv("APP_ENV", "development"),
		Addr:        getenv("HTTP_ADDR", "127.0.0.1:8081"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV harus development atau production, didapat %q", cfg.Env)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL wajib diisi")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
