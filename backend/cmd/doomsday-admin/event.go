package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/timezone"
)

const (
	emptyValue = "(kosong)"
	timeLayout = "2006-01-02 15:04"
)

func eventCmd(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	q := sqlc.New(pool)
	switch {
	case len(args) == 1 && args[0] == "show":
		return eventShow(ctx, q)
	case len(args) == 3 && args[0] == "set":
		return eventSet(ctx, q, args[1], args[2])
	default:
		return errUsage
	}
}

func eventShow(ctx context.Context, q *sqlc.Queries) error {
	ev, err := q.GetEvent(ctx)
	if err != nil {
		return fmt.Errorf("ambil event: %w", err)
	}
	sum, err := q.CountSummary(ctx)
	if err != nil {
		return fmt.Errorf("hitung peserta: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, f := range eventFields(ev) {
		fmt.Fprintf(w, "%s\t%s\n", f.name, f.value)
	}
	fmt.Fprintln(w, "\t")
	fmt.Fprintf(w, "terkonfirmasi\t%d dari %d peserta\n", sum.ConfirmedAttendees, ev.Capacity)
	fmt.Fprintf(w, "daftar tunggu\t%d peserta\n", sum.WaitlistedAttendees)
	fmt.Fprintf(w, "batal\t%d registrasi\n", sum.CancelledRegistrations)
	return w.Flush()
}

func eventSet(ctx context.Context, q *sqlc.Queries, name, value string) error {
	ev, err := q.GetEvent(ctx)
	if err != nil {
		return fmt.Errorf("ambil event: %w", err)
	}

	p := sqlc.UpdateEventParams{
		Title:        ev.Title,
		FilmTitle:    ev.FilmTitle,
		CinemaName:   ev.CinemaName,
		City:         ev.City,
		Studio:       ev.Studio,
		GatherAt:     ev.GatherAt,
		ShowAt:       ev.ShowAt,
		MapUrl:       ev.MapUrl,
		Capacity:     ev.Capacity,
		MaxGuestsVip: ev.MaxGuestsVip,
		IsOpen:       ev.IsOpen,
	}

	value = strings.TrimSpace(value)
	switch name {
	case "title":
		p.Title, err = requiredText(value)
	case "film_title":
		p.FilmTitle, err = requiredText(value)
	case "cinema_name":
		p.CinemaName = optionalText(value)
	case "city":
		p.City = optionalText(value)
	case "studio":
		p.Studio = optionalText(value)
	case "gather_at":
		p.GatherAt, err = optionalTime(value)
	case "show_at":
		p.ShowAt, err = optionalTime(value)
	case "map_url":
		p.MapUrl, err = optionalHTTPS(value)
	case "capacity":
		var n int
		n, err = intInRange(value, 1, 10000)
		p.Capacity = int32(n)
	case "max_guests_vip":
		var n int
		n, err = intInRange(value, 1, 3)
		p.MaxGuestsVip = int16(n)
	case "is_open":
		p.IsOpen, err = strconv.ParseBool(value)
	default:
		return fmt.Errorf("field %q tidak dikenal, lihat daftar field dengan: doomsday-admin", name)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	updated, err := q.UpdateEvent(ctx, p)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" {
		return errors.New("nilai ditolak aturan tabel, misalnya jam kumpul setelah jam tayang")
	}
	if err != nil {
		return fmt.Errorf("simpan event: %w", err)
	}

	fmt.Printf("%s: %s  ->  %s\n", name, fieldValue(ev, name), fieldValue(updated, name))

	if name == "capacity" {
		sum, err := q.CountSummary(ctx)
		if err != nil {
			return fmt.Errorf("hitung peserta: %w", err)
		}
		if sum.ConfirmedAttendees > int64(updated.Capacity) {
			// Tidak ada yang otomatis dipindah ke daftar tunggu: keputusan itu milik admin, bukan sistem.
			fmt.Printf("Perhatian: sudah ada %d peserta terkonfirmasi, melebihi kapasitas baru. Tidak ada yang dipindah otomatis.\n",
				sum.ConfirmedAttendees)
		}
	}
	return nil
}

type field struct{ name, value string }

func eventFields(ev sqlc.EventConfig) []field {
	return []field{
		{"title", ev.Title},
		{"film_title", ev.FilmTitle},
		{"cinema_name", showText(ev.CinemaName)},
		{"city", showText(ev.City)},
		{"studio", showText(ev.Studio)},
		{"gather_at", showTime(ev.GatherAt)},
		{"show_at", showTime(ev.ShowAt)},
		{"map_url", showText(ev.MapUrl)},
		{"capacity", strconv.Itoa(int(ev.Capacity))},
		{"max_guests_vip", strconv.Itoa(int(ev.MaxGuestsVip))},
		{"is_open", strconv.FormatBool(ev.IsOpen)},
	}
}

func fieldValue(ev sqlc.EventConfig, name string) string {
	for _, f := range eventFields(ev) {
		if f.name == name {
			return f.value
		}
	}
	return ""
}

func showText(s *string) string {
	if s == nil {
		return emptyValue
	}
	return *s
}

func showTime(t *time.Time) string {
	if t == nil {
		return emptyValue
	}
	return t.In(timezone.WIB).Format(timeLayout) + " WIB"
}

func requiredText(v string) (string, error) {
	if v == "" || v == "-" {
		return "", errors.New("tidak boleh kosong")
	}
	return v, nil
}

func optionalText(v string) *string {
	if v == "" || v == "-" {
		return nil
	}
	return &v
}

func optionalTime(v string) (*time.Time, error) {
	if v == "" || v == "-" {
		return nil, nil
	}
	t, err := time.ParseInLocation(timeLayout, v, timezone.WIB)
	if err != nil {
		return nil, errors.New(`format harus "YYYY-MM-DD HH:MM", contoh "2026-12-18 19:00"`)
	}
	return &t, nil
}

func optionalHTTPS(v string) (*string, error) {
	if v == "" || v == "-" {
		return nil, nil
	}
	u, err := url.Parse(v)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.New("harus berupa link yang diawali https://")
	}
	return &v, nil
}

func intInRange(v string, min, max int) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("harus angka %d sampai %d", min, max)
	}
	return n, nil
}
