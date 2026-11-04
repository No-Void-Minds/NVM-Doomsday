package registration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

var ErrNotFound = errors.New("kode registrasi tidak ditemukan")

type Snapshot struct {
	Status     sqlc.RegistrationStatus
	Attendees  int
	Confirmed  int64
	Waitlisted int64
	Capacity   int32
}

type WaitlistHead struct {
	Code      string
	Attendees int64
}

type Change struct {
	Code   string
	Before Snapshot
	After  Snapshot
	// Terisi bila registrasi terkonfirmasi dibatalkan dan masih ada daftar tunggu.
	NextWaitlisted *WaitlistHead
}

func (s *Service) Promote(ctx context.Context, rawCode string) (Change, error) {
	return s.changeStatus(ctx, rawCode, sqlc.RegistrationStatusConfirmed,
		func(reg sqlc.Registration, snap Snapshot) error {
			if reg.Status != sqlc.RegistrationStatusWaitlisted {
				return fmt.Errorf("hanya registrasi daftar tunggu yang bisa dipromosikan, status sekarang %s", reg.Status)
			}
			left := int64(snap.Capacity) - snap.Confirmed
			if int64(snap.Attendees) > left {
				return fmt.Errorf("kursi tidak cukup: rombongan %d orang, sisa %d kursi", snap.Attendees, max(left, 0))
			}
			return nil
		})
}

func (s *Service) Cancel(ctx context.Context, rawCode string) (Change, error) {
	return s.changeStatus(ctx, rawCode, sqlc.RegistrationStatusCancelled,
		func(reg sqlc.Registration, _ Snapshot) error {
			if reg.Status == sqlc.RegistrationStatusCancelled {
				return errors.New("registrasi ini sudah batal")
			}
			return nil
		})
}

// Memakai kunci event yang sama dengan pendaftaran, supaya promosi tidak balapan dengan pendaftar baru
// yang mengambil kursi terakhir.
func (s *Service) changeStatus(ctx context.Context, rawCode string, to sqlc.RegistrationStatus,
	check func(sqlc.Registration, Snapshot) error) (Change, error) {
	code, ok := NormalizeCode(rawCode)
	if !ok {
		return Change{}, errors.New("format kode tidak valid, contoh: NVM-DD-ABC234")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Change{}, fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(s.pool).WithTx(tx)

	ev, err := q.LockEvent(ctx)
	if err != nil {
		return Change{}, fmt.Errorf("kunci event: %w", err)
	}
	reg, err := q.GetRegistrationByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return Change{}, ErrNotFound
	}
	if err != nil {
		return Change{}, fmt.Errorf("ambil registrasi: %w", err)
	}

	before, err := snapshot(ctx, q, ev.Capacity, reg)
	if err != nil {
		return Change{}, err
	}
	if err := check(reg, before); err != nil {
		return Change{}, err
	}

	// status_changed_at dan attendees.is_active diperbarui trigger migrasi 0002.
	updated, err := q.UpdateRegistrationStatus(ctx, sqlc.UpdateRegistrationStatusParams{Status: to, Code: code})
	if err != nil {
		return Change{}, fmt.Errorf("ubah status: %w", err)
	}
	after, err := snapshot(ctx, q, ev.Capacity, updated)
	if err != nil {
		return Change{}, err
	}

	ch := Change{Code: code, Before: before, After: after}
	if to == sqlc.RegistrationStatusCancelled && reg.Status == sqlc.RegistrationStatusConfirmed {
		head, err := q.FirstWaitlisted(ctx)
		switch {
		case err == nil:
			ch.NextWaitlisted = &WaitlistHead{Code: head.Code, Attendees: head.Attendees}
		case !errors.Is(err, pgx.ErrNoRows):
			return Change{}, fmt.Errorf("ambil daftar tunggu teratas: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Change{}, fmt.Errorf("commit: %w", err)
	}
	return ch, nil
}

func snapshot(ctx context.Context, q *sqlc.Queries, capacity int32, reg sqlc.Registration) (Snapshot, error) {
	attendees, err := q.ListAttendees(ctx, reg.ID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("ambil peserta: %w", err)
	}
	sum, err := q.CountSummary(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("hitung peserta: %w", err)
	}
	return Snapshot{
		Status:     reg.Status,
		Attendees:  len(attendees),
		Confirmed:  sum.ConfirmedAttendees,
		Waitlisted: sum.WaitlistedAttendees,
		Capacity:   capacity,
	}, nil
}

// Nil berarti field itu tidak diubah.
type AttendeePatch struct {
	FullName    *string
	Email       *string
	AccountName *string
}

type AttendeeChange struct {
	Code     string
	Position int
	Before   Attendee
	After    Attendee
}

func (s *Service) UpdateAttendee(ctx context.Context, rawCode string, position int, patch AttendeePatch) (AttendeeChange, error) {
	code, ok := NormalizeCode(rawCode)
	if !ok {
		return AttendeeChange{}, errors.New("format kode tidak valid, contoh: NVM-DD-ABC234")
	}
	if position < 0 || position > maxGuestsHard {
		return AttendeeChange{}, fmt.Errorf("posisi harus 0 (pendaftar) sampai %d (tamu)", maxGuestsHard)
	}

	q := sqlc.New(s.pool)
	reg, err := q.GetRegistrationByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendeeChange{}, ErrNotFound
	}
	if err != nil {
		return AttendeeChange{}, fmt.Errorf("ambil registrasi: %w", err)
	}
	attendees, err := q.ListAttendees(ctx, reg.ID)
	if err != nil {
		return AttendeeChange{}, fmt.Errorf("ambil peserta: %w", err)
	}

	var current *sqlc.Attendee
	for i := range attendees {
		if int(attendees[i].Position) == position {
			current = &attendees[i]
		}
	}
	if current == nil {
		return AttendeeChange{}, fmt.Errorf("registrasi ini tidak punya peserta di posisi %d", position)
	}

	before := Attendee{FullName: current.FullName, Email: current.Email, AccountName: current.AccountName}
	req := AttendeeRequest{FullName: before.FullName, Email: before.Email, AccountName: before.AccountName}
	if patch.FullName != nil {
		req.FullName = *patch.FullName
	}
	if patch.Email != nil {
		req.Email = *patch.Email
	}
	if patch.AccountName != nil {
		req.AccountName = *patch.AccountName
	}

	after, errs := ValidateAttendee(req)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Message
		}
		return AttendeeChange{}, errors.New(strings.Join(msgs, " "))
	}
	if after == before {
		return AttendeeChange{}, errors.New("tidak ada yang berubah")
	}

	_, err = q.UpdateAttendee(ctx, sqlc.UpdateAttendeeParams{
		FullName:    after.FullName,
		Email:       after.Email,
		AccountName: after.AccountName,
		Code:        code,
		Position:    int16(position),
	})
	if isUniqueViolation(err, "attendees_email_active_key") {
		return AttendeeChange{}, errors.New("email ini sudah dipakai peserta aktif lain")
	}
	if err != nil {
		return AttendeeChange{}, fmt.Errorf("simpan perubahan: %w", err)
	}
	return AttendeeChange{Code: code, Position: position, Before: before, After: after}, nil
}
