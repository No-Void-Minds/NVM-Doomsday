-- name: CountSummary :one
-- Dipakai GET /api/event (sisa kursi, jumlah daftar tunggu) dan ringkasan admin (FR-37).
SELECT
    count(a.id) FILTER (WHERE r.status = 'confirmed') AS confirmed_attendees,
    count(a.id) FILTER (WHERE r.status = 'waitlisted') AS waitlisted_attendees,
    count(DISTINCT r.id) FILTER (WHERE r.status = 'cancelled') AS cancelled_registrations
FROM registrations r
LEFT JOIN attendees a ON a.registration_id = r.id;

-- name: ListExportRows :many
-- Urutan FR-33: terkonfirmasi, lalu daftar tunggu, lalu batal, masing-masing urut waktu daftar.
-- Di dalam satu registrasi, pendaftar utama (posisi 0) selalu di atas tamunya.
-- Nomor urut daftar tunggu dihitung di Go saat membaca baris, karena urutannya sudah benar di sini.
SELECT
    r.code,
    r.status,
    r.type,
    r.created_at,
    a.role,
    a.position,
    a.full_name,
    a.email,
    a.account_name,
    p.full_name AS primary_name
FROM registrations r
JOIN attendees a ON a.registration_id = r.id
JOIN attendees p ON p.registration_id = r.id AND p.position = 0
ORDER BY
    CASE r.status WHEN 'confirmed' THEN 0 WHEN 'waitlisted' THEN 1 ELSE 2 END,
    r.created_at,
    r.id,
    a.position;
