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

1. Request ID acak per request (header `X-Request-ID`), log akses JSON tanpa IP, query string, atau body, format error seragam `{"error": {"code", "message", "fields"}}`, recovery panic yang menjawab JSON.
2. `GET /api/event`: detail acara, `is_open`, `seats_left`, `waitlisted_attendees`, `max_guests_vip`. Field kosong dikirim `null`. Header `Cache-Control: no-store`.
3. Validasi input di paket `registration`: tipe, jumlah peserta (Solo 1, VIP 2 sampai 4), nama 1 sampai 100 karakter, email valid maksimal 254, nama akun 1 sampai 50, trim spasi, email huruf kecil, tanpa email ganda dalam satu formulir, persetujuan wajib, honeypot `website` harus kosong.
4. Generator kode `NVM-DD-XXXXXX` dengan `crypto/rand` dari alfabet tanpa O, I, L, 0, 1, plus `NormalizeCode` untuk kode yang diketik manusia.
5. Transaksi pendaftaran: kunci baris event, cek idempotency key, cek `is_open`, cek batas tamu VIP yang berlaku, cek email aktif, hitung sisa kursi, tentukan `confirmed` atau `waitlisted` untuk seluruh rombongan, buat kode unik, simpan, baca hasil sebelum commit.
6. `POST /api/registrations`: 201 untuk pendaftaran baru, 200 untuk kiriman ulang dengan idempotency key yang sama, 409 untuk ditutup dan email terdaftar, 422 untuk isian salah, body maksimal 16 KB.
7. Rate limit per IP (10 beruntun, lalu 1 per 6 detik), `TRUSTED_PROXIES` dari env, wajib `Content-Type: application/json` (415 kalau bukan).

Selesai bila: semua kode respons di SRS 7 bisa dipicu lewat curl, check.sh lulus.
Referensi: FR-10 sampai FR-19, FR-21, NFR-05, NFR-06, NFR-08.

### Laporan

Status: selesai, 30 Oktober 2026.

Hasil uji curl:
- Solo baru 201, kiriman ulang key sama 200 dengan kode sama, email sama berhuruf kapital 409 `duplicate_email`.
- Isian salah 422 dengan pesan per kolom, VIP dengan satu tamu 201.
- Kapasitas diturunkan ke 3: VIP masuk daftar tunggu utuh dengan nomor urut 1, `seats_left` 0.
- `is_open` false: 409 `registration_closed`.
- Kiriman tanpa JSON 415. Kiriman beruntun ke-10 dan seterusnya 429.
- Data uji dihapus, `event_config` dikembalikan ke kapasitas 200 dan terbuka.

Perubahan dari rencana:
- CORS tidak dipasang. Frontend dan API selalu satu origin (proxy Vite di dev, Nginx di server), jadi tanpa header CORS browser sudah menolak situs lain membaca API. Kiriman lintas situs berjenis `text/plain` ditutup dengan mewajibkan `application/json`.
- Idempotency key dicek di dalam transaksi setelah kunci event didapat, bukan sebelumnya, supaya dua kiriman dengan key sama tidak sama-sama lolos. Langkah 6 jadi bagian dari langkah 5.
- Langkah 5 dipecah: logika transaksi di paket `registration`, handler HTTP terpisah.
- Batas tamu VIP dicek dua kali: batas mutlak 3 saat validasi, `max_guests_vip` yang berlaku di dalam transaksi.
- Batas rate dibuat longgar karena banyak pengguna HP berbagi satu IP publik dari operator.

Dipindah ke Sprint 8:
- Isi `TRUSTED_PROXIES` dengan subnet network Docker Nginx, lalu pastikan rate limit membaca IP asli pengunjung, bukan IP Nginx.

Dipindah ke Sprint 9:
- Uji balapan: banyak pendaftaran serentak saat sisa kursi sedikit tidak boleh melebihi kapasitas.
- Unit test validasi dan generator kode.


## Sprint 3: API admin, ekspor, CLI (backend)

Tujuan: admin bisa masuk, melihat ringkasan, mengunduh data, dan mengelola status lewat CLI.

