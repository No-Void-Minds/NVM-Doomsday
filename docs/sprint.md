# Sprint nvm-doomsday

Rencana kerja landing page undangan nonton bareng Avengers: Doomsday untuk No Void Minds.
Acuan kebutuhan: SRS v0.2 (`docs/SRS-NVM-Doomsday-v0.2.docx`). Kode FR/NFR di bawah merujuk ke dokumen itu.
Acuan tampilan: proyek Stitch "Void screening invitation system" (versi final yang sudah direview).

## Cara kerja

- Satu langkah dikerjakan sampai jalan, baru lanjut ke langkah berikutnya.
- Setiap langkah selesai: `./scripts/check.sh && git commit && git push`.
- Pengujian menyeluruh dilakukan di Sprint 9 setelah semua fitur selesai. Sampai saat itu, `check.sh` menjaga format, lint, dan type check.
- Status: `belum`, `jalan`, `selesai`, `tertahan` (menunggu keputusan).

## Ringkasan status

| Sprint | Isi | Status |
|---|---|---|
| 0 | Fondasi repo, tooling, check.sh | belum |
| 1 | Database dan migrasi (backend) | belum |
| 2 | API publik: event dan pendaftaran (backend) | belum |
| 3 | API admin, ekspor, CLI admin (backend) | belum |
| 4 | Kerangka dan komponen dasar (frontend) | belum |
| 5 | Halaman undangan (frontend) | belum |
| 6 | Formulir dan kartu akses (frontend) | belum |
| 7 | Halaman admin (frontend) | belum |
| 8 | Docker dan deploy LAN | belum |
| 9 | Pengujian menyeluruh | belum |
| 10 | Rilis publik lewat Cloudflare Tunnel | tertahan |

## Stack

- Backend (`backend/`): Go 1.27, Gin, pgx/v5, sqlc 1.31.1, golang-migrate, excelize (XLSX), encoding/csv.
- Frontend (`frontend/`): Vue 3.5, TypeScript, Vite, Tailwind v4, Vue Router. Tanpa Pinia, karena state cukup di composable.
- Font: @fontsource Archivo, Archivo Narrow, IBM Plex Mono (di-host sendiri).
- Database: PostgreSQL 18 di container yang sudah ada di VM `nvm-server`. Database `nvm_doomsday`, role `doomsday_app` (hanya punya akses ke database itu).
- Deploy: Docker Compose di VM, akses LAN `http://<IP VM>:8090`.

## Struktur repo

nvm-doomsday/
  backend/
    cmd/
      server/main.go
      doomsday-admin/main.go
    internal/
      config/
      db/
        queries/       file .sql untuk sqlc
        sqlc/          hasil generate, jangan diedit manual
      httpapi/
      registration/
      export/
    migrations/
    sqlc.yaml
    .env.example
    go.mod
  frontend/            dibuat oleh create-vite
    src/
      api/
      components/
      composables/
      pages/
      router/
      styles/
  deploy/              diisi di Sprint 8
  docs/
  scripts/
  .gitignore
  README.md

Backend dan frontend tidak saling import. Satu-satunya kontrak di antara keduanya adalah API di SRS bagian 7.

## Port

| Layanan | Dev (laptop) | Server (VM) |
|---|---|---|
| Backend | 127.0.0.1:8081 | internal container, tidak dibuka ke LAN |
| Frontend | 127.0.0.1:5174 (Vite, proxy `/api` ke 8081) | 0.0.0.0:8090 |
| Postgres | 127.0.0.1:15432 lewat SSH tunnel | container yang sudah ada |

Port 8081 dan 5174 dipilih agar tidak bentrok dengan NVM-Internal (8080, 5173).

---

## Sprint 0: Fondasi

Tujuan: repo siap dikerjakan, `check.sh` hijau dengan proyek kosong.

1. Buat repo `nvm-doomsday` dengan folder `backend/`, `frontend/`, `deploy/`, `scripts/`, `docs/`, plus `.gitignore` dan `.editorconfig` di root.
2. Backend: `go mod init` di `backend/`, `cmd/api/main.go` yang menjawab `GET /api/health`, `.env.example`.
3. Frontend: aplikasi Vite Vue TS di `frontend/`, pasang Tailwind v4, ESLint, Prettier, vue-tsc, proxy `/api` ke backend di `vite.config.ts`, `.env.example`.
4. Token warna, font, radius dari SRS 11.3 sampai 11.5 di `frontend/src/styles/tokens.css` (sebagai `@theme` Tailwind). Pasang @fontsource.
5. `scripts/check.sh` di root: bagian backend (gofmt, go vet, go test ./...) dijalankan dari `backend/`, bagian frontend (vue-tsc --noEmit, eslint, prettier --check) dari `frontend/`. Berhenti di error pertama dan sebutkan bagian mana yang gagal.
6. `scripts/dev-db-tunnel.sh` untuk membuka SSH tunnel ke Postgres di VM (port lokal 15432).

