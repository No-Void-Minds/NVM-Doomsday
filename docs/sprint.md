# Sprint nvm-doomsday

Rencana kerja landing page undangan nonton bareng Avengers: Doomsday untuk No Void Minds.

- Acuan kebutuhan: SRS v0.2 (`docs/SRS-NVM-Doomsday-v0.2.docx`). Kode FR dan NFR di bawah merujuk ke dokumen itu. SRS v0.3 ditulis setelah semua sprint selesai.
- Acuan tampilan: proyek Stitch "Void screening invitation system", lalu identitas NVM sejak Sprint 5.

## Cara kerja

- Satu langkah dikerjakan sampai jalan, baru lanjut ke langkah berikutnya.
- Setiap langkah selesai: `./scripts/check.sh && git commit && git push`.
- `check.sh` menjaga format, lint, type check, staticcheck, unit test Go, Vitest, dan em dash. Integration test dan E2E butuh database uji, jadi dijalankan terpisah lewat `scripts/test-integration.sh` dan `scripts/test-e2e.sh`.
- Test boleh ditulis dan dijalankan Claude Code, tapi commit dan push selalu dikerjakan sendiri.
- Status: `belum`, `jalan`, `selesai`, `tertahan` (menunggu keputusan).

## Ringkasan status

| Sprint | Isi | Status | Tanggal |
|---|---|---|---|
| 0 | Fondasi repo, tooling, check.sh | selesai | 30 Okt 2026 |
| 1 | Database dan migrasi | selesai | 30 Okt 2026 |
| 2 | API publik: event dan pendaftaran | selesai | 30 Okt 2026 |
| 3 | API admin, ekspor, CLI admin | selesai | 4 Nov 2026 |
| 4 | Kerangka dan komponen dasar frontend | selesai | 4 Nov 2026 |
| 5 | Halaman undangan | selesai | 4 Nov 2026 |
| 6 | Formulir dan kartu akses | selesai | 5 Nov 2026 |
| 7 | Halaman admin | selesai | 5 Nov 2026 |
| 8 | Docker dan deploy LAN | selesai | 5 Nov 2026 |
| 9 | Revisi: cek nama akun VIP dan aset Doomsday | selesai | 5 Nov 2026 |
| 10 | Pengujian menyeluruh | selesai | 5 Nov 2026 |
| 11 | Rilis publik lewat Cloudflare Tunnel | tertahan | |

## Stack

| Bagian | Isi |
|---|---|
| Backend (`backend/`) | Go 1.27, Gin, pgx/v5, sqlc 1.31.1, golang-migrate, excelize (XLSX), encoding/csv, bcrypt |
| Frontend (`frontend/`) | Vue 3.5, TypeScript, Vite 8, Tailwind v4, Vue Router, html-to-image. Tanpa Pinia, state cukup di composable |
| Font | D-DIN (woff2 di `public/fonts`, lisensi OFL) dan JetBrains Mono (@fontsource), semua di-host sendiri |
| Database | PostgreSQL 18 di container `nvm-internal-postgres` di VM `nvm-server`. Database `nvm_doomsday`, role `doomsday_migrator` (pemilik, hanya untuk migrasi) dan `doomsday_app` (backend dan CLI, hanya SELECT, INSERT, UPDATE) |
| Test | Go `testing` (unit, integration dengan tag `integration`), staticcheck v0.8.2, Vitest 5.1.0, Playwright 1.64.1 dengan Chromium, @axe-core/playwright 4.14.0 |
| Deploy | Docker Compose di VM, akses LAN `http://192.168.68.144:8090` |

## Struktur repo

```
nvm-doomsday/
  backend/
    cmd/
      server/            API
      doomsday-admin/    CLI admin
    internal/
      adminauth/         sesi admin HMAC
      config/
      db/
        queries/         file .sql untuk sqlc
        sqlc/            hasil generate, jangan diedit manual
      export/            CSV dan XLSX
      httpapi/
      registration/      pendaftaran, cek nama akun VIP
      testdb/            helper database uji, menolak database yang tidak berakhiran _test
      timezone/          WIB sebagai offset tetap
      vip/               pengelolaan daftar akun VIP
    migrations/
    Dockerfile
    sqlc.yaml
    .env.example
  frontend/
    public/              brand, fonts, favicon
      doomsday/          aset Doomsday di sisi kiri dan kanan layar lebar
    src/
      api/
      components/
        admin/
        invite/
      composables/
      content/           isi FAQ dan daftar aset
      lib/               tanggal, validasi, uuid, clipboard
      pages/
      router/
      styles/
    e2e/                 test Playwright
    Dockerfile
    nginx.conf
  deploy/
    docker-compose.yml
    .env.example
    backup.sh
  docs/
  scripts/
    check.sh
    dev-db-tunnel.sh
    test-env.sh
    test-integration.sh
    test-e2e.sh
  testdata/              kasus uji bersama Go dan Vitest
```

Backend dan frontend tidak saling import. Satu-satunya kontrak di antara keduanya adalah API di SRS bagian 7.

## Port

| Layanan | Dev (PC) | Server (VM) |
|---|---|---|
| Backend | 127.0.0.1:8081 | 8080 di dalam container, tidak dibuka ke host |
| Frontend | 127.0.0.1:5174 (Vite, proxy `/api` ke 8081) | 0.0.0.0:8090 (Nginx) |
| Postgres | 127.0.0.1:15432 lewat SSH tunnel | `nvm-internal-postgres:5432` di network `nvm-internal_default` |
| Database uji | `nvm_doomsday_test` lewat tunnel yang sama | tidak dipakai server |

Port 8081 dan 5174 dipilih agar tidak bentrok dengan NVM-Internal (8080, 5173). Port 8092 di VM dipakai compro.

---

## Sprint 0: Fondasi

Tujuan: repo siap dikerjakan, `check.sh` hijau dengan proyek kosong.

1. Repo `nvm-doomsday` dengan folder `backend/`, `frontend/`, `deploy/`, `scripts/`, `docs/`, plus `.gitignore` dan `.editorconfig` di root.
2. Backend: `go mod init`, endpoint `GET /api/health`, `.env.example`.
3. Frontend: aplikasi Vite Vue TS, Tailwind v4, ESLint, Prettier, vue-tsc, proxy `/api` ke backend.
4. Token warna, font, radius dari SRS 11.3 sampai 11.5 sebagai `@theme` Tailwind.
5. `scripts/check.sh`: backend (gofmt, go vet, go test) dari `backend/`, frontend (vue-tsc, eslint, prettier) dari `frontend/`. Berhenti di error pertama dan menyebut bagian yang gagal.
6. `scripts/dev-db-tunnel.sh` untuk SSH tunnel ke Postgres di VM (port lokal 15432).

Selesai bila: `check.sh` lulus, backend dan frontend jalan bersamaan, frontend memanggil `/api/health` lewat proxy dan memakai token.

### Laporan

Status: selesai, 30 Oktober 2026.

Hasil:
- `check.sh` lulus untuk backend dan frontend.
- Frontend memanggil `/api/health` lewat proxy Vite dengan state memuat, berhasil, dan gagal.
- Token terpasang di `frontend/src/styles/tokens.css`, font di-host sendiri.
- Tunnel menyambung ke Postgres di VM lewat port 15432.

Perubahan dari rencana:
- Struktur mengikuti NVM-Internal: `cmd/server`, `internal/db/queries` dan `internal/db/sqlc`, `internal/httpapi`, `composables/` sebagai pengganti `stores/`. Tidak ada `docker-compose.dev.yml` karena database dev memakai Postgres di VM.
- Frontend dibuat dengan create-vue (TypeScript, Router, ESLint, Prettier).
- `.gitattributes` (akhir baris LF), karena kode ditulis di Windows dan dijalankan di Linux.
- `.vscode/extensions.json` dan `.vscode/settings.json` supaya editor menambahkan newline di akhir file dan memformat saat simpan.
- `check.sh` ditambah cek em dash (aturan AntiSlop).
- Izin eksekusi script dicatat di Git dengan `git add --chmod=+x`.

