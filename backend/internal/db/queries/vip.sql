-- name: FindVipAccount :one
-- Nama nonaktif dianggap bukan VIP. used = true kalau nama ini sudah dipakai registrasi yang belum batal.
SELECT
    v.id,
    EXISTS (
        SELECT 1 FROM registrations r
        WHERE r.vip_account_id = v.id AND r.status <> 'cancelled'
    ) AS used
FROM vip_accounts v
WHERE v.account_key = @account_key AND v.is_active;

-- name: ListVipAccounts :many
-- Index unik parsial di 0004 menjamin paling banyak satu registrasi aktif per nama,
-- jadi LEFT JOIN ini tidak menggandakan baris. Kode kosong berarti nama belum dipakai.
SELECT
    v.id,
    v.account_name,
    v.is_active,
    v.created_at,
    COALESCE(r.code, '')::text AS registration_code
FROM vip_accounts v
LEFT JOIN registrations r ON r.vip_account_id = v.id AND r.status <> 'cancelled'
ORDER BY v.is_active DESC, v.account_key;

-- name: GetVipAccountByKey :one
SELECT * FROM vip_accounts WHERE account_key = @account_key;

-- name: InsertVipAccount :one
-- Kalau nama yang sama masuk bersamaan dari jalur lain, baris kedua dilewati dan tidak ada baris yang kembali.
INSERT INTO vip_accounts (account_name, account_key)
VALUES (@account_name, @account_key)
ON CONFLICT (account_key) DO NOTHING
RETURNING *;

-- name: SetVipAccountActive :one
UPDATE vip_accounts SET is_active = @is_active, updated_at = now()
WHERE id = @id
RETURNING *;
