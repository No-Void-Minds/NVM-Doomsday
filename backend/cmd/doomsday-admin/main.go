package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/vip"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errUsage = errors.New("pemakaian salah")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:])
	stop()

	if errors.Is(err, errUsage) {
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "hash-password":
		return hashPassword(os.Stdin, os.Stdout)
	case "event":
		return withPool(ctx, func(pool *pgxpool.Pool) error { return eventCmd(ctx, pool, args[1:]) })
	case "promote", "cancel", "rename":
		return withPool(ctx, func(pool *pgxpool.Pool) error {
			return registrationCmd(ctx, registration.NewService(pool), args)
		})
	case "vip":
		return withPool(ctx, func(pool *pgxpool.Pool) error { return vipCmd(ctx, vip.NewService(pool), args[1:]) })
	default:
		return errUsage
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Pemakaian: doomsday-admin <perintah>

Perintah:
  hash-password             baca sandi dari stdin, cetak hash bcrypt untuk ADMIN_PASSWORD_HASH
  event show                tampilkan konfigurasi acara dan jumlah peserta
  event set <field> <nilai> ubah satu field, nilai "-" mengosongkan field yang boleh kosong
  promote <kode>            pindahkan registrasi daftar tunggu ke terkonfirmasi, kalau kursi cukup
  cancel <kode>             batalkan registrasi, kursinya dibebaskan
  rename <kode> <posisi> [--name ..] [--email ..] [--account ..]
                            ubah data satu peserta, posisi 0 pendaftar, 1 sampai 3 tamu
  vip list                  tampilkan daftar nama akun VIP dan registrasi yang memakainya
  vip add <nama>...         tambah satu atau beberapa nama akun VIP
  vip off <nama>            nonaktifkan nama, registrasi yang sudah memakainya tetap VIP
  vip on <nama>             aktifkan lagi nama yang pernah dinonaktifkan

Field event: title, film_title, cinema_name, city, studio, gather_at, show_at,
             map_url, capacity, max_guests_vip, is_open
Format jam: "YYYY-MM-DD HH:MM" dalam WIB, contoh: event set show_at "2026-12-18 19:00"`)
}

// CLI login sebagai doomsday_app, sama seperti backend, jadi tidak bisa menghapus data atau mengubah tabel.
func withPool(ctx context.Context, fn func(*pgxpool.Pool) error) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL belum diisi, jalankan dulu: set -a; . ./.env; set +a")
	}
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	return fn(pool)
}