Bentol:
- Prettier dan gofmt menolak file tanpa newline di akhir, karena editor tidak membaca `.editorconfig`. Diselesaikan dengan `.vscode/settings.json`.
- Login Postgres sempat memakai user Linux `nvmadmin`. User Postgres berbeda dari user SSH VM.

---

## Sprint 1: Database

Tujuan: skema sesuai SRS bagian 6, bisa dimigrasi naik dan turun.

1. Di VM: database `nvm_doomsday`, role `doomsday_migrator` dan `doomsday_app`, izin CONNECT dicabut dari PUBLIC di `nvm_doomsday`, `nvm_internal`, dan `nvm_internal_test`.
2. pgxpool (maksimal 10 koneksi), `/api/health` ikut mengecek database dan menjawab 503 saat database putus.
3. Migrasi 0001: enum `registration_type`, `registration_status`, `attendee_role`, tabel `event_config`, `registrations`, `attendees` dengan CHECK untuk kode, email huruf kecil, dan posisi peserta.
4. Migrasi 0002: index unik email peserta aktif, index `registrations(status, created_at)`, trigger untuk `status_changed_at` dan `attendees.is_active`.
5. Migrasi 0003: seed `event_config` (capacity 200, max_guests_vip 3, lokasi dan jadwal NULL).
6. sqlc untuk acara, pendaftaran, ringkasan, dan ekspor. `check.sh` menjalankan `sqlc diff`.
7. Uji migrate down dan up penuh.

Selesai bila: `migrate down -all` dan `migrate up` bersih, izin dan seed terbentuk ulang, `sqlc generate` tanpa error, `check.sh` lulus.
Referensi: SRS 6.1 sampai 6.3, NFR-06.

### Laporan

Status: selesai, 30 Oktober 2026.

Perubahan dari rencana:
- Dua role database, bukan satu. `doomsday_app` tidak punya DDL dan DELETE, supaya celah di backend tidak bisa menghapus tabel atau data peserta.
- Kolom `attendees.is_active` ditambahkan, karena index unik email untuk peserta aktif tidak bisa membaca kolom dari tabel `registrations`. Nilainya dijaga trigger.
- Tidak ada index terpisah untuk `attendees(registration_id)`, karena sudah tercakup `UNIQUE (registration_id, position)`.
- Migrasi yang sudah dijalankan tidak pernah diedit. Perubahan selalu jadi migrasi baru.
- `created_at` registrasi memakai `clock_timestamp()` supaya urutan daftar tunggu sama dengan urutan diproses.

Bentol:
- Query hak akses sempat dijalankan di database `postgres` karena koneksi TablePlus belum pindah database. Sudah dikembalikan. Pelajaran: cek nama database di bar atas TablePlus sebelum menjalankan SQL yang mengubah izin.
- Sandi role tidak cocok dan nama database ikut spasi saat ditempel. Diselesaikan dengan `ALTER ROLE ... PASSWORD` dan mengetik ulang.

Catatan di luar repo ini:
- Backend NVM-Internal login sebagai `nvm_internal`, yang superuser. Sebaiknya NVM-Internal punya role aplikasi sendiri.
- Riwayat query TablePlus menyimpan `CREATE ROLE` dan `ALTER ROLE` beserta sandinya. Hapus dari tab History.

---

## Sprint 2: API publik

Tujuan: dua endpoint publik bekerja sesuai SRS bagian 7.

1. Request ID per request (`X-Request-ID`), log akses JSON tanpa IP, query string, atau body, format error `{"error": {"code", "message", "fields"}}`, recovery panic yang menjawab JSON.
2. `GET /api/event`: detail acara, `is_open`, `seats_left`, `waitlisted_attendees`, `max_guests_vip`, field kosong `null`, `Cache-Control: no-store`.
3. Validasi di paket `registration`: tipe, jumlah peserta (Solo 1, VIP 2 sampai 4), nama 1 sampai 100 karakter, email valid maksimal 254, nama akun 1 sampai 50, trim, email huruf kecil, tanpa email ganda dalam satu formulir, persetujuan wajib, honeypot `website` kosong.
4. Kode `NVM-DD-XXXXXX` dengan `crypto/rand` dari alfabet tanpa O, I, L, 0, 1, plus `NormalizeCode`.
5. Transaksi: kunci baris event, cek idempotency key, `is_open`, batas tamu VIP, email aktif, sisa kursi, status seluruh rombongan, kode unik, simpan.
6. `POST /api/registrations`: 201 baru, 200 kiriman ulang dengan key sama, 409 ditutup dan email terdaftar, 422 isian salah, body maksimal 16 KB.
7. Rate limit per IP (10 beruntun, lalu 1 per 6 detik), `TRUSTED_PROXIES` dari env, wajib `Content-Type: application/json` (415).

Selesai bila: semua kode respons di SRS 7 bisa dipicu lewat curl, `check.sh` lulus.
Referensi: FR-10 sampai FR-19, FR-21, NFR-05, NFR-06, NFR-08.

### Laporan

Status: selesai, 30 Oktober 2026.

Hasil uji curl:

| Uji | Hasil |
|---|---|
| Solo baru | 201 |
| Kiriman ulang dengan key sama | 200, kode sama |
| Email sama berhuruf kapital | 409 `duplicate_email` |
| Isian salah | 422 dengan pesan per kolom |
| VIP satu tamu | 201 |
| Kapasitas 3, VIP datang | masuk daftar tunggu utuh, nomor urut 1, `seats_left` 0 |
| `is_open` false | 409 `registration_closed` |
| Bukan JSON | 415 |
| Kiriman beruntun ke-10 dan seterusnya | 429 |

Data uji dihapus, `event_config` dikembalikan ke kapasitas 200 dan terbuka.

Perubahan dari rencana:
- CORS tidak dipasang. Frontend dan API selalu satu origin (proxy Vite di dev, Nginx di server). Kiriman lintas situs berjenis `text/plain` ditutup dengan mewajibkan `application/json`.
- Idempotency key dicek di dalam transaksi setelah kunci event didapat, supaya dua kiriman dengan key sama tidak sama-sama lolos. Langkah 6 jadi bagian dari langkah 5.
- Logika transaksi di paket `registration`, handler HTTP terpisah.
- Batas tamu VIP dicek dua kali: batas mutlak 3 saat validasi, `max_guests_vip` yang berlaku di dalam transaksi.
- Rate limit dibuat longgar karena banyak pengguna HP berbagi satu IP publik dari operator.

---

## Sprint 3: API admin, ekspor, CLI

Tujuan: admin bisa masuk, melihat ringkasan, mengunduh data, dan mengelola status lewat CLI.

1. Login: bcrypt cost 12 dari `ADMIN_PASSWORD_HASH`. Cookie `doomsday_admin` berisi kedaluwarsa dan HMAC dari `SESSION_SECRET`, HttpOnly, SameSite=Strict, path `/api/admin`, 8 jam, `Secure` lewat `COOKIE_SECURE`. Rate limit 5 beruntun lalu 1 per 12 detik. Endpoint `POST /api/admin/login`, `POST /api/admin/logout`, `GET /api/admin/session`.
2. `GET /api/admin/summary`: kapasitas, terkonfirmasi, daftar tunggu, batal.
3. `GET /api/admin/export?format=csv|xlsx`: kolom FR-32, urutan FR-33, waktu WIB, nama file FR-35. CSV dengan BOM UTF-8, CRLF, apostrof untuk sel berawalan `=`, `+`, `-`, `@`, tab, atau CR. XLSX dengan judul tebal dan terkunci.
4. CLI `doomsday-admin`: `hash-password`, `event show`, `event set <field> <nilai>`, `promote <kode>`, `cancel <kode>`, `rename <kode> <posisi> [--name] [--email] [--account]`, masing-masing mencetak keadaan sebelum dan sesudah.