1. Login admin: sandi dicek terhadap hash bcrypt (cost 12) di `ADMIN_PASSWORD_HASH`. Sesi berupa cookie `doomsday_admin` berisi waktu kedaluwarsa dan HMAC dari `SESSION_SECRET`, HttpOnly, SameSite=Strict, path `/api/admin`, berlaku 8 jam. `Secure` diatur lewat `COOKIE_SECURE`. Rate limit login 5 beruntun lalu 1 per 12 detik per IP. Endpoint: `POST /api/admin/login`, `POST /api/admin/logout`, `GET /api/admin/session`.
2. `GET /api/admin/summary`: kapasitas, peserta terkonfirmasi, peserta daftar tunggu, registrasi batal.
3. `GET /api/admin/export?format=csv` dan `format=xlsx`: kolom FR-32, urutan FR-33, waktu WIB, nama file FR-35. CSV memakai BOM UTF-8, CRLF, dan apostrof untuk sel berawalan `=`, `+`, `-`, `@`, tab, atau CR. XLSX dengan judul tebal dan terkunci.
4. CLI `doomsday-admin`: `hash-password`, `event show`, `event set <field> <nilai>`, `promote <kode>`, `cancel <kode>`, `rename <kode> <posisi> [--name] [--email] [--account]`. Setiap perubahan mencetak keadaan sebelum dan sesudah.

Selesai bila: file CSV dan XLSX terbuka benar di Excel, CLI jalan di database dev, check.sh lulus.
Referensi: FR-30 sampai FR-38.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:
- Login: tanpa cookie 401, sandi salah 401 `invalid_password`, sandi benar 204, cookie HttpOnly dengan path `/api/admin`.
- Ringkasan: 401 tanpa cookie, 200 dengan `Cache-Control: no-store`, angka cocok dengan database.
- Ekspor: format salah 422, nama file dengan jam WIB, BOM ada, urutan terkonfirmasi lalu daftar tunggu lalu batal, nomor urut daftar tunggu ikut maju saat registrasi di depannya dibatalkan, `=1+1` dan `@uji2` diberi apostrof di CSV, XLSX menampilkannya sebagai teks. File kosong berisi judul kolom saja.
- CLI event: field opsional bisa dikosongkan dengan `-`, jam dibaca sebagai WIB, jam kumpul setelah jam tayang ditolak CHECK database, link non-https dan kapasitas di luar 1 sampai 10000 ditolak.
- CLI status: sepuluh skenario promote, cancel, rename lolos, termasuk kode huruf kecil, rombongan yang tidak muat, email yang sudah dipakai, dan posisi kosong.
- Email dari registrasi yang dibatalkan bisa dipakai mendaftar lagi (trigger `is_active` dari migrasi 0002 terbukti).

Perubahan dari rencana:
- Langkah 3 dan 4 digabung: pengamanan CSV ditulis bersama kode yang menulis CSV.
- Langkah 5 dan 6 digabung: ringkasan sebelum dan sesudah langsung jadi bagian setiap perintah CLI.
- `COOKIE_SECURE` dibuat bisa diatur karena Sprint 8 masih HTTP lewat LAN. Wajib `true` di Sprint 10.
- CLI login sebagai `doomsday_app`, cukup `DATABASE_URL`, tidak butuh hash sandi atau session secret.
- `promote` dan `cancel` memakai kunci event yang sama dengan pendaftaran, supaya tidak balapan dengan pendaftar baru. Promosi dari daftar tunggu tetap manual, CLI hanya menyebut registrasi teratas.
- Zona waktu WIB didefinisikan sekali di `internal/timezone` sebagai offset tetap UTC+7.
- `GET /api/event` mengirim waktu dalam UTC supaya keluaran tidak bergantung pada zona waktu mesin. Perubahan ini ikut masuk di commit `4e6e825` (CLI event), bukan commit tersendiri.
- "Waktu ekspor terakhir" di FR-37 ditunda: butuh tabel baru untuk mencatat ekspor dan tidak ada di desain yang sudah direview.

Keputusan untuk Sprint 6:
- Nomor urut daftar tunggu bisa maju setelah pendaftar mendapat kartu. Kartu akses perlu menulis "nomor urut saat mendaftar", atau frontend perlu cara mengambil posisi terbaru.

Dipindah ke Sprint 8:
- Cara menjalankan CLI di server: `docker compose exec doomsday-backend doomsday-admin <perintah>`.

Dipindah ke Sprint 9:
- Buka file CSV ekspor di Excel dengan pengaturan regional Indonesia. Kalau semua isi menumpuk di satu kolom, putuskan antara pemisah titik koma atau XLSX sebagai format utama untuk Excel.
- Unit test `adminauth` (token kedaluwarsa, tanda tangan salah) dan `export` (urutan, nomor urut, sanitasi CSV).