Selesai bila: `./scripts/check.sh` lulus, backend dan frontend jalan bersamaan, halaman frontend memanggil `/api/health` lewat proxy dan memakai font serta warna dari token.

### Laporan

Status: selesai, 30 Oktober 2026.

Hasil:
- `./scripts/check.sh` lulus untuk backend dan frontend.
- Backend menjawab `GET /api/health`, frontend memanggilnya lewat proxy Vite dengan state memuat, berhasil, dan gagal.
- Token warna, font (di-host sendiri), radius, dan bayangan dari SRS 11.3 sampai 11.5 terpasang di `frontend/src/styles/tokens.css`.
- `scripts/dev-db-tunnel.sh` terbukti menyambung ke Postgres di VM lewat port lokal 15432.

Perubahan dari rencana:
- Struktur repo mengikuti pola NVM-Internal: `cmd/server`, `internal/db/queries` dan `internal/db/sqlc`, `internal/httpapi`, `composables/` sebagai pengganti `stores/`. `.env.example` hanya di backend. Tidak ada `docker-compose.dev.yml` karena database dev memakai Postgres di VM.
- Frontend dibuat dengan create-vue (TypeScript, Router, ESLint, Prettier), tanpa fitur eksperimental.
- Ditambahkan `.gitattributes` (akhir baris LF) karena pengembangan di Windows sementara script dijalankan di Linux.
- Ditambahkan `.vscode/extensions.json` dan `.vscode/settings.json` supaya editor selalu menambahkan newline di akhir file dan memformat saat simpan.
- `check.sh` ditambah cek em dash di teks backend dan frontend (aturan AntiSlop).
- Izin eksekusi script dicatat di Git dengan `git add --chmod=+x`.

Bentol yang ditemui:
- Prettier dan gofmt menolak file yang tidak diakhiri newline. Penyebab: editor tidak membaca `.editorconfig`. Diselesaikan dengan `.vscode/settings.json`.
- Sempat mencoba login Postgres dengan user Linux `nvmadmin`. User Postgres berbeda dari user SSH VM.

## Sprint 1: Database (backend)

Tujuan: skema sesuai SRS bagian 6, bisa dimigrasi naik dan turun.

## Sprint 1: Database (backend)

Tujuan: skema sesuai SRS bagian 6, bisa dimigrasi naik dan turun.

1. Di VM: database `nvm_doomsday`, role `doomsday_migrator` (pemilik) dan `doomsday_app` (SELECT, INSERT, UPDATE lewat default privileges). Izin CONNECT dicabut dari PUBLIC di `nvm_doomsday`, `nvm_internal`, dan `nvm_internal_test`.
2. Backend tersambung lewat pgxpool (maksimal 10 koneksi), `GET /api/health` ikut mengecek database dan menjawab 503 saat database putus.
3. Migrasi 0001: enum `registration_type`, `registration_status`, `attendee_role`; tabel `event_config`, `registrations`, `attendees` dengan CHECK untuk format kode, email huruf kecil, dan posisi peserta.
4. Migrasi 0002: index unik email untuk peserta aktif, index `registrations(status, created_at)`, trigger yang menjaga `status_changed_at` dan `attendees.is_active` sinkron dengan status.
5. Migrasi 0003: seed `event_config` (capacity 200, max_guests_vip 3, lokasi dan jadwal NULL).
6. sqlc: query acara, pendaftaran, ringkasan admin, dan ekspor. `check.sh` menjalankan `sqlc diff`.
7. Uji migrate down dan up penuh.

Selesai bila: `migrate down -all` dan `migrate up` bersih, izin dan seed terbentuk ulang, `sqlc generate` tanpa error, check.sh lulus.
Referensi: SRS 6.1 sampai 6.3, NFR-06.

### Laporan

Status: selesai, 30 Oktober 2026.