Selesai bila: CSV dan XLSX terbuka benar di Excel, CLI jalan di database dev, `check.sh` lulus.
Referensi: FR-30 sampai FR-38.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:

| Bagian | Hasil |
|---|---|
| Login | tanpa cookie 401, sandi salah 401 `invalid_password`, sandi benar 204, cookie HttpOnly dengan path `/api/admin` |
| Ringkasan | 401 tanpa cookie, 200 dengan `no-store`, angka cocok dengan database |
| Ekspor | format salah 422, nama file dengan jam WIB, BOM ada, urutan terkonfirmasi lalu daftar tunggu lalu batal, nomor urut ikut maju saat registrasi di depannya batal, `=1+1` dan `@uji2` diberi apostrof di CSV dan tampil sebagai teks di XLSX, file kosong berisi judul kolom |
| CLI event | `-` mengosongkan field opsional, jam dibaca WIB, jam kumpul setelah jam tayang ditolak CHECK, link non-https dan kapasitas di luar 1 sampai 10000 ditolak |
| CLI status | sepuluh skenario promote, cancel, rename lolos, termasuk kode huruf kecil, rombongan tidak muat, email terpakai, posisi kosong |
| Trigger | email dari registrasi batal bisa dipakai mendaftar lagi |

Perubahan dari rencana:
- Pengamanan CSV ditulis bersama kode penulis CSV.
- Ringkasan sebelum dan sesudah langsung jadi bagian setiap perintah CLI.
- `COOKIE_SECURE` bisa diatur karena Sprint 8 masih HTTP lewat LAN. Wajib `true` di Sprint 11.
- CLI login sebagai `doomsday_app`, cukup `DATABASE_URL`.
- `promote` dan `cancel` memakai kunci event yang sama dengan pendaftaran. Promosi dari daftar tunggu tetap manual, CLI hanya menyebut registrasi teratas.
- WIB didefinisikan sekali di `internal/timezone` sebagai offset tetap UTC+7.
- `GET /api/event` mengirim waktu dalam UTC supaya tidak bergantung pada zona waktu mesin (ikut di commit `4e6e825`).
- "Waktu ekspor terakhir" (FR-37) ditunda karena butuh tabel baru dan tidak ada di desain yang direview.

---

## Sprint 4: Kerangka frontend

Tujuan: fondasi UI yang dipakai semua halaman.

1. Router: `/`, `/admin` (dimuat terpisah), halaman tidak ditemukan, `/_komponen` khusus development, judul tab per halaman. `?to=` lewat `useRecipient`: buang karakter kontrol dan penanda arah teks, rapikan spasi, potong 60 karakter, parameter ganda dianggap kosong.
2. Klien API: tipe semua respons, hasil `{ok, status, data}` atau `{ok, status, error}`, batas waktu 15 detik, putus jaringan jadi `network_error`, respons bukan JSON jadi `internal_error`.
3. `BaseButton`, `BaseField`, `BasePanel`.
4. `BaseStamp`, `VoidSlit`, `BaseAccordion` (Enter, Space, Escape), `BaseSkeleton`.
5. `useEvent` dengan memuat, siap, gagal, `reload`, dan `refresh` diam-diam.
6. `/_komponen` dicek di 320px dan 1440px.

Selesai bila: halaman contoh komponen benar di 320px dan 1440px tanpa scroll horizontal, `check.sh` lulus.
Referensi: NFR-01 sampai NFR-03, SRS 11.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:
- `?to=` dengan spasi berlebih, tag HTML, penanda arah teks, 80 karakter, dan parameter ganda tampil sesuai aturan.
- `/admin` dimuat sebagai file terpisah. Halaman tidak ditemukan bisa kembali ke undangan dengan keyboard.
- Klien API: event 200, ringkasan tanpa login 401, sandi salah 401, pendaftaran kosong 422 per kolom.
- Komponen: focus ring terlihat, kontrol nonaktif dilewati Tab, label memindahkan fokus, error terbaca sebagai deskripsi kolom, Escape menutup accordion dan fokus tetap di pertanyaan, panah tidak berputar saat gerak dikurangi.
- `useEvent`: gagal saat backend mati, pulih dengan "Coba lagi", refresh diam-diam mempertahankan data saat gagal.
- Tanpa scroll horizontal di 320px dan 1440px.
- Build produksi tidak memuat `/_komponen` maupun teksnya.

Keputusan:
- Nama komponen diawali `Base` karena ESLint Vue mewajibkan nama dua kata.
- Seluruh situs `noindex, nofollow`.
- Skeleton tanpa animasi berdenyut.
- Accordion memakai tombol, bukan `<details>`, karena `<details>` tidak bisa ditutup dengan Escape.
- Latar kolom isian memakai `void` supaya tetap terlihat di dalam panel.
- Kapital dengan spasi renggang hanya di stempel (diperluas ke label spesifikasi di Sprint 5).

---

## Sprint 5: Halaman undangan

Tujuan: alur pembuka sampai isi undangan dengan identitas visual NVM.

1. Identitas NVM: palet dari tema gelap Shopify NVM (`#08090b`, `#0d0f12`, `#e8ebef`, krem `#e8e3d5`), D-DIN dan JetBrains Mono, tombol utama krem. Oranye, Archivo, dan IBM Plex dihapus. Logo, favicon, dan D-DIN dari aset NVM.
2. Tanggal dan jam WIB (`Asia/Jakarta`), FAQ yang membaca kapasitas dan batas tamu dari data acara.
3. Amplop pembuka dengan logo NVM, stempel, dan celah Void gelap. Animasi sekitar 600 ms, fokus pindah ke judul, tanpa animasi bila gerak dikurangi, sekali per sesi.
4. Lembar spesifikasi bergaya label produk NVM, "Menyusul" untuk data kosong, link peta hanya bila ada.
5. Hitung mundur ke jam tayang dengan bingkai celah Void: jadwal kosong, berjalan, sudah dimulai.
6. Panel sisa kursi, penuh dengan jumlah daftar tunggu, dan ditutup. Diperbarui saat tab kembali terlihat.
7. FAQ, satu kolom di bawah 1024px dan dua kolom di atasnya.

Selesai bila: alur undangan bisa dicapai dengan data dev, `check.sh` lulus.
Referensi: FR-01 sampai FR-08, SRS 4, SRS 11.

### Laporan

Status: selesai, 4 November 2026.

Hasil uji:
- Palet dan font NVM tampil, font dimuat dari server sendiri.
- Lembar spesifikasi: "Menyusul" pudar dengan kalimat penjelasan, tanggal "Jumat, 18 Desember 2026" dan jam "18.15 WIB" serta "19.00 WIB" benar pada data contoh, link peta hanya muncul bila ada URL.
- Hitung mundur ke `2026-12-16 19:00` menunjukkan 42 hari 1 jam 39 menit pada 4 November 17.21, sesuai perhitungan.
- FAQ membaca "200 kursi" dan "maksimal 3 orang" dari data acara.
- Dua kolom di layar lebar, satu kolom di HP.

Perubahan dari rencana:
- Arah desain diganti dari "berkas rahasia" ke bahasa produk NVM, karena desain awal tidak memakai aset brand NVM dan pola berkas rahasia terlalu umum. Dossier jadi lembar spesifikasi, kartu akses jadi hangtag. Manajer setuju.
- Desain baru langsung dikoding, tanpa memperbarui Stitch.
- Label mono kapital dengan spasi renggang dipakai di lembar spesifikasi, mengikuti label di tema Shopify NVM.
- Celah Void di amplop berupa garis gelap, karena garis krem tidak terlihat di atas kertas krem.
- Kolom kiri desktop tidak menempel saat digulir, karena tingginya melebihi layar laptop.
- Judul memakai `text-wrap: balance`.

