import { readFile } from 'node:fs/promises'

import { expect, test } from './fixtures'
import {
  accountField,
  cardHeading,
  INVITE_HEADING,
  openInvite,
  person,
  registerSolo,
} from './helpers'

test('simpan kartu sebagai PNG lewat unduhan, gambarnya juga ditampilkan', async ({
  page,
}, testInfo) => {
  // Menu bagikan hanya ada di HTTPS. Di LAN lewat HTTP, kartu disimpan lewat unduhan biasa.
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'canShare', { value: undefined, configurable: true })
  })
  await openInvite(page)
  const code = await registerSolo(page, person('Budi'))

  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Simpan kartu akses' }).click()
  const file = await download
  expect(file.suggestedFilename()).toBe(`nvm-doomsday-${code}.png`)
  const path = testInfo.outputPath('kartu.png')
  await file.saveAs(path)
  const png = await readFile(path)
  expect(png.subarray(0, 8)).toEqual(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]))
  // Lebar dan tinggi di header IHDR. pixelRatio 2, jadi minimal dua kali lebar kartu di layar.
  expect(png.readUInt32BE(16)).toBeGreaterThan(400)
  expect(png.readUInt32BE(20)).toBeGreaterThan(400)
  await testInfo.attach('kartu-akses.png', { path, contentType: 'image/png' })

  await expect(
    page.getByRole('img', { name: 'Gambar kartu akses yang siap disimpan' }),
  ).toBeVisible()
  await expect(page.getByText('Gambar kartu gagal dibuat.')).toHaveCount(0)
})

test('simpan kartu lewat menu bagikan bila tersedia', async ({ page }) => {
  await page.addInitScript(() => {
    const shared: string[] = []
    Object.assign(window, { __shared: shared })
    Object.defineProperty(navigator, 'canShare', { value: () => true, configurable: true })
    Object.defineProperty(navigator, 'share', {
      value: async (data: { files: File[] }) => {
        shared.push(...data.files.map((f) => `${f.name}|${f.type}|${f.size}`))
      },
      configurable: true,
    })
  })
  await openInvite(page)
  const code = await registerSolo(page, person('Budi'))
  await page.getByRole('button', { name: 'Simpan kartu akses' }).click()
  await expect(page.getByRole('button', { name: 'Simpan kartu akses' })).toBeEnabled()
  const shared = await page.evaluate(() => (window as unknown as { __shared: string[] }).__shared)
  expect(shared).toHaveLength(1)
  expect(shared[0]).toMatch(new RegExp(`^nvm-doomsday-${code}\\.png\\|image/png\\|\\d{4,}$`))
})

test('salin kode registrasi', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await openInvite(page)
  const code = await registerSolo(page, person('Budi'))
  await page.getByRole('button', { name: 'Salin kode' }).click()
  await expect(page.getByRole('button', { name: 'Kode tersalin' })).toBeVisible()
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(code)
})

test('kartu dipulihkan dari localStorage saat link dibuka lagi, tanpa amplop', async ({
  page,
  context,
}) => {
  await openInvite(page)
  const code = await registerSolo(page, person('Budi'))

  // Tab baru tidak punya sessionStorage, jadi amplop hanya dilewati karena kartu tersimpan.
  const again = await context.newPage()
  await again.goto('/')
  await expect(again.getByRole('heading', { level: 1, name: INVITE_HEADING })).toBeVisible()
  await expect(cardHeading(again)).toBeVisible()
  await expect(again.getByText(code, { exact: true })).toBeVisible()
  await expect(again.getByText('Kartu ini tersimpan di perangkat ini sejak')).toBeVisible()
})

test('isi localStorage yang rusak diabaikan, formulir tetap tampil', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem(
      'doomsday_card',
      JSON.stringify({ result: { code: 1, type: 'admin' }, savedAt: 'x' }),
    )
  })
  await openInvite(page)
  await expect(accountField(page)).toBeVisible()

  await page.addInitScript(() => localStorage.setItem('doomsday_card', '{bukan json'))
  await page.reload()
  await expect(accountField(page)).toBeVisible()
})