Perubahan dari rencana:
- Dua role database, bukan satu. `doomsday_app` tidak punya DDL dan DELETE, supaya celah di backend tidak bisa menghapus tabel atau data peserta.
- Kolom `attendees.is_active` ditambahkan. Index unik email hanya untuk peserta aktif butuh kolom ini, karena index tidak bisa membaca kolom dari tabel `registrations`. Nilainya dijaga trigger, bukan kode aplikasi.
- Index terpisah untuk `attendees(registration_id)` tidak dibuat, karena sudah tercakup index dari `UNIQUE (registration_id, position)`.
- Nomor migrasi: 0001 tabel, 0002 index dan trigger, 0003 seed. Migrasi yang sudah dijalankan tidak diedit, perubahan selalu jadi migrasi baru.
- `created_at` registrasi memakai `clock_timestamp()` supaya urutan daftar tunggu sama dengan urutan pendaftaran diproses.

Bentol yang ditemui:
- Query hak akses sempat dijalankan di database `postgres` karena koneksi TablePlus belum pindah database. Sudah dikembalikan ke pengaturan bawaan. Pelajaran: selalu cek nama database di bar atas TablePlus sebelum menjalankan SQL yang mengubah izin.
- Sandi role sempat tidak cocok dan nama database ikut spasi saat ditempel. Diselesaikan dengan `ALTER ROLE ... PASSWORD` dan mengetik ulang nama database.

Catatan di luar repo ini:
- Backend NVM-Internal login sebagai `nvm_internal`, yang superuser. Sebaiknya NVM-Internal punya role aplikasi sendiri seperti `doomsday_app`, dan `nvm_internal` hanya untuk administrasi.
- Riwayat query TablePlus menyimpan perintah `CREATE ROLE` dan `ALTER ROLE` beserta sandinya. Hapus dari tab History.

Dipindah ke Sprint 9:
- Uji health saat database putus, pulih tanpa restart, dan gagal di awal saat database tidak terjangkau.
- Uji trigger: membatalkan registrasi membuat email pesertanya bisa dipakai lagi.

Selesai bila: `migrate up` dan `migrate down` bersih di database dev, `sqlc generate` tanpa error, check.sh lulus.
Referensi: SRS 6.1 sampai 6.3, NFR-06.

- Database: PostgreSQL 18 di container yang sudah ada di VM `nvm-server`. Database `nvm_doomsday` dengan dua role: `doomsday_migrator` (pemilik, hanya untuk migrasi) dan `doomsday_app` (backend dan CLI, hanya SELECT, INSERT, UPDATE).

## Sprint 2: API publik (backend)

Tujuan: dua endpoint publik bekerja sesuai SRS bagian 7.

1. Konfigurasi dari env, logger terstruktur dengan request ID, format error JSON yang seragam (`code`, `message`, `fields`).
2. `GET /api/event`: detail acara, `is_open`, sisa kursi, jumlah daftar tunggu, `max_guests_vip`. Field kosong dikirim `null` agar frontend menampilkan "Menyusul".
3. Validasi input `POST /api/registrations`: tipe, jumlah tamu (Solo 0, VIP 1 sampai 3), nama 1 sampai 100, email valid maksimal 254, nama akun 1 sampai 50, trim spasi, email lowercase, tidak ada email ganda dalam satu registrasi, persetujuan wajib, honeypot harus kosong.
4. Generator kode `NVM-DD-XXXXXX` (6 karakter dari alfabet tanpa O, I, L, 0, 1), dibuat dengan crypto/rand, coba ulang bila bentrok.
5. Transaksi pendaftaran: kunci baris event, cek `is_open`, cek email terdaftar, hitung sisa kursi, tentukan status `confirmed` atau `waitlisted` untuk seluruh rombongan, simpan, commit. Kirim balik nomor urut bila daftar tunggu.
6. Idempotency key: kalau key yang sama datang lagi, kembalikan registrasi yang sudah ada dengan status 200.
7. Rate limit per IP untuk POST (baca IP dari header yang hanya dipercaya dari Nginx), CORS terbatas ke origin yang diatur di env.

Selesai bila: semua kode respons di SRS 7 bisa dipicu lewat curl, check.sh lulus.
Referensi: FR-10 sampai FR-19, FR-21, NFR-05, NFR-06, NFR-08.

## Sprint 3: API admin, ekspor, CLI (backend)

Tujuan: admin bisa masuk, melihat ringkasan, mengunduh data, dan mengelola status lewat CLI.