Bentol:
- `git rm` file TTF ikut menghapus folder `public/fonts`, sehingga commit `2818da4` tanpa woff2. Diperbaiki di `0f946d6`.
- GitHub dua kali menjawab Internal Server Error saat push. Berhasil saat diulang.
- Workspace asisten tidak bisa membuka RAR, aset dikirim ulang sebagai ZIP.

Catatan data:
- `show_at` diisi perkiraan `2026-12-16 19:00`, berdasarkan pola film Avengers sebelumnya yang tayang di Indonesia hari Rabu sebelum rilis global. Tanggal resmi Indonesia belum diumumkan per 4 November 2026. Wajib diganti sebelum link disebar.

Dokumen yang perlu diperbarui:
- SRS bagian 2.2 dan 11 masih menjelaskan konsep berkas rahasia, palet oranye, Archivo, dan IBM Plex. Diperbarui di SRS v0.3.

---

## Sprint 6: Formulir dan kartu akses

Tujuan: pendaftaran ujung ke ujung dari browser.

1. Formulir: Solo atau VIP sebagai radio, data pendaftar, tamu 1 sampai `max_guests_vip`, tamu tidak terhapus saat pindah ke Solo, fokus ke tombol tambah tamu setelah tamu dihapus.
2. Validasi di browser dengan aturan dan pesan yang sama dengan backend, ringkasan jumlah isian salah, fokus ke kolom salah pertama.
3. Kirim ke API: formulir terkunci, idempotency key sekali per isi formulir, error server ditempel ke kolomnya.
4. Mode daftar tunggu saat kursi habis, peringatan saat rombongan lebih besar dari sisa kursi.
5. Kartu akses hangtag: lubang tali, logo NVM, stempel, garis sobek bertakik, kode, peserta, acara, jadwal. Varian daftar tunggu "Nomor urut {n} saat mendaftar".
6. Simpan kartu sebagai PNG (menu bagikan di HTTPS, unduhan biasa, gambar untuk ditekan lama), salin kode dengan cadangan untuk HTTP.
7. Kartu diingat di `localStorage` dengan pemeriksaan bentuk data, "Daftarkan orang lain", tombol ke formulir di bawah 1024px.

Selesai bila: Solo, VIP, dan daftar tunggu berhasil dari browser dan tercatat di database, `check.sh` lulus.
Referensi: FR-10 sampai FR-24.

### Laporan

Status: selesai, 5 November 2026.

Hasil uji:
- Solo, VIP dua tamu, dan daftar tunggu tercatat di `registrations` dan `attendees` dengan status, posisi, dan kode yang sama dengan kartu.
- Email terdaftar ditolak dan pesannya tampil di kolom tamu yang tepat, tanpa baris baru.
- Peringatan rombongan muncul dan hilang saat tamu ditambah dan dihapus, mode daftar tunggu tampil saat kursi habis.
- PNG kartu utuh dengan D-DIN, JetBrains Mono, dan logo NVM.
- Kartu tampil lagi saat link dibuka ulang di perangkat yang sama, dengan tanggal penyimpanan.

Keputusan:
- Idempotency key memakai `crypto.getRandomValues`, karena `crypto.randomUUID` tidak tersedia di HTTP lewat IP LAN.
- Salin kode memakai `execCommand` sebagai cadangan, karena Clipboard API hanya ada di HTTPS.
- Simpan kartu juga menampilkan gambarnya, karena browser bawaan WhatsApp dan Instagram sering mengabaikan unduhan.
- Kartu yang dipulihkan menyebut tanggal penyimpanan, karena belum ada endpoint untuk status terbaru sebuah kode.
- Tombol "Konfirmasi kehadiran" di kolom kiri hanya di bawah 1024px, berupa link ke `#formulir`.
- Label "Nama akun" belum diberi petunjuk karena jenis akunnya belum diputuskan NVM.

Bentol:
- Kolom isi otomatis Chrome berwarna biru di luar palet. Diperbaiki dengan gaya `-webkit-autofill`.
- "Data kamu" tidak tebal karena berupa `legend`. Diberi `font-bold`.
- PNG kartu terpotong di kanan karena margin `mx-auto` ikut tergambar. Diperbaiki dengan `style: { margin: '0' }`.
- `html-to-image` sempat terpasang di root repo, bukan di `frontend/`. Dipindah, dan `.gitignore` mengabaikan `node_modules/` di mana pun.

Untuk dipertimbangkan:
- Endpoint publik untuk menanyakan status terbaru berdasarkan kode.

---

## Sprint 7: Halaman admin

Tujuan: admin mengelola data dari browser.

1. Form masuk dengan pesan "Sandi salah. Coba lagi."
2. Ringkasan dengan state memuat, kosong, gagal.
3. Tombol "Unduh Excel" dan "Unduh CSV".
4. Tombol keluar yang menghapus sesi.

Selesai bila: admin bisa masuk, melihat angka yang cocok dengan database, dan mengunduh dua format, `check.sh` lulus.
Referensi: FR-30, FR-31, FR-37.

### Laporan

Status: selesai, 5 November 2026. Commit `f1b4601`.

Yang dikerjakan:

| Bagian | Hasil |
|---|---|
| Cek sesi saat dibuka | Skeleton dulu. Cookie masih berlaku, langsung ke rekap. Kalau tidak, ke form masuk |
| Form masuk | Satu kolom "Sandi admin" dengan fokus otomatis. Kosong: "Isi sandi admin." Sandi salah dan batas percobaan dari server |
| Rekap | Terkonfirmasi "{x} dari {kapasitas} peserta", Daftar tunggu "{y} peserta", Batal "{z} registrasi", angka JetBrains Mono tabular |
| State | Memuat (skeleton), gagal (pesan dan "Coba lagi"), kosong ("Belum ada peserta. File unduhan hanya berisi judul kolom.") |
| Unduh | "Unduh Excel" (utama) dan "Unduh CSV" (outline), bertumpuk di HP, berdampingan mulai `sm` |
| Keluar | Cookie dihapus lewat API, form tampil dengan "Kamu sudah keluar." |

Keputusan:
- Sesi dicek lagi sebelum mengunduh. File diunduh lewat link biasa supaya browser memakai nama dari `Content-Disposition`. Tanpa cek ini, sesi yang habis akan menyimpan error JSON sebagai `.xlsx` rusak.
- Jawaban 401 dari endpoint mana pun mengembalikan admin ke form masuk.
- Tanpa persentase atau grafik. Admin butuh angka yang sama dengan isi file.
- Setelah masuk, fokus pindah ke judul "Rekap pendaftaran" untuk pembaca layar.

Hasil uji:

| Uji | Hasil |
|---|---|
| Rekap dengan data uji (kapasitas 3, dua VIP, satu batal) | "2 dari 3 peserta", "0 peserta", "1 registrasi" |
| CSV dibuka di Excel dengan pengaturan regional Indonesia | Kolom terpisah rapi, tidak perlu pemisah titik koma |
| `check.sh` | Lulus |

---

## Sprint 8: Docker dan deploy LAN

Tujuan: aplikasi jalan di VM dan bisa dibuka dari perangkat lain di Wi-Fi kantor.

