import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const ENV_FILE = fileURLToPath(new URL('../../backend/.env', import.meta.url))

// Nilai dari env didahulukan (diisi scripts/test-e2e.sh). Kalau kosong, dibaca dari backend/.env supaya
// npx playwright test juga bisa dijalankan langsung. Hanya kunci yang diminta yang diambil.
function lookup(key: string): string {
  const fromEnv = process.env[key]
  if (fromEnv) {
    return fromEnv
  }
  try {
    for (const line of readFileSync(ENV_FILE, 'utf8').split(/\r?\n/)) {
      const m = /^\s*([A-Z_]+)\s*=\s*(.*)$/.exec(line)
      if (m?.[1] === key) {
        return (m[2] ?? '').trim().replace(/^['"]|['"]$/g, '')
      }
    }
  } catch {
    // Tidak ada backend/.env, pesan di bawah yang menjelaskan.
  }
  return ''
}

// Hanya nama database yang disebut di pesan, URL tidak pernah dicetak karena berisi sandi.
export function requireTestDatabase(url: string, key: string): string {
  if (url === '') {
    throw new Error(`${key} belum diisi di env atau backend/.env`)
  }
  let name: string
  try {
    name = decodeURIComponent(new URL(url).pathname.replace(/^\//, ''))
  } catch {
    throw new Error(`${key} tidak bisa dibaca sebagai URL`)
  }
  if (!name.endsWith('_test')) {
    throw new Error(
      `${key}: database "${name}" tidak berakhiran _test. E2E menolak jalan supaya data asli tidak terhapus.`,
    )
  }
  return url
}

export function testDatabaseUrls(): { app: string; migrator: string } {
  return {
    app: requireTestDatabase(lookup('TEST_DATABASE_URL'), 'TEST_DATABASE_URL'),
    migrator: requireTestDatabase(lookup('TEST_MIGRATE_DATABASE_URL'), 'TEST_MIGRATE_DATABASE_URL'),
  }
}

// Sandi admin khusus E2E, bukan sandi dev atau produksi. Hash dibuat dengan doomsday-admin hash-password.
export const E2E_ADMIN_PASSWORD = 'sandi-e2e-doomsday-2026'
export const E2E_ADMIN_PASSWORD_HASH =
  '$2a$12$pTLYpstZDf8qWXM7U3Sulug6wXECxBAzkzsdo06IZfc83AIK9hLLy'

export const BASE_URL = 'http://127.0.0.1:5174'
export const BACKEND_URL = 'http://127.0.0.1:8081'
