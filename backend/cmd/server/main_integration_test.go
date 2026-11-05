//go:build integration

package main

import (
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/testdb"
)

func setServerEnv(t *testing.T, databaseURL string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("sandi-uji-server"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", "127.0.0.1:0")
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("ADMIN_PASSWORD_HASH", string(hash))
	t.Setenv("SESSION_SECRET", strings.Repeat("s", 32))
	t.Setenv("COOKIE_SECURE", "false")
}

// Checklist Sprint 1: gagal di awal saat database tidak terjangkau, tanpa sempat mencatat "server jalan".
func TestRun_GagalSaatDatabaseTidakTerjangkau(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	setServerEnv(t, "postgres://u:p@"+addr+"/x_test?connect_timeout=1")

	var logs strings.Builder
	err = run(slog.New(slog.NewTextHandler(&logs, nil)))
	if err == nil || !strings.Contains(err.Error(), "ping database") {
		t.Errorf("error = %v, mau ping database", err)
	}
	if strings.Contains(logs.String(), "server jalan") {
		t.Error("server sempat jalan")
	}
}

// Sprint 9: TRUSTED_PROXIES salah ketik menghentikan server, bukan menjalankan router nil.
func TestRun_TrustedProxiesTidakValid(t *testing.T) {
	env := testdb.Setup(t)
	setServerEnv(t, env.AppURL)
	t.Setenv("TRUSTED_PROXIES", "172.30.93.0/33")
	err := run(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "invalid CIDR address") {
		t.Errorf("error = %v, mau invalid CIDR address", err)
	}
}
