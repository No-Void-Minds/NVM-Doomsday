import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import { TestDb } from './db'
import { testDatabaseUrls } from './env'

// Skema database uji disamakan dengan migrasi terbaru sebelum backend menerima request dari test.
export default async function globalSetup() {
  const { migrator } = testDatabaseUrls()
  try {
    execFileSync('migrate', ['-path', 'migrations', '-database', migrator, 'up'], {
      cwd: fileURLToPath(new URL('../../backend', import.meta.url)),
      stdio: 'pipe',
    })
  } catch (err) {
    // Keluaran migrate bisa memuat URL, jadi sandinya diganti sebelum dicetak.
    const pw = decodeURIComponent(new URL(migrator).password)
    const out = String((err as { stderr?: Buffer }).stderr ?? err)
    throw new Error(`migrate up gagal: ${pw ? out.replaceAll(pw, '***') : out}`)
  }
  await new TestDb().reset()
}
