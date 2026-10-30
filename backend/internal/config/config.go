package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env  string
	Addr string
}

func Load() (Config, error) {
	cfg := Config{
		Env:  getenv("APP_ENV", "development"),
		Addr: getenv("HTTP_ADDR", "127.0.0.1:8081"),
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV harus development atau production, didapat %q", cfg.Env)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}