-- Backend tidak perlu membaca atau mengubah riwayat migrasi.
REVOKE ALL ON TABLE schema_migrations FROM doomsday_app;

CREATE TYPE registration_type AS ENUM ('solo', 'vip');
CREATE TYPE registration_status AS ENUM ('confirmed', 'waitlisted', 'cancelled');
CREATE TYPE attendee_role AS ENUM ('primary', 'guest');

CREATE TABLE event_config (
    -- Hanya satu acara di v1. CHECK ini mencegah baris kedua terbentuk tanpa sengaja.
    id             smallint PRIMARY KEY CHECK (id = 1),
    title          text NOT NULL CHECK (btrim(title) <> ''),
    film_title     text NOT NULL CHECK (btrim(film_title) <> ''),
    cinema_name    text,
    city           text,
    studio         text,
    gather_at      timestamptz,
    show_at        timestamptz,
    map_url        text CHECK (map_url IS NULL OR map_url LIKE 'https://%'),
    capacity       integer NOT NULL CHECK (capacity > 0),
    max_guests_vip smallint NOT NULL DEFAULT 3 CHECK (max_guests_vip BETWEEN 1 AND 3),
    is_open        boolean NOT NULL DEFAULT true,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (gather_at IS NULL OR show_at IS NULL OR gather_at <= show_at)
);

CREATE TABLE registrations (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Alfabet tanpa O, I, L, 0, 1 supaya kode mudah dibacakan saat check-in (FR-21).
    code              varchar(16) NOT NULL UNIQUE CHECK (code ~ '^NVM-DD-[A-HJKMNP-Z2-9]{6}$'),
    type              registration_type NOT NULL,
    status            registration_status NOT NULL,
    idempotency_key   uuid NOT NULL UNIQUE,
    consent_at        timestamptz NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    status_changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE attendees (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_id uuid NOT NULL REFERENCES registrations (id) ON DELETE CASCADE,
    role            attendee_role NOT NULL,
    position        smallint NOT NULL CHECK (position BETWEEN 0 AND 3),
    full_name       varchar(100) NOT NULL CHECK (btrim(full_name) <> ''),
    -- Backend menyimpan email dalam huruf kecil; CHECK ini menjaga jalur lain (CLI, input manual)
    -- tidak lupa, karena index unik email di langkah 4 bergantung padanya.
    email           varchar(254) NOT NULL CHECK (email = lower(email) AND email LIKE '%_@_%._%'),
    account_name    varchar(50) NOT NULL CHECK (btrim(account_name) <> ''),
    -- Salinan status registrasi: false saat dibatalkan. Dibutuhkan index unik email di langkah 4,
    -- karena index tidak bisa membaca kolom dari tabel registrations.
    is_active       boolean NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (registration_id, position),
    -- Posisi 0 selalu pendaftar utama, posisi 1 sampai 3 selalu tamu.
    CHECK ((role = 'primary') = (position = 0))
);
