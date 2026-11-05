package testdb

import (
	"strings"
	"testing"
)

func TestRequireTestDatabase(t *testing.T) {
	ok := []string{
		"postgres://doomsday_app:rahasia@127.0.0.1:15432/nvm_doomsday_test?sslmode=disable",
		"postgres://u@localhost/x_test",
	}
	for _, u := range ok {
		if err := RequireTestDatabase(u); err != nil {
			t.Errorf("%s ditolak: %v", u, err)
		}
	}

	bad := []string{
		"",
		"postgres://doomsday_app:rahasia@127.0.0.1:15432/nvm_doomsday?sslmode=disable",
		"postgres://doomsday_app:rahasia@127.0.0.1:15432/nvm_doomsday_test_lama",
		"postgres://doomsday_app:rahasia@127.0.0.1:15432/?sslmode=disable",
		// Nama _test di query string atau nama user tidak dihitung.
		"postgres://u_test:rahasia@127.0.0.1/nvm_doomsday?application_name=x_test",
		"bukan url",
	}
	for _, u := range bad {
		err := RequireTestDatabase(u)
		if err == nil {
			t.Errorf("%q diterima", u)
			continue
		}
		if strings.Contains(err.Error(), "rahasia") {
			t.Errorf("pesan error membocorkan sandi: %v", err)
		}
	}
}

func TestRedact(t *testing.T) {
	u := "postgres://doomsday_migrator:s3cr%40t@127.0.0.1:15432/nvm_doomsday_test"
	got := redact("gagal: postgres://doomsday_migrator:s3cr%40t@127.0.0.1 dan s3cr@t", u)
	if strings.Contains(got, "s3cr") {
		t.Errorf("sandi masih terlihat: %s", got)
	}
}
