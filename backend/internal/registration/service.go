package registration

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

var ErrClosed = errors.New("pendaftaran ditutup")

// ValidationError untuk aturan yang baru bisa dicek di dalam transaksi, misalnya batas tamu VIP saat ini.
type ValidationError struct{ Fields []FieldError }

func (e *ValidationError) Error() string { return "input pendaftaran tidak valid" }

type DuplicateEmailError struct{ Fields []FieldError }

func (e *DuplicateEmailError) Error() string { return "email sudah terdaftar" }

type Result struct {
	Code   string
	Type   sqlc.RegistrationType
	Status sqlc.RegistrationStatus
	Names  []string
	// Nomor urut di daftar tunggu, hanya terisi bila Status waitlisted.
	WaitlistPosition int64
	// True kalau idempotency key sudah pernah dipakai dan hasil lama yang dikembalikan.
	Replayed bool
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

const maxCodeAttempts = 5

func (s *Service) Register(ctx context.Context, in Input) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("mulai transaksi: %w", err)
	}
	// Tidak berpengaruh setelah Commit berhasil; membatalkan transaksi di semua jalur keluar lain.
	defer tx.Rollback(ctx)

	q := sqlc.New(s.pool).WithTx(tx)

	// Semua pendaftaran antre di kunci ini, jadi cek di bawahnya tidak bisa balapan (NFR-06).
	ev, err := q.LockEvent(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("kunci event: %w", err)
	}

	// Dicek setelah kunci, supaya dua kiriman dengan key yang sama tidak sama-sama lolos.
	existing, err := q.GetRegistrationByIdempotencyKey(ctx, in.IdempotencyKey)
	switch {
	case err == nil:
		return s.load(ctx, q, existing, true)
	case !errors.Is(err, pgx.ErrNoRows):
		return Result{}, fmt.Errorf("cek idempotency key: %w", err)
	}

	if !ev.IsOpen {
		return Result{}, ErrClosed
	}

	guests := len(in.Attendees) - 1
	if guests > int(ev.MaxGuestsVip) {
		return Result{}, &ValidationError{Fields: []FieldError{{
			Field:   "attendees",
			Message: fmt.Sprintf("Maksimal %d tamu.", ev.MaxGuestsVip),
		}}}
	}

	if fields, err := duplicateEmails(ctx, q, in.Attendees); err != nil {
		return Result{}, err
	} else if len(fields) > 0 {
		return Result{}, &DuplicateEmailError{Fields: fields}
	}

	sum, err := q.CountSummary(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("hitung peserta: %w", err)
	}
	// Rombongan tidak pernah dipecah: masuk semua, atau semua ke daftar tunggu (FR-17).
	status := sqlc.RegistrationStatusWaitlisted
	if int64(len(in.Attendees)) <= int64(ev.Capacity)-sum.ConfirmedAttendees {
		status = sqlc.RegistrationStatusConfirmed
	}

	code, err := uniqueCode(ctx, q)
	if err != nil {
		return Result{}, err
	}

	reg, err := q.InsertRegistration(ctx, sqlc.InsertRegistrationParams{
		Code:           code,
		Type:           in.Type,
		Status:         status,
		IdempotencyKey: in.IdempotencyKey,
	})
	if err != nil {
		return Result{}, fmt.Errorf("simpan registrasi: %w", err)
	}

	for i, a := range in.Attendees {
		role := sqlc.AttendeeRoleGuest
		if i == 0 {
			role = sqlc.AttendeeRolePrimary
		}
		err := q.InsertAttendee(ctx, sqlc.InsertAttendeeParams{
			RegistrationID: reg.ID,
			Role:           role,
			Position:       int16(i),
			FullName:       a.FullName,
			Email:          a.Email,
			AccountName:    a.AccountName,
		})
		if isUniqueViolation(err, "attendees_email_active_key") {
			// Seharusnya sudah tertangkap duplicateEmails di atas. Ini jaring terakhir dari index unik.
			return Result{}, &DuplicateEmailError{Fields: []FieldError{{
				Field:   fmt.Sprintf("attendees[%d].email", i),
				Message: "Email ini sudah terdaftar.",
			}}}
		}
		if err != nil {
			return Result{}, fmt.Errorf("simpan peserta %d: %w", i, err)
		}
	}

	res, err := s.load(ctx, q, reg, false)
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit pendaftaran: %w", err)
	}
	return res, nil
}

func (s *Service) load(ctx context.Context, q *sqlc.Queries, reg sqlc.Registration, replayed bool) (Result, error) {
	attendees, err := q.ListAttendees(ctx, reg.ID)
	if err != nil {
		return Result{}, fmt.Errorf("ambil peserta: %w", err)
	}
	res := Result{
		Code:     reg.Code,
		Type:     reg.Type,
		Status:   reg.Status,
		Names:    make([]string, 0, len(attendees)),
		Replayed: replayed,
	}
	for _, a := range attendees {
		res.Names = append(res.Names, a.FullName)
	}
	if reg.Status == sqlc.RegistrationStatusWaitlisted {
		pos, err := q.WaitlistPosition(ctx, reg.ID)
		if err != nil {
			return Result{}, fmt.Errorf("hitung nomor urut: %w", err)
		}
		res.WaitlistPosition = pos
	}
	return res, nil
}

func duplicateEmails(ctx context.Context, q *sqlc.Queries, attendees []Attendee) ([]FieldError, error) {
	emails := make([]string, len(attendees))
	for i, a := range attendees {
		emails[i] = a.Email
	}
	taken, err := q.FindActiveEmails(ctx, emails)
	if err != nil {
		return nil, fmt.Errorf("cek email terdaftar: %w", err)
	}
	if len(taken) == 0 {
		return nil, nil
	}

	takenSet := make(map[string]struct{}, len(taken))
	for _, e := range taken {
		takenSet[e] = struct{}{}
	}
	var fields []FieldError
	for i, a := range attendees {
		if _, ok := takenSet[a.Email]; ok {
			// Tidak menyebut siapa pemiliknya, hanya bahwa email itu sudah dipakai.
			fields = append(fields, FieldError{
				Field:   fmt.Sprintf("attendees[%d].email", i),
				Message: "Email ini sudah terdaftar.",
			})
		}
	}
	return fields, nil
}

// Cek sebelum insert aman dari balapan karena dijalankan di bawah kunci event.
func uniqueCode(ctx context.Context, q *sqlc.Queries) (string, error) {
	for range maxCodeAttempts {
		code, err := NewCode()
		if err != nil {
			return "", err
		}
		_, err = q.GetRegistrationByCode(ctx, code)
		if errors.Is(err, pgx.ErrNoRows) {
			return code, nil
		}
		if err != nil {
			return "", fmt.Errorf("cek kode kembar: %w", err)
		}
	}
	return "", fmt.Errorf("gagal membuat kode unik setelah %d percobaan", maxCodeAttempts)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
