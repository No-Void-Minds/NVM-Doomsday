import { randomInt } from 'node:crypto'

import { test as base, expect, type Page, type TestInfo } from '@playwright/test'

import { TestDb } from './db'

// Backend E2E percaya X-Forwarded-For dari proxy Vite (TRUSTED_PROXIES=127.0.0.1). Setiap test memakai IP
// acak, supaya batas percobaan per IP dari test lain tidak ikut menggagalkannya.
export function randomIp(): string {
  return `10.${randomInt(1, 255)}.${randomInt(0, 255)}.${randomInt(1, 255)}`
}

// Jawaban 4xx dari API memang bagian alur (belum masuk admin, sandi salah, nama sudah dipakai), tapi
// Chrome tetap menulisnya ke console sebagai error. Yang itu dicatat, bukan digagalkan. Error lain gagal.
export function watchConsole(page: Page, testInfo: TestInfo, problems: string[]) {
  page.on('pageerror', (err) => problems.push(`pageerror: ${err.message}`))
  page.on('console', (msg) => {
    if (msg.type() === 'warning' && msg.text().startsWith('[Vue warn]')) {
      testInfo.annotations.push({ type: 'vue warn', description: msg.text().slice(0, 300) })
      return
    }
    if (msg.type() !== 'error') {
      return
    }
    const url = msg.location().url
    if (
      /^Failed to load resource: the server responded with a status of 4\d\d/.test(msg.text()) &&
      url.includes('/api/')
    ) {
      testInfo.annotations.push({ type: 'api 4xx', description: url })
      return
    }
    problems.push(`console.error: ${msg.text()} (${url})`)
  })
}

interface Fixtures {
  db: TestDb
  clientIp: string
  // Untuk test yang sengaja memicu respons gagal, misalnya 502 dari proxy. Selain pola ini tetap gagal.
  allowConsoleError: (pattern: RegExp) => void
  consoleAllowed: RegExp[]
}

export const test = base.extend<Fixtures>({
  clientIp: async ({}, use) => use(randomIp()),
  extraHTTPHeaders: async ({ clientIp }, use) => use({ 'X-Forwarded-For': clientIp }),
  db: [
    async ({}, use) => {
      const db = new TestDb()
      await db.reset()
      await use(db)
    },
    { auto: true },
  ],
  consoleAllowed: async ({}, use) => use([]),
  allowConsoleError: async ({ consoleAllowed }, use) =>
    use((pattern) => {
      consoleAllowed.push(pattern)
    }),
  page: async ({ page, consoleAllowed }, use, testInfo) => {
    const problems: string[] = []
    watchConsole(page, testInfo, problems)
    await use(page)
    const unexpected = problems.filter((p) => !consoleAllowed.some((re) => re.test(p)))
    expect(unexpected, 'console tanpa error').toEqual([])
  },
})

export { expect }
