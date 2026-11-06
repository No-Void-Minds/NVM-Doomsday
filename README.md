# NVM Doomsday

Halaman undangan nonton bareng Avengers: Doomsday untuk affiliator No Void Minds. Pengunjung mengisi nama akun, lalu sistem mencocokkannya dengan daftar VIP yang dikelola admin. Nama yang cocok dapat formulir VIP (boleh membawa sampai 3 tamu). Nama yang tidak cocok dapat formulir Solo. Setelah terdaftar, pengunjung menerima kode dan kartu akses yang bisa disimpan sebagai gambar.

Situs: https://letter.novoidminds.com

## Fitur

- Pendaftaran dua langkah: cek nama akun, lalu formulir VIP atau Solo.
- Satu nama VIP hanya bisa dipakai satu registrasi aktif.
- Kuota kursi dengan daftar tunggu otomatis saat kursi penuh.
- Kartu akses yang bisa dibagikan atau diunduh sebagai PNG.
- Halaman admin: ringkasan peserta, ekspor Excel, dan pengelolaan daftar VIP.
- CLI admin untuk konfigurasi acara, promote, cancel, rename, dan VIP.

## Teknologi

| Bagian | Stack |
|---|---|
| Backend | Go 1.27, Gin, pgx v5, sqlc, golang-migrate, excelize |
| Frontend | Vue 3, TypeScript, Vite, Tailwind CSS v4 |
| Database | PostgreSQL 18 |
| Deploy | Docker Compose (backend distroless, frontend Nginx), Cloudflare Tunnel |
| Test | go test (unit, integration, race), staticcheck, Vitest, Playwright, axe-core |

## Struktur repo

```
backend/
  cmd/server/           server HTTP
  cmd/doomsday-admin/   CLI admin
  internal/             registrasi, vip, adminauth, export, httpapi, db (hasil sqlc)
  migrations/           migrasi SQL golang-migrate
frontend/
  src/                  aplikasi Vue
  e2e/                  test Playwright
  nginx.conf            konfigurasi Nginx untuk container frontend
deploy/
  docker-compose.yml
  .env.example
  backup.sh             backup harian pg_dump dengan retensi 14 hari
scripts/
  check.sh              pemeriksaan wajib sebelum commit
  dev-db-tunnel.sh      SSH tunnel ke Postgres di server
  test-integration.sh
  test-e2e.sh
testdata/               kasus uji bersama untuk Go dan TypeScript
docs/                   catatan sprint, SRS, dan panduan admin
```

## Menjalankan di lokal

Kebutuhan: Go sesuai `backend/go.mod`, Node.js dan npm, sqlc, golang-migrate v4 (tag postgres), dan akses SSH ke server database.

1. Buka tunnel ke database, biarkan terminalnya tetap terbuka:

   ```bash
   ./scripts/dev-db-tunnel.sh
   ```

2. Siapkan konfigurasi backend. Penjelasan tiap variabel ada di file contohnya.

   ```bash
   cp backend/.env.example backend/.env
   ```

3. Jalankan migrasi dan server:

   ```bash
   cd backend
   set -a; source .env; set +a
   migrate -path migrations -database "$MIGRATE_DATABASE_URL" up
   go run ./cmd/server
   ```

4. Jalankan frontend di terminal lain:

   ```bash
   cd frontend
   npm ci
   npm run dev
   ```

Hash sandi admin dibuat dengan `go run ./cmd/doomsday-admin hash-password`, lalu ditempel ke `ADMIN_PASSWORD_HASH` dengan kutip tunggal.

## Pemeriksaan dan test

Wajib lulus sebelum commit:

```bash
./scripts/check.sh
```

Script ini menjalankan gofmt, sqlc diff, go vet, staticcheck, dan unit test Go, lalu type check, ESLint, Prettier, dan Vitest untuk frontend, ditutup pengecekan em dash di teks.

Integration test dan E2E butuh database uji. Isi `TEST_DATABASE_URL` dan `TEST_MIGRATE_DATABASE_URL` di `backend/.env`. Nama database wajib berakhiran `_test`, karena isinya dikosongkan di antara test dan harness menolak jalan kalau tidak.

```bash
./scripts/test-integration.sh
./scripts/test-e2e.sh
```

## Deploy

Aplikasi berjalan dengan Docker Compose di server internal NVM dan dibuka ke publik lewat Cloudflare Tunnel.

```bash
cp deploy/.env.example deploy/.env   # isi lalu chmod 600
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d --build
```

Langkah lengkap (migrasi produksi, backup, restore, dan pengelolaan VIP) ada di panduan admin di folder `docs/`.

## Keamanan

- `backend/.env` dan `deploy/.env` tidak pernah masuk git. Repo ini public.
- Role database aplikasi hanya punya SELECT, INSERT, dan UPDATE. Migrasi memakai role terpisah.
- Halaman admin hanya bisa dipakai lewat HTTPS karena cookie sesinya Secure.
