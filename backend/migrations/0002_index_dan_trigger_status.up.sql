-- Satu email hanya boleh dipakai satu peserta aktif (FR-15). Peserta dari registrasi yang dibatalkan
-- tidak dihitung, supaya orangnya bisa mendaftar lagi. Email sudah dijamin huruf kecil oleh CHECK di 0001,
-- jadi index tidak perlu lower().
CREATE UNIQUE INDEX attendees_email_active_key ON attendees (email) WHERE is_active;

-- Dipakai untuk menghitung nomor urut daftar tunggu dan mengurutkan file ekspor.
CREATE INDEX registrations_status_created_at_idx ON registrations (status, created_at);

CREATE FUNCTION registrations_touch_status_changed_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.status_changed_at := now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER registrations_touch_status_changed_at
BEFORE UPDATE OF status ON registrations
FOR EACH ROW
WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION registrations_touch_status_changed_at();

-- Menyalin status ke attendees.is_active. Kalau registrasi dipulihkan dari cancelled padahal
-- emailnya sudah dipakai pendaftar lain, index unik di atas menolak perubahan itu, dan itu memang benar.
CREATE FUNCTION registrations_sync_attendees_active() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE attendees
    SET is_active = (NEW.status <> 'cancelled')
    WHERE registration_id = NEW.id;
    RETURN NULL;
END;
$$;

CREATE TRIGGER registrations_sync_attendees_active
AFTER UPDATE OF status ON registrations
FOR EACH ROW
WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION registrations_sync_attendees_active();
