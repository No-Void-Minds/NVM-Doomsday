-- Lokasi, jadwal, dan link peta sengaja NULL: belum diumumkan, ditampilkan "Menyusul" di frontend,
-- dan diisi lewat CLI admin setelah booking studio pasti.
INSERT INTO event_config (id, title, film_title, capacity, max_guests_vip, is_open)
VALUES (1, 'No Void Minds Doomsday', 'Avengers: Doomsday', 200, 3, true)
ON CONFLICT (id) DO NOTHING;
