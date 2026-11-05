-- Daftar nama akun yang mendapat formulir VIP. Diisi admin, tidak pernah dihapus, hanya dinonaktifkan,
-- karena doomsday_app sengaja tidak punya izin DELETE.
CREATE TABLE vip_accounts (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Seperti yang diketik admin, untuk ditampilkan.
    account_name varchar(50) NOT NULL CHECK (btrim(account_name) <> ''),
    -- Kunci pencocokan, dinormalisasi di Go (trim, tanpa @ di depan, huruf kecil). CHECK ini menjaga
    -- jalur lain seperti input manual tidak menyimpan bentuk mentah, karena index unik di bawah bergantung padanya.
    account_key  varchar(50) NOT NULL CHECK (
        account_key <> ''
        AND account_key = btrim(account_key)
        AND account_key = lower(account_key)
        AND account_key NOT LIKE '@%'
    ),
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Unik juga untuk baris nonaktif: menambah nama yang pernah dinonaktifkan berarti mengaktifkannya lagi.
CREATE UNIQUE INDEX vip_accounts_account_key_key ON vip_accounts (account_key);

ALTER TABLE registrations
    ADD COLUMN vip_account_id uuid REFERENCES vip_accounts (id),
    -- Hanya satu arah. Registrasi VIP dari masa uji tidak punya akun VIP, dan CHECK NOT VALID tetap
    -- dicek saat baris lama di-UPDATE, sehingga cancel lewat CLI akan gagal. Arah lainnya dijaga kode.
    ADD CONSTRAINT registrations_vip_account_only_vip CHECK (vip_account_id IS NULL OR type = 'vip');

-- Satu nama VIP untuk satu registrasi yang belum batal. Penjaga terakhir saat dua pendaftaran
-- dengan nama yang sama masuk bersamaan. Setelah dibatalkan, nama itu bisa dipakai lagi.
CREATE UNIQUE INDEX registrations_vip_account_active_key
    ON registrations (vip_account_id)
    WHERE status <> 'cancelled';
