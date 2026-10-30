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

step "backend: go vet"
go vet ./...

step "backend: go test"
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

step "teks: em dash"
# Aturan AntiSlop: teks yang dibaca pengguna tidak boleh memakai em dash.
if grep -rnI $'\u2014' \
  --include='*.vue' --include='*.ts' --include='*.css' --include='*.go' \
  "$root/frontend/src" "$root/backend"; then
  echo "Ganti em dash di atas dengan koma, titik, titik dua, atau kurung."
  false
fi

echo
echo "Semua cek lulus."