## Sprint 4: Kerangka (frontend)

Tujuan: fondasi UI yang dipakai semua halaman.

1. Router: `/` (undangan), `/admin` (dimuat terpisah), halaman tidak ditemukan, dan `/_komponen` khusus development. Judul tab per halaman. Parameter `?to=` lewat `useRecipient`: buang karakter kontrol dan penanda arah teks, rapikan spasi, potong 60 karakter Unicode, `?to=` ganda dianggap kosong.
2. Klien API di `frontend/src/api/`: tipe untuk semua respons dan error, hasil selalu berbentuk `{ok, status, data}` atau `{ok, status, error}`. Batas waktu 15 detik, putus jaringan jadi `network_error`, respons bukan JSON jadi `internal_error`.
3. Komponen `BaseButton` (primary, outline, text, memuat, nonaktif, lebar penuh), `BaseField` (label, petunjuk, error, `aria-describedby`, `aria-invalid`, fungsi `focus`), `BasePanel`.
4. Komponen `BaseStamp`, `VoidSlit`, `BaseAccordion` (Enter, Space, Escape, `aria-expanded`, `aria-controls`), `BaseSkeleton`.
5. Composable `useEvent` dengan state memuat, siap, gagal, plus `reload` dan `refresh` diam-diam.
6. Halaman `/_komponen` dicek di 320px dan 1440px.

Selesai bila: halaman contoh komponen tampil benar di 320px dan 1440px tanpa scroll horizontal, check.sh lulus.
Referensi: NFR-01 sampai NFR-03, SRS 11.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:
- `?to=` dengan spasi berlebih, tag HTML, penanda arah teks, 80 karakter, dan parameter ganda tampil sesuai aturan sanitasi.
- `/admin` dimuat sebagai file terpisah, halaman tidak ditemukan bisa kembali ke undangan dengan keyboard.
- Klien API dari console: event 200, ringkasan tanpa login 401, sandi salah 401, pendaftaran kosong 422 dengan pesan per kolom.
- Komponen: focus ring terlihat di semua kontrol, kontrol nonaktif dilewati Tab, label memindahkan fokus ke kolomnya, pesan error terbaca sebagai deskripsi kolom, Escape menutup jawaban accordion dan fokus tetap di pertanyaannya, panah accordion tidak berputar saat gerak dikurangi.
- `useEvent`: memuat, gagal saat backend mati, pulih dengan "Coba lagi", refresh diam-diam mempertahankan data saat gagal dan memperbarui angka saat berhasil.
- Tidak ada scroll horizontal di 320px dan 1440px.
- Hasil build produksi tidak memuat halaman `/_komponen` maupun teksnya.

Keputusan:
- Nama komponen diawali `Base` karena aturan ESLint Vue mewajibkan nama komponen dua kata.
- Halaman contoh komponen dibuat sejak langkah 3, dan hanya ada di development lewat `import.meta.env.DEV`.
- Seluruh situs `noindex, nofollow`: undangan disebar lewat link dan halaman admin tidak boleh terindeks.
- Skeleton tanpa animasi berdenyut, sesuai aturan tidak ada animasi berulang tanpa pemicu pengguna.
- Accordion memakai tombol, bukan `<details>`, karena `<details>` tidak bisa ditutup dengan Escape.
- Latar kolom isian memakai `void` supaya tetap terlihat di dalam panel.
- Kapital dan spasi renggang hanya di

## Sprint 5: Halaman undangan (frontend)

Tujuan: alur pembuka sampai isi undangan dengan identitas visual NVM.

0. Identitas NVM: palet dari tema gelap Shopify NVM (`#08090b`, `#0d0f12`, `#e8ebef`, krem `#e8e3d5`), font D-DIN dan JetBrains Mono, tombol utama krem. Oranye dan font Archivo serta IBM Plex dihapus.
0b. Aset brand dari NVM: logo, favicon, D-DIN woff2 dengan lisensi OFL.
1. Format tanggal dan jam WIB (`Asia/Jakarta`), isi FAQ yang membaca kapasitas dan batas tamu dari data acara.
2. Amplop pembuka dengan logo NVM, stempel, dan celah Void gelap di bibir amplop. Animasi sekitar 600 ms, fokus pindah ke judul isi undangan, langsung tanpa animasi bila gerak dikurangi, hanya sekali per sesi.
3. Lembar spesifikasi acara bergaya label produk NVM, menggantikan dossier. "Menyusul" untuk data kosong, link peta hanya bila ada.
4. Hitung mundur ke jam tayang dengan bingkai celah Void, tiga keadaan: jadwal kosong, berjalan, sudah dimulai.
5. Panel sisa kursi, kursi penuh dengan jumlah daftar tunggu, dan pendaftaran ditutup. Diperbarui diam-diam saat tab kembali terlihat.
6. FAQ, layout satu kolom di bawah 1024px dan dua kolom di atasnya.