1. `backend/Dockerfile` multi-stage ke distroless, user non-root, binary `api` dan `doomsday-admin`.
2. `frontend/Dockerfile` multi-stage, disajikan Nginx.
3. `frontend/nginx.conf`: aset statis, proxy `/api`, `X-Forwarded-For`, batas body, header keamanan.
4. `deploy/docker-compose.yml`: backend tanpa port ke host, frontend di 8090, tersambung ke network Postgres, `restart: unless-stopped`.
5. `deploy/.env` di VM (tidak masuk git), migrasi sebagai langkah deploy.
6. Backup harian `pg_dump` ke folder di VM.
7. Uji akses dari HP dan PC lain di Wi-Fi kantor.

Selesai bila: `http://<IP VM>:8090` terbuka dari perangkat lain di LAN, pendaftaran tercatat, backup pertama terbentuk.
Referensi: NFR-04, NFR-05, NFR-07.

### Laporan

Status: selesai, 5 November 2026. Commit `77c2131`, `b26e6e5`, `b93ebe2`, `fb71dda`.

Pelajaran dari deploy compro yang langsung diterapkan:

| Kejadian di compro | Penerapan di Doomsday |
|---|---|
| Binary hasil CGO restart terus di distroless dengan `no such file or directory` | `CGO_ENABLED=0` dan pengaman `go version -m` yang menggagalkan build |
| `compose config` tanpa `--quiet` mencetak sandi | Validasi selalu dengan `--quiet` |
| Jam VM memakai UTC | Cron ditulis dalam UTC dengan keterangan WIB |
| Backup lewat `docker exec pg_dump` tanpa sandi, dengan trap, log, dan izin ketat | Dipakai sejak awal, ditambah verifikasi `pg_restore --list` |
| Request LAN membawa IP asli klien | `X-Forwarded-For` ditimpa Nginx, backend hanya percaya subnet Doomsday |

Hasil uji:

| Uji | Hasil |
|---|---|
| Build pertama di VM | Selesai dalam 128 detik |
| `doomsday ps` | Backend `Up` tanpa port ke host, frontend `Up (healthy)` di `0.0.0.0:8090` |
| Log backend | `server jalan`, `env: production`, tanpa bentol config atau koneksi |
| `/api/health` lewat Nginx | `{"database":"up","status":"ok"}` |
| `/`, `/admin`, `/assets/tidak-ada.js` | 200, 200, 404 |
| Header | CSP, `X-Frame-Options: DENY`, `Cache-Control: no-cache` di `index.html` |
| Migrasi lewat container `migrate/migrate:v4.18.1` | versi 3, `up` menjawab `no change` |
| Daftar Solo dari HP Android lewat Wi-Fi | 201 dari `192.168.68.141`, tercatat di database |
| Salin kode dan simpan kartu PNG di HP | Berhasil lewat HTTP, CSP tidak memblokir html-to-image |
| Admin dari PC lewat LAN | Masuk, rekap, dan unduh Excel berhasil, console bersih |
| IP palsu: 7 login dengan `X-Forwarded-For: 203.0.113.1` sampai `.7` | 401 lima kali lalu 429, backend mengabaikan header palsu |
| Perangkat dihitung terpisah | Login HP tetap "Sandi salah" saat PC sedang terkena batas |
| Log Nginx | IP asli PC `192.168.68.143` dan HP `192.168.68.141`, bukan IP gateway Docker |
| Backup manual | Dua kali `OK`, file `-rw-------`, folder `drwx------` |
| Uji pemulihan ke `nvm_doomsday_restore` | Versi migrasi 3 dan jumlah baris sama dengan asli. Tabel peserta saat itu kosong, jadi isi data diuji ulang di Sprint 10 |
| Cron | `15 19 * * *` (02.15 WIB), 15 menit setelah backup compro |

Keputusan:
- Image akhir backend `distroless/static-debian12:nonroot` tanpa shell dan libc, jalan sebagai uid 65532.
- Migrasi tidak ikut di image backend. Dijalankan dengan container `migrate/migrate` sebagai `doomsday_migrator`, supaya backend tidak pernah memegang kredensial pemilik tabel.
- Nginx me-resolve ulang `doomsday-backend` lewat DNS Docker (`resolver 127.0.0.11`), supaya tidak 502 setelah backend di-build ulang sendirian.
- `try_files $uri /index.html` tanpa `$uri/`, dan file yang tidak ada di `/assets/` dijawab 404.
- Cache: `/assets/` 1 tahun, font dan logo 7 hari, `index.html` selalu dicek ulang.
- Header keamanan hanya di level `server`, cache diatur dengan `expires`, karena `add_header` di location mematikan pewarisan.
- CSP mengizinkan `style-src 'unsafe-inline'` dan `img-src data: blob:` untuk html-to-image. Script hanya `'self'`.
- Network `doomsday` dengan subnet tetap `172.30.93.0/24` hanya berisi Nginx dan backend. Container lain di `nvm-internal_default` tidak dipercaya.
- Backend read-only, semua capability dicabut, `no-new-privileges`. Nginx hanya `no-new-privileges`.
- Log container dibatasi 3 x 10 MB per service.
- Health check di frontend memanggil `/api/health` lewat Nginx, sekaligus menguji backend dan database.
- `SESSION_SECRET` dan sandi admin produksi berbeda dari dev.
- Backup di `/srv/nvm-doomsday-data/backup`, terpisah dari repo `/srv/nvm-doomsday`, retensi 14 hari. Backup lama baru dihapus setelah backup baru terbukti baik.

Bentol:
- `git log` tanpa `--oneline` membuka pager, lalu sempat dihentikan dengan `Ctrl+Z` sehingga prosesnya tertinggal. Dibereskan dengan `kill %1`. Untuk melihat commit tanpa pager: `git --no-pager log --oneline -5`.
- Uji daftar pertama dari HP tidak tercatat di log Nginx. Diulang dan tercatat 201.

Operasional di VM:

| Keperluan | Perintah |
|---|---|
| Alias (di `~/.bashrc`) | `doomsday` = `docker compose -f /srv/nvm-doomsday/deploy/docker-compose.yml --env-file /srv/nvm-doomsday/deploy/.env` |
| Status dan log | `doomsday ps`, `doomsday logs doomsday-backend --tail 50` |
| Deploy ulang | `cd /srv/nvm-doomsday && git pull`, migrasi bila ada file baru di `backend/migrations`, lalu `doomsday up -d --build` |
| Migrasi | `docker run --rm --network nvm-internal_default -v /srv/nvm-doomsday/backend/migrations:/migrations:ro migrate/migrate:v4.18.1 -path=/migrations -database "postgres://doomsday_migrator:<sandi>@nvm-internal-postgres:5432/nvm_doomsday?sslmode=disable" up`, sandi dibaca dengan `read -rs` |
| CLI admin | `doomsday exec doomsday-backend doomsday-admin <perintah>` |
| Backup manual | `/srv/nvm-doomsday/deploy/backup.sh`, hasilnya di `backup.log` |
| Pulihkan produksi (belum diuji) | `doomsday stop doomsday-backend`, lalu `docker exec -i nvm-internal-postgres pg_restore -U doomsday_migrator -d nvm_doomsday --clean --if-exists --single-transaction < <file>`, lalu `doomsday start doomsday-backend`. Tanpa `--no-privileges`, supaya grant ke `doomsday_app` ikut pulih |

---

## Sprint 9: Revisi cek nama akun VIP dan aset Doomsday

Tujuan: revisi dari manajer. Tipe pendaftaran ditentukan dari daftar VIP milik admin, bukan dipilih pengunjung, dan halaman punya tempat untuk foto atau aset Doomsday.

Keputusan revisi (5 November 2026):

