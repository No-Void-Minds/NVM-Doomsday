-- name: GetRegistrationByIdempotencyKey :one
SELECT * FROM registrations WHERE idempotency_key = @idempotency_key;

-- name: GetRegistrationByCode :one
SELECT * FROM registrations WHERE code = @code;

-- name: FindActiveEmails :many
-- Dipakai sebelum insert untuk menunjuk baris mana yang emailnya sudah terdaftar (FR-15).
-- Index unik attendees_email_active_key tetap jadi penjaga terakhir.
SELECT email FROM attendees WHERE is_active AND email = ANY(@emails::text[]);

-- name: InsertRegistration :one
-- Waktu memakai clock_timestamp(), bukan now(). now() adalah waktu transaksi dimulai, padahal
-- urutan sebenarnya ditentukan saat kunci acara didapat. Dengan clock_timestamp(), urutan created_at
-- sama dengan urutan pendaftaran diproses, jadi nomor daftar tunggu tidak bergeser setelah ditampilkan.
INSERT INTO registrations (code, type, status, idempotency_key, consent_at, created_at, status_changed_at)
VALUES (@code, @type, @status, @idempotency_key, clock_timestamp(), clock_timestamp(), clock_timestamp())
RETURNING *;

-- name: InsertAttendee :exec
INSERT INTO attendees (registration_id, role, position, full_name, email, account_name)
VALUES (@registration_id, @role, @position, @full_name, @email, @account_name);

-- name: ListAttendees :many
SELECT * FROM attendees WHERE registration_id = @registration_id ORDER BY position;

-- name: WaitlistPosition :one
SELECT count(*) FROM registrations w
WHERE w.status = 'waitlisted'
  AND (w.created_at, w.id) <= (SELECT r.created_at, r.id FROM registrations r WHERE r.id = @id);

-- name: UpdateRegistrationStatus :one
-- status_changed_at dan attendees.is_active diperbarui oleh trigger di migrasi 0002.
UPDATE registrations SET status = @status WHERE code = @code RETURNING *;

-- name: UpdateAttendee :one
UPDATE attendees
SET full_name = @full_name, email = @email, account_name = @account_name
WHERE registration_id = (SELECT id FROM registrations WHERE code = @code)
  AND position = @position
RETURNING *;
