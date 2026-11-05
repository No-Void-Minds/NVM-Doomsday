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