| Hal | Keputusan |
|---|---|
| Alur | Setelah amplop dibuka, pengunjung mengisi satu kolom "Nama akun". Ada di daftar VIP: formulir VIP. Tidak ada: formulir Solo |
| Hak VIP | Tetap seperti sebelumnya: boleh membawa maksimal 3 tamu. Solo hanya dirinya |
| Pengaman | Satu nama akun VIP hanya untuk satu registrasi yang belum batal. Kalau registrasinya dibatalkan, nama itu bisa dipakai lagi |
| Input VIP | Halaman admin: tambah satu, tempel banyak (satu nama per baris), nonaktifkan, lihat nama mana yang sudah dipakai kode registrasi mana |
| Aset | File di `frontend/public/doomsday/` dan satu file daftar di `src/content/`. Ganti foto artinya commit dan build ulang. Daftar kosong, bagiannya tidak tampil |
| Penentu tipe | Server, di dalam transaksi pendaftaran. Tipe tidak lagi dikirim dari browser |
| Pencocokan | Tidak peka huruf besar kecil, spasi di awal dan akhir dibuang, `@` di depan dibuang |
| Nama VIP sudah dipakai | Pengunjung melihat "Nama akun ini sudah terdaftar. Hubungi admin NVM." Tidak otomatis jadi Solo |
| Hapus VIP | Ditandai nonaktif, bukan dihapus, karena `doomsday_app` tidak punya izin DELETE |

Rencana:

1. Migrasi 0004: tabel `vip_accounts` (nama asli, nama ternormalisasi unik, aktif, waktu dibuat) dan kolom `registrations.vip_account_id` dengan index unik untuk registrasi yang belum batal.
2. Backend: fungsi normalisasi nama, endpoint publik cek nama akun dengan rate limit, pendaftaran menentukan tipe dari daftar VIP di dalam kunci event.
3. API admin daftar VIP: lihat, tambah, tempel banyak (laporan ditambah, sudah ada, dilewati), nonaktifkan. Perintah CLI yang sama untuk cadangan.
4. Halaman undangan: langkah nama akun, lalu formulir VIP atau Solo dengan nama akun pendaftar terkunci dan tombol untuk mengganti nama akun.
5. Halaman admin: bagian Daftar VIP dengan state memuat, kosong, gagal.
6. Slot aset Doomsday dari folder dan file daftar, dengan teks alternatif wajib.
7. Deploy ulang ke VM: `git pull`, migrasi 0004, build, uji ulang pemulihan backup dengan data.

Selesai bila: nama VIP menghasilkan formulir VIP, nama lain menghasilkan Solo, nama VIP kedua kalinya ditolak, admin bisa mengelola daftar dari browser, aset tampil dari folder, semuanya jalan di VM, `check.sh` lulus.

### Laporan

Status: selesai, 5 November 2026. Commit `497b564`, `12dd797`, `a9fdb90`, `b38fd42`, `5343dc6`, `70bdc06`.

Yang dikerjakan:

| Bagian | Hasil |
|---|---|
| Database | Migrasi 0004: tabel `vip_accounts` (`account_name` seperti diketik admin, `account_key` ternormalisasi dan unik), kolom `registrations.vip_account_id`, index unik parsial untuk registrasi yang belum batal, CHECK Solo tanpa akun VIP |
| API publik | `POST /api/account-check` menjawab `vip`, `solo`, atau 409 `account_taken`, dengan rate limit 20 beruntun lalu 1 per 3 detik. `POST /api/registrations` tidak lagi membaca `type` |
| API admin | `GET`, `POST`, dan `PATCH /api/admin/vip` untuk melihat, menambah satu atau banyak nama, dan menonaktifkan atau mengaktifkan lagi |
| CLI | `vip list`, `vip add <nama>...`, `vip off <nama>`, `vip on <nama>` |
| Undangan | Formulir dua langkah: nama akun dulu, lalu formulir VIP (0 sampai 3 tamu) atau Solo. Nama akun pendaftar dikunci, dengan tombol "Ganti nama akun". FAQ ditambah pertanyaan nama VIP yang terbaca Solo |
| Admin | Panel "Daftar VIP": tambah lewat tempel dari Excel, laporan per kelompok, cari, nonaktifkan, status pemakaian per nama |
| Aset | Ilustrasi orisinal studio bioskop di sisi kiri dan kanan layar mulai 1360px, dengan keterangan "Ilustrasi sementara". Daftar aset di `src/content/assets.ts` |

Hasil uji:

| Uji | Hasil |
|---|---|
| Migrasi 0004 `up`, `down 1`, `up` | Bersih. Izin `doomsday_app` di `vip_accounts` hanya INSERT, SELECT, UPDATE |
| CHECK `account_key` | Huruf besar dan awalan `@` ditolak database |
| CHECK Solo dengan `vip_account_id` | Ditolak database |
| Normalisasi | `ujivip` dan ` @UjiVIP ` sama-sama `vip`. Kosong dan `@@` 422 |
| Nama bukan VIP membawa tamu | 422 di kolom `attendees` |
| Nama VIP dipakai dua kali | Cek dan daftar kedua 409 `account_taken`. Setelah registrasi pertama dibatalkan, nama itu kembali `vip` |
| API admin VIP | Tanpa cookie 401. Tempel 7 baris: 3 ditambah, `@` dan 51 karakter ditolak, duplikat dan baris kosong dilewati. Nama yang dinonaktifkan terbaca Solo dan tetap menunjukkan kode registrasinya. Tambah lagi mengaktifkan ulang. ID salah 404, tanpa `is_active` 422 |
| CLI VIP | `vip list` dan `vip off` sesuai |
| `TRUSTED_PROXIES` tidak valid | Server berhenti dengan `invalid CIDR address`, tanpa `server jalan` |
| Uji gabungan frontend di dev (admin, formulir dua langkah, aset di 1440, 1280, dan 375 px, keyboard di 320 px) | Lulus |
| Deploy ke VM | Build 15,9 detik dengan cache, migrasi `no change`, frontend `healthy`, `account-check` menjawab, SVG `200 image/svg+xml`, CLI di container membaca daftar VIP |
| Data setelah uji | 1 registrasi (dari HP di Sprint 8), daftar VIP kosong |

Perubahan dari rencana:
- VIP boleh datang tanpa tamu. Dulu VIP wajib membawa tamu karena pengunjung bisa memilih Solo, sekarang affiliator VIP tidak punya pilihan itu.
- Menonaktifkan nama yang sudah dipakai tidak mengubah registrasinya. Registrasi tetap VIP sampai admin membatalkannya, supaya status yang sudah masuk tidak berubah diam-diam karena daftar dirapikan.
- Tempelan nama VIP diproses satu per satu tanpa satu transaksi besar, supaya satu baris salah dari Excel tidak menggagalkan nama lain. Batas 500 nama per tempelan. Kalau tersalin lebih dari satu kolom, hanya kolom pertama yang dipakai.
- Aset dipindah dari key visual di atas undangan ke sisi kiri dan kanan layar lebar, atas permintaan, supaya tidak menambah jarak gulir. Di bawah 1360px aset tidak tampil dan tidak diunduh.
- Aset sementara berupa ilustrasi orisinal, bukan materi resmi film, sampai NVM memutuskan materi yang dipakai.
- Ditemukan dan diperbaiki: error dari `httpapi.NewRouter` di `cmd/server/main.go` tidak pernah dicek, sehingga `TRUSTED_PROXIES` yang salah ketik akan membuat server jalan dengan router nil.
- Uji pemulihan backup dengan data (langkah 7) dan uji alur dari HP setelah deploy Sprint 9 dipindah ke checklist Sprint 10, karena data uji sudah dibersihkan sebelum keduanya dijalankan.