1. Login admin: sandi dicek terhadap hash bcrypt di env `ADMIN_PASSWORD_HASH`. Sesi berupa cookie HttpOnly, SameSite=Strict, ditandatangani HMAC dengan `SESSION_SECRET`, kedaluwarsa 8 jam. Rate limit percobaan login.
2. `GET /api/admin/summary`: jumlah peserta terkonfirmasi, daftar tunggu, registrasi batal.
3. `GET /api/admin/export?format=csv` dan `format=xlsx`: kolom dan urutan sesuai FR-32 dan FR-33, waktu dalam WIB, nama file sesuai FR-35.
4. CSV: UTF-8 dengan BOM, nilai yang diawali `=`, `+`, `-`, `@` diberi awalan apostrof.
5. CLI `backend/cmd/doomsday-admin`: `promote <kode>` (gagal kalau kursi tidak cukup), `cancel <kode>`, `rename <kode> <posisi>` (ubah nama, email, atau nama akun), `event set <field> <nilai>` dan `event show` untuk FR-36.
6. Semua perubahan lewat CLI mengisi `status_changed_at` dan mencetak ringkasan sebelum dan sesudah.

Selesai bila: file CSV dan XLSX terbuka benar di Excel, CLI jalan di database dev, check.sh lulus.
Referensi: FR-30 sampai FR-38.

## Sprint 4: Kerangka (frontend)

Tujuan: fondasi UI yang dipakai semua halaman.

1. Router: `/` (undangan) dan `/admin`. Parameter `?to=` dibaca, disanitasi, dibatasi 60 karakter.
2. Klien API di `frontend/src/api/` dengan tipe TypeScript untuk semua respons dan error.
3. Komponen dasar: Button (primary, outline, text), Field (label, input, pesan error, aria-describedby), Panel, Stamp, VoidSlit, Accordion, Skeleton.
4. Aturan global: focus ring 2px #E8E2D4 offset 2px lewat `:focus-visible`, tap target minimal 44px, padanan `:active` untuk semua hover, `prefers-reduced-motion`.
5. Composable `useEvent` dengan state memuat, berhasil, gagal.

Selesai bila: halaman contoh komponen tampil benar di 320px dan 1440px tanpa scroll horizontal, check.sh lulus.
Referensi: NFR-01 sampai NFR-03, SRS 11.

## Sprint 5: Halaman undangan (frontend)

Tujuan: alur pembuka sampai isi surat, sesuai layar Stitch.

1. Pembuka: varian umum dan varian `?to=`. Tombol "Buka surat panggilan" memicu animasi amplop sekitar 600ms, lalu fokus pindah ke judul isi surat. Tanpa animasi bila reduced motion.
2. Isi surat: dossier dengan nilai "Menyusul" untuk field kosong, link peta hanya muncul bila `map_url` terisi.
3. Hitung mundur WIB: teks "Jadwal tayang diumumkan menyusul." bila jadwal kosong, angka bila ada, teks acara sudah berlangsung bila lewat. Tidak diumumkan ke pembaca layar tiap detik.
4. Panel sisa kursi dari data server, berubah menjadi info daftar tunggu saat penuh.
5. FAQ dengan teks SRS 3.5, accordion bisa dioperasikan keyboard.
6. State pendaftaran ditutup dan gagal memuat (dengan "Coba lagi").
7. Layout desktop dua kolom (kolom kiri menempel saat scroll), bertumpuk di layar sempit.

Selesai bila: semua layar Stitch untuk bagian ini bisa dicapai di browser dengan data dev, check.sh lulus.
Referensi: FR-01 sampai FR-08, SRS 4.

## Sprint 6: Formulir dan kartu akses (frontend)

Tujuan: pendaftaran ujung ke ujung dari browser.

1. Pilihan Solo atau VIP, data pendaftar, tamu dinamis 1 sampai 3 (tombol tambah nonaktif di 3 tamu, hapus tamu dengan fokus yang tidak hilang).
2. Validasi di klien dengan aturan yang sama dengan backend, ringkasan error diumumkan, fokus ke field salah pertama.
3. Pesan error dari backend dipetakan ke field yang tepat (email terdaftar menunjuk baris yang bermasalah).
4. Mode daftar tunggu saat kursi habis, peringatan saat sisa kursi kurang dari jumlah rombongan.
5. Status mengirim: tombol nonaktif, input dikunci, idempotency key dibuat sekali per percobaan kirim.
6. Kartu akses terkonfirmasi dan daftar tunggu, tombol "Simpan kartu akses" (PNG) dan "Salin kode".
7. Kartu disimpan di localStorage dalam try/catch, dibaca lagi saat pendaftar kembali.

