#!/usr/bin/env bash
# Integration test backend dengan database uji nvm_doomsday_test. Tidak masuk check.sh karena butuh database.
# Butuh: tunnel (./scripts/dev-db-tunnel.sh), TEST_DATABASE_URL dan TEST_MIGRATE_DATABASE_URL di backend/.env,
# dan CLI migrate. Argumen tambahan diteruskan ke go test, misalnya: -run TestBalapan -v
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/test-env.sh
source "$root/scripts/test-env.sh"

require_test_env
require_tunnel
require_cmd migrate "pasang golang-migrate v4 dengan tag postgres"

cd "$root/backend"
# -p 1: semua paket memakai satu database uji yang dikosongkan di antara test.
# -count=1: hasil dari database tidak boleh diambil dari cache go test.
exec go test -tags integration -count=1 -p 1 "$@" ./...