Bentol:
- `sqlc generate` sempat jalan sebelum `sqlc.yaml` tersimpan, sehingga `idempotency_key` jadi `pgtype.UUID` dan build gagal. Beres setelah generate ulang.
- Backend dev gagal start dengan `ping database: context deadline exceeded` karena tunnel SSH sedang menyambung ulang. Saran `ServerAliveInterval 60` di `~/.ssh/config`.
- `<kode>` di contoh perintah sempat diketik apa adanya, dan bash membacanya sebagai pengalihan input. Tanda kurung sudut di contoh artinya "ganti dengan nilai sebenarnya".
- Uji cancel sempat memakai kode registrasi yang salah (Solo, bukan VIP), sehingga nama VIP masih terpakai. Hasilnya justru benar.
- Login curl kedua memakai sandi yang sudah di-`unset`, dan `-c` menimpa cookie jar. Login ulang sebelum blok uji berikutnya.
- Asisten tidak bisa menjalankan `vue-tsc` dan ESLint di workspace-nya karena registry npm diblokir. Pemeriksaan frontend sepenuhnya lewat `check.sh` di PC.

Operasional tambahan di VM:

| Keperluan | Perintah |
|---|---|
| Lihat daftar VIP | `doomsday exec doomsday-backend doomsday-admin vip list` |
| Tambah nama VIP | `doomsday exec doomsday-backend doomsday-admin vip add <nama> <nama>...`, atau lewat halaman admin |
| Nonaktifkan atau aktifkan lagi | `doomsday exec doomsday-backend doomsday-admin vip off <nama>` atau `vip on <nama>` |
| Ganti aset | Taruh file di `frontend/public/doomsday/`, ubah `src/content/assets.ts`, commit, lalu `git pull` dan `doomsday up -d --build` di VM |

---

## Sprint 10: Pengujian menyeluruh

Tujuan: semua kebutuhan terbukti jalan sebelum link disebar.

Rencana:

1. Backend unit test: validasi, generator kode, normalisasi nama akun (`NormalizeAccount` di Go dan `normalizeAccount` di frontend harus memberi hasil sama), aturan kuota dan daftar tunggu, sanitasi CSV, `adminauth` (token kedaluwarsa, tanda tangan salah), `export` (urutan, nomor urut).
2. Backend integration test dengan database uji: semua kode respons di SRS 7, idempotency, email unik, VIP sekali pakai, CLI admin.
3. Uji balapan: banyak pendaftaran serentak saat sisa kursi sedikit tidak melebihi kapasitas, dan dua pendaftaran serentak dengan nama VIP sama hanya satu yang lolos.
4. Frontend Vitest: validasi formulir, `newIdempotencyKey`, pemetaan error, kesamaan pesan dengan backend. `vitest run` masuk ke `check.sh`.
5. Uji manual per FR, termasuk browser bawaan WhatsApp dan Instagram.
6. Aksesibilitas: keyboard penuh, focus terlihat, kontras dihitung ulang, reduced motion.
7. Responsif: 320, 375, 768, 1024, 1440 px tanpa scroll horizontal.
8. Delivery Gate AntiSlop, satu baris PASS atau FAIL per poin beserta bukti.

Checklist uji yang ditunda dari sprint sebelumnya:

| Asal | Uji | Hasil |
|---|---|---|
| Sprint 1 | Health 503 saat database putus, pulih tanpa restart, gagal di awal saat database tidak terjangkau | lulus, `health_integration_test.go` dan `cmd/server` |
| Sprint 1 | Membatalkan registrasi membuat email pesertanya bisa dipakai lagi | lulus, integration test |
| Sprint 5 | Animasi amplop, fokus setelah dibuka, ingatan per sesi, versi tanpa animasi | lulus, `undangan.spec.ts` |
| Sprint 5 | Panel kursi sebagian terisi, penuh, ada daftar tunggu, ditutup, pembaruan saat tab kembali terlihat | lulus, `undangan.spec.ts` |
| Sprint 6 | Keyboard penuh formulir, tambah dan hapus tamu, fokus setelah gagal dan berhasil | lulus setelah perbaikan bentol 2, `keyboard.spec.ts` dan `formulir.spec.ts` |
| Sprint 6 | Idempotency key: jawaban hilang, kirim ulang, kode sama | lulus, `formulir.spec.ts` |
| Sprint 7 | Kirim kosong, sandi salah, batas 5 percobaan, sesi habis saat unduh, keluar, rekap kosong, keyboard di 320px | lulus, `admin.spec.ts` dan `keyboard.spec.ts` |
| Sprint 7 | Unduhan admin di Chrome Android dan Safari iOS | Android lulus manual, Safari iOS belum |
| Sprint 8 | Uji pemulihan backup dengan data, prosedur pulih produksi | lulus, lihat hasil uji di bawah |
| Sprint 9 | Dua tab mengirim nama VIP yang sama | lulus, `formulir.spec.ts` |
| Sprint 9 | Alur Solo dan VIP dari HP lewat LAN setelah deploy Sprint 9 | lulus manual di Android Chrome dan Firefox |
| Sprint 9 | Pembaca layar di Daftar VIP | axe tanpa pelanggaran. Pembaca layar sungguhan belum diuji |
| Sprint 9 | `staticcheck` di `check.sh` | dipasang, v0.8.2 |

Selesai bila: semua checklist lulus, tidak ada bentol terbuka berprioritas M.

### Laporan

Status: selesai, 5 November 2026. Commit `abaeb8f` (revisi teks), `7588324` (perbaikan), `18ef2fa` (test).

Cara kerja:
- Test ditulis dan dijalankan Claude Code di PC. Aturannya: hanya file test, config test, dan script test. Bentol di aplikasi dilaporkan tanpa diperbaiki, ekspektasi test tidak boleh dilonggarkan, dan hanya database `nvm_doomsday_test` yang boleh disentuh.
- Commit, push, reset, dan checkout dikunci di `.claude/settings.local.json`. Perbaikan aplikasi dan commit dikerjakan sendiri.
- Database uji `nvm_doomsday_test` dibuat terpisah dengan role yang sama. `ALTER DEFAULT PRIVILEGES` diulang di database itu, karena pengaturan ini berlaku per database.

Hasil test (setelah perbaikan):

| Kelompok | Isi | Hasil |
|---|---|---|
| A. Unit Go | registration, adminauth, export, config, httpapi tanpa database, penjaga testdb | semua lulus |
| B. Kasus bersama | 25 kasus normalisasi dan 18 validasi di `testdata/`, dibaca Go dan Vitest | Go 43 lulus. Vitest lulus, `İ` dan `Σ` dilewati sebagai khusus Go |
| C. Integration Go | kode respons SRS 7, idempotency, email unik, VIP sekali pakai, API admin VIP, health, CLI | semua lulus |
| D. Uji balapan | 20 pendaftar serentak dengan kapasitas 5, 10 rebutan satu nama VIP | 5 terkonfirmasi dan 15 daftar tunggu bernomor 1 sampai 15. Tepat 1 yang dapat nama VIP |
| E. Vitest | validasi, uuid, useRecipient, pemetaan error klien API | 88 lulus, 2 dilewati |
| F. Playwright E2E | 84 test: alur undangan, formulir dua langkah, kartu, admin, keyboard, axe, 6 lebar layar, gambar samping | 84 lulus |
| G. Script | env kosong, database bukan `_test`, tunnel mati, port terpakai | 4 skenario berhenti dengan pesan jelas |
| Uji pemulihan | backup produksi dipulihkan ke `nvm_doomsday_test` sebagai `doomsday_migrator` dengan `--clean`, tanpa `--no-privileges` | Jumlah baris sama dengan produksi. `doomsday_app` bisa SELECT, DELETE ditolak |
| Uji manual | Android Chrome dan Firefox lewat LAN | lulus |

Bentol yang ditemukan test dan diperbaiki di `7588324`:

