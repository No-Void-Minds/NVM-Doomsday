package vip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
)

// Satu kali tempel dari Excel. Lebih dari ini kemungkinan besar seluruh sheet ikut tersalin.
const MaxBatch = 500

var (
	ErrNotFound = errors.New("nama akun VIP tidak ditemukan")
	ErrTooMany  = fmt.Errorf("maksimal %d nama sekali tambah", MaxBatch)
)

type Account struct {
	ID     uuid.UUID
	Name   string
	Active bool
	// Kosong kalau nama ini belum dipakai registrasi yang belum batal.
	RegistrationCode string
	CreatedAt        time.Time
}

type Rejected struct {
	Name    string
	Message string
}

type AddReport struct {
	Added       []string
	Reactivated []string
	Existing    []string
	Rejected    []Rejected
}

type Service struct {
	q *sqlc.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{q: sqlc.New(pool)}
}

func (s *Service) List(ctx context.Context) ([]Account, error) {
	rows, err := s.q.ListVipAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("ambil daftar VIP: %w", err)
	}
	out := make([]Account, len(rows))
	for i, r := range rows {
		out[i] = Account{
			ID:               r.ID,
			Name:             r.AccountName,
			Active:           r.IsActive,
			RegistrationCode: r.RegistrationCode,
			CreatedAt:        r.CreatedAt,
		}
	}
	return out, nil
}

// Add memproses nama satu per satu tanpa satu transaksi besar, supaya satu baris salah dari Excel
// tidak menggagalkan nama lain. Laporannya memberi tahu admin baris mana yang perlu diperbaiki.
func (s *Service) Add(ctx context.Context, names []string) (AddReport, error) {
	rep := AddReport{Added: []string{}, Reactivated: []string{}, Existing: []string{}, Rejected: []Rejected{}}
	if len(names) > MaxBatch {
		return rep, ErrTooMany
	}

	seen := make(map[string]struct{}, len(names))
	for _, raw := range names {
		// Baris kosong wajar ada di tempelan Excel.
		if strings.TrimSpace(raw) == "" {
			continue
		}
		name, errs := registration.ValidateAccount(raw)
		if len(errs) > 0 {
			rep.Rejected = append(rep.Rejected, Rejected{Name: strings.TrimSpace(raw), Message: errs[0].Message})
			continue
		}
		key := registration.NormalizeAccount(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		cur, err := s.q.GetVipAccountByKey(ctx, key)
		switch {
		case err == nil && cur.IsActive:
			rep.Existing = append(rep.Existing, cur.AccountName)
		case err == nil:
			if _, err := s.q.SetVipAccountActive(ctx, sqlc.SetVipAccountActiveParams{ID: cur.ID, IsActive: true}); err != nil {
				return rep, fmt.Errorf("aktifkan %q: %w", cur.AccountName, err)
			}
			rep.Reactivated = append(rep.Reactivated, cur.AccountName)
		case errors.Is(err, pgx.ErrNoRows):
			_, err := s.q.InsertVipAccount(ctx, sqlc.InsertVipAccountParams{AccountName: name, AccountKey: key})
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				// Masuk lebih dulu lewat jalur lain di saat yang sama.
				rep.Existing = append(rep.Existing, name)
			case err != nil:
				return rep, fmt.Errorf("tambah %q: %w", name, err)
			default:
				rep.Added = append(rep.Added, name)
			}
		default:
			return rep, fmt.Errorf("cek %q: %w", name, err)
		}
	}
	return rep, nil
}

// Menonaktifkan nama yang sudah dipakai tidak mengubah registrasinya: tetap VIP sampai admin membatalkannya.
func (s *Service) SetActive(ctx context.Context, id uuid.UUID, active bool) (Account, error) {
	row, err := s.q.SetVipAccountActive(ctx, sqlc.SetVipAccountActiveParams{ID: id, IsActive: active})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("ubah status VIP: %w", err)
	}
	return Account{ID: row.ID, Name: row.AccountName, Active: row.IsActive, CreatedAt: row.CreatedAt}, nil
}

func (s *Service) SetActiveByName(ctx context.Context, name string, active bool) (Account, error) {
	cur, err := s.q.GetVipAccountByKey(ctx, registration.NormalizeAccount(name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("cari nama VIP: %w", err)
	}
	return s.SetActive(ctx, cur.ID, active)
}
