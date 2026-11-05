package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// Sandi dibaca dari stdin, bukan argumen, supaya tidak tersimpan di riwayat shell atau daftar proses.
func hashPassword(in io.Reader, out io.Writer) error {
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("baca sandi: %w", err)
	}
	pw := strings.TrimRight(line, "\r\n")

	if utf8.RuneCountInString(pw) < 10 {
		return errors.New("sandi minimal 10 karakter")
	}
	// bcrypt hanya memakai 72 byte pertama. Sandi lebih panjang ditolak supaya tidak ada ilusi keamanan.
	if len(pw) > 72 {
		return errors.New("sandi maksimal 72 byte")
	}

	// Cost 12 sekitar 0,25 detik per cek di laptop biasa: lambat untuk penebak, tidak terasa bagi admin.
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		return fmt.Errorf("buat hash: %w", err)
	}
	fmt.Fprintln(out, string(hash))
	return nil
}