Selesai bila: alur undangan bisa dicapai di browser dengan data dev, check.sh lulus.
Referensi: FR-01 sampai FR-08, SRS 4, SRS 11.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:
- Komponen tampil dengan palet dan font NVM, font dimuat dari server sendiri.
- Lembar spesifikasi: "Menyusul" pudar dan kalimat penjelasan muncul saat data kosong, tanggal "Jumat, 18 Desember 2026" dan jam "18.15 WIB" serta "19.00 WIB" benar pada data contoh, link peta hanya muncul bila ada URL.
- Hitung mundur ke `2026-12-16 19:00` menunjukkan 42 hari 1 jam 39 menit pada 4 November 17.21, sesuai perhitungan.
- FAQ membaca "200 kursi" dan "maksimal 3 orang" dari data acara.
- Layout dua kolom di layar lebar, satu kolom di HP.

Perubahan dari rencana:
- Arah desain diganti dari "berkas rahasia" ke bahasa produk NVM, karena desain awal tidak memakai aset brand NVM sama sekali dan pola berkas rahasia terlalu umum. Dossier jadi lembar spesifikasi, kartu akses akan jadi hangtag di Sprint 6. Manajer setuju.
- Desain baru langsung dikoding, tanpa memperbarui Stitch.
- Label mono kapital dengan spasi renggang dipakai di lembar spesifikasi, mengikuti gaya label di tema Shopify NVM. Sebelumnya hanya untuk stempel.
- Celah Void di amplop berupa garis gelap, karena garis krem tidak terlihat di atas kertas krem.
- Kolom kiri desktop tidak menempel saat digulir: tingginya melebihi layar laptop, dan kolom menempel yang lebih tinggi dari layar menyembunyikan bagian bawahnya.
- Tombol "Konfirmasi kehadiran" belum dipasang karena formulirnya baru ada di Sprint 6.
- Judul memakai `text-wrap: balance` supaya tidak ada satu kata menggantung sendirian.

Bentol yang ditemui:
- `git rm` file TTF ikut menghapus folder `public/fonts`, sehingga commit `2818da4` tidak berisi font woff2 dan halaman sempat memakai font cadangan. Diperbaiki di commit `0f946d6`.
- GitHub dua kali menjawab Internal Server Error saat push. Commit tetap aman di laptop, push berhasil saat diulang.
- Workspace asisten tidak bisa membuka arsip RAR, aset dikirim ulang sebagai ZIP.

Catatan data:
- `show_at` di database diisi perkiraan `2026-12-16 19:00`, berdasarkan pola film Avengers sebelumnya yang tayang di Indonesia hari Rabu sebelum rilis global. Tanggal resmi Indonesia belum diumumkan per 4 November 2026.
- Database yang dipakai saat development adalah database yang sama dengan yang akan dipakai server.

Dipindah ke Sprint 6:
- Tombol "Konfirmasi kehadiran" di kolom kiri yang menggulir ke formulir.

Dipindah ke Sprint 8:
- Sebelum link disebar: isi jadwal resmi atau kosongkan dengan `event set show_at -`, dan pastikan tidak ada data uji tersisa.
- Pertimbangkan database terpisah untuk development (misalnya `nvm_doomsday_dev`), supaya uji coba tidak menyentuh data asli setelah pendaftaran dibuka.

Dipindah ke Sprint 9:
- Uji animasi amplop, fokus setelah dibuka, ingatan per sesi, dan versi tanpa animasi.
- Uji panel kursi saat sebagian terisi, penuh, ada daftar tunggu, dan ditutup, termasuk pembaruan saat tab kembali terlihat.

Dokumen yang perlu diperbarui:
- SRS bagian 2.2 dan 11 masih menjelaskan konsep berkas rahasia, palet oranye, serta font Archivo dan IBM Plex. Perlu SRS v0.3.

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
