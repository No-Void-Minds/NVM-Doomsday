package adminauth

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	secret = []byte("rahasia-uji-yang-panjangnya-32-byte!")
	now    = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
)

func TestValid_TokenBaru(t *testing.T) {
	token := Sign(secret, now.Add(TTL))
	if !Valid(secret, token, now) {
		t.Fatal("token baru ditolak")
	}
	if !Valid(secret, token, now.Add(TTL-time.Second)) {
		t.Error("token ditolak sedetik sebelum kedaluwarsa")
	}
}

func TestValid_Kedaluwarsa(t *testing.T) {
	token := Sign(secret, now.Add(TTL))
	// Tepat di detik kedaluwarsa sudah ditolak, supaya sesi tidak lebih panjang dari 8 jam.
	if Valid(secret, token, now.Add(TTL)) {
		t.Error("token diterima tepat saat kedaluwarsa")
	}
	if Valid(secret, token, now.Add(TTL+time.Hour)) {
		t.Error("token kedaluwarsa diterima")
	}
}

func TestValid_TandaTanganSalah(t *testing.T) {
	token := Sign(secret, now.Add(TTL))
	if Valid([]byte("rahasia-lain-yang-panjangnya-32-byte"), token, now) {
		t.Error("token dari secret lain diterima, padahal mengganti SESSION_SECRET harus mengeluarkan semua admin")
	}

	exp, sig, _ := strings.Cut(token, ".")
	// Memperpanjang kedaluwarsa tanpa menandatangani ulang harus ditolak.
	later := strconv.FormatInt(now.Add(30*24*time.Hour).Unix(), 10)
	if Valid(secret, later+"."+sig, now) {
		t.Error("kedaluwarsa yang diubah tetap diterima")
	}

	flipped := []byte(sig)
	if flipped[0] == 'A' {
		flipped[0] = 'B'
	} else {
		flipped[0] = 'A'
	}
	if Valid(secret, exp+"."+string(flipped), now) {
		t.Error("tanda tangan yang diubah satu karakter diterima")
	}
}

func TestValid_FormatRusak(t *testing.T) {
	good := Sign(secret, now.Add(TTL))
	exp, sig, _ := strings.Cut(good, ".")
	cases := map[string]string{
		"kosong":             "",
		"tanpa titik":        exp + sig,
		"hanya titik":        ".",
		"tanpa tanda tangan": exp + ".",
		"tanpa kedaluwarsa":  "." + sig,
		"titik tambahan":     good + ".x",
		"spasi":              " " + good,
	}
	// base64url peka huruf besar kecil. Hanya diuji kalau tanda tangannya memang berisi huruf besar.
	if lower := strings.ToLower(sig); lower != sig {
		cases["tanda tangan huruf kecil"] = exp + "." + lower
	}
	for name, token := range cases {
		if Valid(secret, token, now) {
			t.Errorf("%s: token %q diterima", name, token)
		}
	}
}

// Tanda tangan yang sah untuk teks bukan angka tetap ditolak, supaya kedaluwarsa tidak bisa dilewati.
func TestValid_KedaluwarsaBukanAngka(t *testing.T) {
	for _, exp := range []string{"abc", "1e10", "99999999999999999999", ""} {
		if Valid(secret, exp+"."+mac(secret, exp), now) {
			t.Errorf("kedaluwarsa %q diterima", exp)
		}
	}
}

func TestSign_Bentuk(t *testing.T) {
	expires := now.Add(TTL)
	exp, sig, ok := strings.Cut(Sign(secret, expires), ".")
	if !ok || exp != strconv.FormatInt(expires.Unix(), 10) {
		t.Fatalf("bentuk token salah: %q", exp)
	}
	// Dipakai di cookie, jadi tidak boleh berisi karakter yang perlu di-escape.
	if strings.ContainsAny(sig, "+/=;, ") {
		t.Errorf("tanda tangan %q berisi karakter yang tidak aman untuk cookie", sig)
	}
}
