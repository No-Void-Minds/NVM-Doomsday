package registration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Pola dibaca dari migrasi, supaya alfabet di Go dan CHECK registrations.code tidak bisa diam-diam berbeda.
func codePatternFromMigration(t *testing.T) *regexp.Regexp {
	t.Helper()
	sql, err := os.ReadFile(filepath.Join("..", "..", "migrations", "0001_init.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`code ~ '(\^NVM-DD-[^']+)'`).FindSubmatch(sql)
	if m == nil {
		t.Fatal("CHECK kode registrasi tidak ditemukan di migrasi 0001")
	}
	return regexp.MustCompile(string(m[1]))
}

func TestNewCode_SesuaiCheckDatabase(t *testing.T) {
	pattern := codePatternFromMigration(t)
	seen := make(map[string]struct{})
	chars := make(map[rune]int)
	for range 1000 {
		code, err := NewCode()
		if err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(code) {
			t.Fatalf("%s tidak cocok dengan CHECK %s", code, pattern)
		}
		if strings.ContainsAny(code[len(codePrefix):], "OIL01") {
			t.Fatalf("%s berisi huruf yang mudah tertukar", code)
		}
		seen[code] = struct{}{}
		for _, r := range code[len(codePrefix):] {
			chars[r]++
		}
	}
	// 31^6 kemungkinan. Dari 1000 kode, lebih dari 5 kembar berarti acakannya rusak.
	if len(seen) < 995 {
		t.Errorf("hanya %d kode berbeda dari 1000", len(seen))
	}
	// 6000 karakter, rata-rata 190 per huruf. Huruf yang tidak pernah muncul berarti alfabet terpotong.
	if len(chars) != len(codeAlphabet) {
		t.Errorf("%d huruf muncul, alfabet berisi %d", len(chars), len(codeAlphabet))
	}
}

func TestCodeAlphabet_SamaDenganCheckDatabase(t *testing.T) {
	pattern := codePatternFromMigration(t)
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
		code := codePrefix + strings.Repeat(string(r), codeLength)
		inGo := strings.ContainsRune(codeAlphabet, r)
		if inDB := pattern.MatchString(code); inGo != inDB {
			t.Errorf("%c: di alfabet Go %v, diterima CHECK %v", r, inGo, inDB)
		}
	}
}

func TestNormalizeCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"NVM-DD-ABC234", "NVM-DD-ABC234", true},
		{" nvm-dd-abc234 ", "NVM-DD-ABC234", true},
		{"\tNvm-Dd-XyZ789\n", "NVM-DD-XYZ789", true},
		{"NVM-DD-ABC23", "", false},
		{"NVM-DD-ABC2345", "", false},
		{"NVM-DD-ABCO23", "", false},
		{"NVM-DD-ABC0I1", "", false},
		{"nvm-dd-abcl23", "", false},
		{"XYZ-DD-ABC234", "", false},
		{"NVMDDABC234", "", false},
		{"NVM-DD-ABC 23", "", false},
		{"NVM-DD-ÄBC23", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeCode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeCode(%q) = %q, %v; mau %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}

	code, err := NewCode()
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := NormalizeCode(strings.ToLower(code)); !ok || got != code {
		t.Errorf("kode baru %s tidak kembali utuh: %q, %v", code, got, ok)
	}
}