Selesai bila: pendaftaran Solo, VIP, dan daftar tunggu berhasil dari browser dan tercatat di database, check.sh lulus.
Referensi: FR-10 sampai FR-24.

## Sprint 7: Halaman admin (frontend)

Tujuan: admin mengelola data dari browser.

1. Form masuk dengan pesan "Sandi salah. Coba lagi."
2. Ringkasan dengan state memuat, kosong, gagal.
3. Tombol "Unduh Excel" dan "Unduh CSV".
4. Tombol keluar yang menghapus sesi.

Selesai bila: admin bisa masuk, melihat angka yang cocok dengan database, dan mengunduh dua format, check.sh lulus.
Referensi: FR-30, FR-31, FR-37.

## Sprint 8: Docker dan deploy LAN

Tujuan: aplikasi jalan di VM dan bisa dibuka dari perangkat lain di Wi-Fi kantor.

1. `backend/Dockerfile` multi-stage (build Go, image akhir distroless atau alpine, jalan sebagai user non-root). Image berisi binary `api` dan `doomsday-admin`.
2. `frontend/Dockerfile` multi-stage (build Vite, sajikan dengan Nginx).
3. `frontend/nginx.conf`: sajikan aset statis, teruskan `/api` ke container backend, kirim header `X-Forwarded-For`, batasi ukuran body, header keamanan dasar.
4. `deploy/docker-compose.yml`: service `doomsday-backend` tanpa port terbuka ke host, `doomsday-frontend` di port 8090, keduanya tersambung ke network Postgres yang sudah ada, `restart: unless-stopped`.
5. Env produksi di `deploy/.env` di VM (tidak masuk git), migrasi dijalankan sebagai langkah deploy.
6. Backup harian `pg_dump` database `nvm_doomsday` ke folder di VM.
7. Uji akses dari HP dan laptop lain di Wi-Fi kantor.

Selesai bila: `http://<IP VM>:8090` terbuka dari perangkat lain di LAN, pendaftaran tercatat, backup pertama terbentuk.
Referensi: NFR-04, NFR-05, NFR-07.

## Sprint 9: Pengujian menyeluruh

Tujuan: semua kebutuhan terbukti jalan sebelum link disebar.

1. Backend, unit test: validasi, generator kode, aturan kuota dan daftar tunggu, sanitasi CSV.
2. Backend, integration test dengan database uji: semua kode respons di SRS 7, idempotency, email unik, CLI admin.
3. Backend, uji balapan kuota: banyak pendaftaran serentak saat sisa kursi sedikit, hasilnya tidak boleh melebihi 200 peserta terkonfirmasi.
4. Frontend, Vitest untuk validasi formulir dan pemetaan error. Tambahkan `vitest run` ke bagian frontend di check.sh.
5. Uji manual per FR dengan checklist, termasuk browser bawaan WhatsApp dan Instagram.
6. Aksesibilitas: navigasi penuh dengan keyboard, focus terlihat, cek kontras ulang, reduced motion.
7. Responsif: 320px, 375px, 768px, 1024px, 1440px, tanpa scroll horizontal.
8. Delivery Gate AntiSlop, satu baris PASS atau FAIL per poin beserta buktinya.

Selesai bila: semua checklist lulus, tidak ada bentol terbuka berprioritas M.

## Sprint 10: Rilis publik (tertahan)

Tujuan: link bisa dibuka affiliator dari luar kantor.

1. Tentukan domain atau subdomain.
2. Arahkan lewat Cloudflare Tunnel ke `doomsday-frontend`, HTTPS dari Cloudflare.
3. Perbarui origin CORS di env backend dan atur cookie admin menjadi Secure.
4. Pertimbangkan membatasi `/admin` lewat Cloudflare Zero Trust.

Tertahan sampai: domain dipilih.

## Keputusan yang masih menunggu

| Keputusan | Mempengaruhi |
|---|---|
| "Nama akun" itu akun apa | Label dan validasi field (Sprint 2, 6) |
| Bioskop, kota, studio, jadwal | Isi `event_config`, bisa diisi lewat CLI kapan saja |
| Batas akhir pendaftaran | Perlu kolom deadline atau cukup tutup manual |
| Perlu identitas saat hari H | Teks FAQ dan kartu akses |
| Kontak admin untuk FAQ | Teks FAQ |
| Domain | Sprint 10 |
| Lama penyimpanan data setelah acara | Prosedur hapus data |
| Batas jumlah daftar tunggu | Aturan pendaftaran (saat ini tanpa batas) |
