#!/usr/bin/env bash
# Backup harian database nvm_doomsday. Dipanggil cron di VM oleh user yang ada di grup docker.
# Semua nilai bisa diganti lewat env, misalnya BACKUP_DIR=/tmp/uji ./deploy/backup.sh
set -Eeuo pipefail
umask 077

PG_CONTAINER="${PG_CONTAINER:-nvm-internal-postgres}"
DB_NAME="${DB_NAME:-nvm_doomsday}"
DB_USER="${DB_USER:-doomsday_migrator}"
BACKUP_DIR="${BACKUP_DIR:-/srv/nvm-doomsday-data/backup}"
KEEP_DAYS="${KEEP_DAYS:-14}"

# Dicek sebelum trap dipasang, karena trap sendiri menulis ke log di folder ini.
if [[ ! -d "$BACKUP_DIR" || ! -w "$BACKUP_DIR" ]]; then
  echo "folder backup $BACKUP_DIR tidak ada atau tidak bisa ditulis oleh $(id -un)" >&2
  exit 1
fi

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="$BACKUP_DIR/nvm_doomsday-$stamp.dump"
tmp="$target.tmp"
log="$BACKUP_DIR/backup.log"

log_line() { printf '%s %s\n' "$(date -Iseconds)" "$*" >>"$log"; }

fail() {
  trap - ERR
  rm -f "$tmp"
  log_line "GAGAL pesan=$1"
  echo "backup gagal: $1" >&2
  exit 1
}
trap 'fail "perintah gagal di baris $LINENO"' ERR

# Sisa run yang terputus tanpa sempat menjalankan trap, misalnya VM mati saat backup.
find "$BACKUP_DIR" -maxdepth 1 -name 'nvm_doomsday-*.dump.tmp' -mtime +0 -delete

# Socket lokal di dalam container tidak butuh sandi. Format custom sudah terkompresi
# dan bisa dipulihkan sebagian dengan pg_restore.
docker exec "$PG_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc >"$tmp"

# Ukuran file bukan bukti. Dump yang terpotong ditolak saat daftar isinya dibaca.
docker exec -i "$PG_CONTAINER" pg_restore --list <"$tmp" >/dev/null ||
  fail "hasil dump tidak bisa dibaca pg_restore"

mv "$tmp" "$target"
size="$(du -h "$target" | cut -f1)"

# Baru dihapus setelah backup hari ini terbukti baik. -mtime +13 berarti umur 14 hari atau lebih.
deleted="$(find "$BACKUP_DIR" -maxdepth 1 -name 'nvm_doomsday-*.dump' -mtime +"$((KEEP_DAYS - 1))" -print -delete | wc -l)"

log_line "OK file=$(basename "$target") ukuran=$size dihapus=$deleted"
