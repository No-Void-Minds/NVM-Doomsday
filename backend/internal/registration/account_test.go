package registration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// File yang sama dibaca Vitest untuk normalizeAccount, supaya cocok tidaknya nama VIP sama di kedua sisi.
func TestNormalizeAccount_KasusBersama(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "normalisasi-nama-akun.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Kasus []struct {
			Input string `json:"input"`
			Hasil string `json:"hasil"`
		} `json:"kasus"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Kasus) == 0 {
		t.Fatal("file kasus kosong")
	}
	for _, c := range file.Kasus {
		if got := NormalizeAccount(c.Input); got != c.Hasil {
			t.Errorf("NormalizeAccount(%+q) = %+q, mau %+q", c.Input, got, c.Hasil)
		}
	}
}

// Setiap nama yang lolos ValidateAccount disimpan sebagai account_key. CHECK di migrasi 0004 menolak kunci
// berawalan @ atau berspasi, jadi kunci yang melanggarnya membuat tambah nama VIP gagal di database.
func TestNormalizeAccount_KunciMemenuhiCheckDatabase(t *testing.T) {
	inputs := []string{"budi", " @Budi ", "@@budi", "@ budi", "@ @budi", "@\t@budi", "@ @ budi", "budi@toko"}
	for _, in := range inputs {
		name, errs := ValidateAccount(in)
		if len(errs) > 0 {
			continue
		}
		key := NormalizeAccount(name)
		switch {
		case key == "":
			t.Errorf("%q: kunci kosong padahal lolos validasi", in)
		case strings.HasPrefix(key, "@"):
			t.Errorf("%q: kunci %q diawali @, ditolak CHECK vip_accounts.account_key", in, key)
		case key != strings.Trim(key, " "):
			t.Errorf("%q: kunci %q berspasi di tepi", in, key)
		case key != strings.ToLower(key):
			t.Errorf("%q: kunci %q belum huruf kecil", in, key)
		}
		// Tidak idempoten berarti nama yang sama bisa menghasilkan kunci berbeda saat dicek ulang.
		if again := NormalizeAccount(key); again != key {
			t.Errorf("%q: tidak idempoten, %q lalu %q", in, key, again)
		}
	}
}

func TestValidateAccount(t *testing.T) {
	cases := []struct {
		in, wantName, wantMsg string
	}{
		{" budi ", "budi", ""},
		{" @Budi ", "@Budi", ""},
		{"", "", "Nama akun wajib diisi."},
		{"   ", "", "Nama akun wajib diisi."},
		{"@", "@", "Nama akun wajib diisi."},
		{"@@", "@@", "Nama akun wajib diisi."},
		{strings.Repeat("a", 50), strings.Repeat("a", 50), ""},
		{strings.Repeat("a", 51), strings.Repeat("a", 51), "Nama akun maksimal 50 karakter."},
		{"bu\u0007di", "bu\u0007di", "Nama akun berisi karakter yang tidak valid."},
	}
	for _, c := range cases {
		name, errs := ValidateAccount(c.in)
		if c.wantMsg == "" {
			if len(errs) > 0 {
				t.Errorf("%q: error tidak diharapkan %v", c.in, errs)
			}
			if name != c.wantName {
				t.Errorf("%q: nama = %q, mau %q", c.in, name, c.wantName)
			}
			continue
		}
		if len(errs) != 1 || errs[0].Field != "account_name" || errs[0].Message != c.wantMsg {
			t.Errorf("%q: errs = %+v, mau %q", c.in, errs, c.wantMsg)
		}
	}
}
