# Dipakai test-integration.sh dan test-e2e.sh lewat source. Bukan script yang dijalankan sendiri.
# backend/.env tidak di-source utuh, supaya DATABASE_URL produksi di dalamnya tidak ikut ter-export.

fail() {
  echo "BERHENTI: $1" >&2
  exit 1
}

# Nilai dari env didahulukan. Kalau kosong, baris KEY=... terakhir di backend/.env dipakai, tanpa kutip.
env_value() {
  local key="$1" value="${!1:-}"
  if [[ -z "$value" && -f "$root/backend/.env" ]]; then
    value="$(grep -E "^${key}=" "$root/backend/.env" | tail -n 1 | cut -d= -f2- || true)"
    value="${value%\"}" value="${value#\"}" value="${value%\'}" value="${value#\'}"
  fi
  printf '%s' "$value"
}

# Nama database diambil dari path URL. URL lengkap tidak pernah dicetak karena berisi sandi.
db_name() {
  local path="${1#*://}"
  path="${path#*/}"
  printf '%s' "${path%%\?*}"
}

require_test_env() {
  local key value name
  for key in TEST_DATABASE_URL TEST_MIGRATE_DATABASE_URL; do
    value="$(env_value "$key")"
    if [[ -z "$value" || "$value" == *GANTI_SANDI* ]]; then
      fail "$key belum diisi. Salin contohnya dari backend/.env.example ke backend/.env, lalu isi sandinya."
    fi
    name="$(db_name "$value")"
    if [[ "$name" != *_test ]]; then
      fail "$key menunjuk database \"$name\". Test hanya boleh jalan di database berakhiran _test."
    fi
    export "$key=$value"
  done
}

# Host dan port dibaca dari TEST_DATABASE_URL, biasanya 127.0.0.1:15432 lewat SSH tunnel.
require_tunnel() {
  local rest hostport host port
  rest="${TEST_DATABASE_URL#*://}"
  rest="${rest##*@}"
  hostport="${rest%%/*}"
  host="${hostport%:*}"
  port="${hostport##*:}"
  [[ "$port" == "$hostport" ]] && port=5432
  if ! (exec 3<>"/dev/tcp/$host/$port") 2>/dev/null; then
    fail "database uji di $host:$port tidak terjangkau. Buka tunnel dulu: ./scripts/dev-db-tunnel.sh"
  fi
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "perintah $1 tidak ditemukan, $2."
}