| # | Bentol | Perbaikan |
|---|---|---|
| 1 | Tempelan `@ @budi` di Daftar VIP menjawab 500 dan menggagalkan nama lain di tempelan yang sama, karena `@` di depan hanya dibuang sekali dan kuncinya ditolak CHECK | `NormalizeAccount` membuang `@`, spasi, dan BOM di depan sekaligus. Tolakan CHECK dari database masuk ke daftar `rejected` per nama |
| 2 | Fokus tidak kembali ke kolom setelah sandi salah atau nama VIP terpakai, karena `focus()` dipanggil saat kolom masih `disabled` | `await nextTick()` sebelum `focus()` di `AdminPage.vue` dan `RegistrationForm.vue` |
| 3 | Browser menerima email seperti `budi..santoso@email.com` yang ditolak server, sehingga memakai jatah rate limit | Regex email disamakan dengan aturan `mail.ParseAddress` untuk alamat polos |
| 4 | Normalisasi Go dan JavaScript berbeda untuk BOM dan NEL | Kedua sisi membuang BOM dan NEL. `İ` dan `Σ` tetap berbeda dan ditandai khusus Go, karena huruf kecil yang menentukan VIP hanya dihitung di backend |
| 5 | `:key` daftar peserta dan daftar nama ditolak memakai nama, sehingga bisa bentrok | Pakai indeks, karena kedua daftar hanya ditampilkan |
| 6 | staticcheck S1016 di `registration/admin.go` | `AttendeeRequest(before)` |

Revisi teks dari manajer (`abaeb8f`):
- Kalimat "Pendaftaran ini untuk kamu sendiri." di formulir Solo dihapus, karena terdengar membedakan. Keterangan VIP tetap ada.
- FAQ tinggal tiga pertanyaan: siapa yang boleh daftar, biaya, dan jam datang.

Delivery Gate AntiSlop:

| Poin | Hasil | Bukti |
|---|---|---|
| Tanpa em dash | PASS | cek em dash di `check.sh` |
| Tanpa scroll horizontal | PASS | `tampilan.spec.ts`, 42 kombinasi keadaan dan lebar dari 320 sampai 1440 px |
| Tap target minimal 44x44 px | PASS | `tampilan.spec.ts`, 42 kombinasi |
| Kontras WCAG AA | PASS | axe-core di 42 kombinasi, tanpa pelanggaran |
| Keyboard penuh, focus terlihat | PASS | `keyboard.spec.ts` di 320 px, outline 2px dicek |
| Fokus setelah error | PASS | setelah perbaikan bentol 2 |
| State kosong, memuat, gagal | PASS | "detail acara gagal dimuat" dan rekap kosong |
| Fakta tidak dikarang | PASS dengan catatan | jadwal tayang masih perkiraan dan aset masih ilustrasi bertanda "sementara", keduanya masuk daftar sebelum link disebar |
| FAQ spesifik produk | PASS | tiga pertanyaan, jam kumpul dibaca dari data acara |
| Link navigasi menuju tempat yang ada | PASS dengan catatan | Hanya ada dua tujuan: `#formulir` dan halaman tidak ditemukan (dicek di `tampilan.spec.ts`). Link `#formulir` belum punya test khusus |
| Toggle tema | tidak berlaku | situs hanya mode gelap, dipilih sejak SRS |
| Console tanpa error | PASS | fixture di 84 test E2E |

Bentol dan catatan proses:
- Claude Code berhenti dua kali untuk bertanya: env database uji belum diisi, dan port 8081 serta 5174 masih dipakai backend dan Vite dev. Keduanya memang perlu keputusan manusia.
- Baris `TEST_` di `backend/.env` sempat tertulis dua kali karena perintah penambahan dijalankan saat barisnya sudah ada.
- Produksi sempat terlihat kosong (0 registrasi, padahal sebelumnya 1). Dilacak lewat `pg_stat_user_tables`: `n_tup_del` sama dengan `n_tup_ins`, artinya terhapus oleh `DELETE` manual saat bersih-bersih. Test memakai `TRUNCATE`, dan kedua URL test terbukti mengarah ke `nvm_doomsday_test`.
- Build frontend di VM jadi sekitar 32 detik karena `npm ci` ikut memasang Vitest dan Playwright. Image akhir tidak terpengaruh, karena yang disalin hanya `dist`.

Belum diuji:
- Safari iOS dan browser bawaan WhatsApp atau Instagram, karena butuh perangkat sungguhan.
- Pembaca layar sungguhan. axe hanya memeriksa label dan ARIA.

Menjalankan test:

| Kelompok | Perintah |
|---|---|
| Unit Go dan Vitest | `./scripts/check.sh` |
| Integration dan uji balapan | `./scripts/test-integration.sh`, atau `./scripts/test-integration.sh -run TestBalapan -v` |
| E2E | matikan backend dan Vite dev, tunnel tetap menyala, lalu `./scripts/test-e2e.sh`. Satu file: `./scripts/test-e2e.sh e2e/admin.spec.ts` |

---

## Sprint 11: Rilis publik (tertahan)

Tujuan: link bisa dibuka affiliator dari luar kantor.

1. Subdomain: `letter.novoidminds.com` diusulkan, menunggu persetujuan manajer.
2. Cloudflare Tunnel ke `doomsday-frontend`, HTTPS dari Cloudflare.
3. IP asli pengunjung dari `CF-Connecting-IP`, hanya dipercaya dari gateway tunnel (pola yang sama dengan compro).
4. `COOKIE_SECURE=true`.
5. Pertimbangkan membatasi `/admin` lewat Cloudflare Zero Trust.
6. Uji menu bagikan bawaan HP untuk simpan kartu dan Clipboard API, yang hanya aktif di HTTPS.

Tertahan sampai: subdomain disetujui.

---

## Sebelum link disebar

| Cek | Cara |
|---|---|
| Jadwal resmi terisi, atau dikosongkan | `doomsday-admin event set show_at ...` atau `event set show_at -` |
| Bioskop, kota, studio, link peta | `doomsday-admin event set ...` |
| Kapasitas 200 dan pendaftaran terbuka | `doomsday-admin event show` |
| Data uji terhapus | Dihapus sebagai `doomsday_migrator`, karena `doomsday_app` tidak punya DELETE. Backup dulu sebelum menghapus |
| Daftar VIP final dari NVM | Halaman admin, bagian Daftar VIP. Nama uji `rifky` dinonaktifkan atau tetap kalau memang VIP |
| Aset Doomsday | Ganti ilustrasi sementara dengan materi resmi, atau tetap pakai ilustrasi dan hapus keterangan "Ilustrasi sementara" |
| Database dev terpisah | Putuskan `nvm_doomsday_dev` untuk development, supaya uji coba tidak menyentuh data asli |
| Kontak admin di FAQ | Isi teks FAQ |

## Keputusan yang masih menunggu

| Keputusan | Mempengaruhi |
|---|---|
| "Nama akun" itu akun apa (TikTok, Shopee, Instagram, atau lainnya) | Label dan petunjuk kolom, dan sekarang juga pencocokan VIP, karena ejaan nama harus sama dengan daftar admin |
| Bioskop, kota, studio, jadwal | Isi `event_config` |
| Batas akhir pendaftaran | Kolom deadline atau tutup manual |
| Perlu identitas saat hari H | Teks FAQ dan kartu akses |
| Kontak admin untuk FAQ | Teks FAQ dan pesan nama VIP sudah dipakai |
| Subdomain (`letter.novoidminds.com` diusulkan) | Sprint 11 |
| Lama penyimpanan data setelah acara | Prosedur hapus data dan backup |
| Batas jumlah daftar tunggu | Aturan pendaftaran (saat ini tanpa batas) |
| Materi visual resmi Avengers: Doomsday boleh dipakai atau tidak | Isi `src/content/assets.ts` |
| Aset juga tampil di HP atau tidak (misalnya di bawah FAQ) | `SideVisuals.vue` dan halaman undangan |
