-- name: GetEvent :one
SELECT * FROM event_config WHERE id = 1;

-- name: LockEvent :one
-- Dipanggil di awal transaksi pendaftaran. Mengunci baris acara sehingga pendaftaran diproses
-- satu per satu, dan hitungan sisa kursi tidak bisa balapan (NFR-06).
SELECT capacity, max_guests_vip, is_open FROM event_config WHERE id = 1 FOR UPDATE;

-- name: UpdateEvent :one
UPDATE event_config SET
    title = @title,
    film_title = @film_title,
    cinema_name = @cinema_name,
    city = @city,
    studio = @studio,
    gather_at = @gather_at,
    show_at = @show_at,
    map_url = @map_url,
    capacity = @capacity,
    max_guests_vip = @max_guests_vip,
    is_open = @is_open,
    updated_at = now()
WHERE id = 1
RETURNING *;
