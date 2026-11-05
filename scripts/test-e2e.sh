#!/usr/bin/env bash
# E2E Playwright (Chromium) dengan backend di 8081 dan Vite di 5174, keduanya dijalankan Playwright sendiri.
# Backend memakai TEST_DATABASE_URL, bukan DATABASE_URL. Tidak masuk check.sh karena butuh database.
# Argumen tambahan diteruskan ke playwright test, misalnya: e2e/admin.spec.ts --headed
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/test-env.sh
source "$root/scripts/test-env.sh"

require_test_env
require_tunnel
require_cmd migrate "pasang golang-migrate v4 dengan tag postgres"
require_cmd go "pasang Go sesuai backend/go.mod"

# Port yang sudah terpakai hampir pasti backend atau Vite dev, yang tersambung ke database asli.
for port in 8081 5174; do
  if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
    fail "port $port sudah dipakai. Hentikan backend atau Vite dev dulu, E2E menjalankan servernya sendiri."
  fi
done

cd "$root/frontend"
if [[ ! -d node_modules/@playwright/test ]]; then
  fail "dependency frontend belum lengkap, jalankan: npm ci (di frontend/)"
fi
exec npx --no-install playwright test "$@"
