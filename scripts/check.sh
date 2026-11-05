#!/usr/bin/env bash
# Jalankan sebelum setiap commit: ./scripts/check.sh && git commit ...
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
current="persiapan"

# set -e menghentikan script di perintah pertama yang gagal; trap ini memberi tahu bagian mana.
trap 'echo; echo "GAGAL di bagian: ${current}" >&2' ERR

step() {
  current="$1"
  echo "==> $1"
}

cd "$root/backend"

step "backend: gofmt"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "File berikut belum diformat, jalankan: gofmt -w <file>"
  echo "$unformatted"
  false
fi

step "backend: sqlc diff"
# Gagal kalau isi internal/db/sqlc tidak sama dengan hasil generate dari query dan migrasi sekarang.
sqlc diff

step "backend: go vet"
go vet ./...
# File integration test hanya terbaca dengan tag ini. Di-vet juga supaya rusaknya ketahuan tanpa database.
go vet -tags integration ./...

step "backend: staticcheck"
# Versi dipin. go run mengunduhnya sekali ke cache modul, tanpa mengubah go.mod.
# Menangkap hal yang lolos go vet, misalnya hasil fungsi yang tidak pernah dipakai.
staticcheck_version="v0.8.2"
go run "honnef.co/go/tools/cmd/staticcheck@${staticcheck_version}" ./...
go run "honnef.co/go/tools/cmd/staticcheck@${staticcheck_version}" -tags integration ./...

step "backend: go test"
# Hanya unit test. Integration test butuh database: ./scripts/test-integration.sh
go test ./...

cd "$root/frontend"

step "frontend: dependency"
if [[ ! -d node_modules ]]; then
  echo "node_modules belum ada, jalankan: npm ci"
  false
fi

step "frontend: type check"
npm run --silent type-check

step "frontend: eslint"
npm run --silent lint:check

step "frontend: prettier"
npm run --silent format:check

step "frontend: vitest"
# E2E butuh database dan server: ./scripts/test-e2e.sh
npm run --silent test

step "teks: em dash"
# Aturan AntiSlop: teks yang dibaca pengguna tidak boleh memakai em dash.
if grep -rnI $'\u2014' \
  --include='*.vue' --include='*.ts' --include='*.css' --include='*.go' --include='*.json' --include='*.sh' \
  "$root/frontend/src" "$root/frontend/e2e" "$root/backend" "$root/testdata" "$root/scripts"; then
  echo "Ganti em dash di atas dengan koma, titik, titik dua, atau kurung."
  false
fi

echo
echo "Semua cek lulus."
