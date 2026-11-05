import { readFile } from 'node:fs/promises'

import type { Page } from '@playwright/test'

import { E2E_ADMIN_PASSWORD } from './env'
import { expect, test } from './fixtures'
import { loginAdmin, person, registerViaApi } from './helpers'

const passwordField = (page: Page) => page.getByLabel('Sandi admin')
const loginButton = (page: Page) => page.getByRole('button', { name: 'Masuk', exact: true })

test('masuk: kosong, sandi salah, lalu benar dan fokus ke rekap', async ({ page }) => {
  await page.goto('/admin')
  await expect(passwordField(page)).toBeFocused()
  await loginButton(page).click()
  await expect(page.getByText('Isi sandi admin.')).toBeVisible()

  await passwordField(page).fill('sandi-yang-salah')
  await loginButton(page).click()
  await expect(page.getByText('Sandi salah. Coba lagi.')).toBeVisible()
  await expect(passwordField(page)).toBeFocused()

  await passwordField(page).fill(E2E_ADMIN_PASSWORD)
  await loginButton(page).click()
  await expect(page.getByRole('heading', { name: 'Rekap pendaftaran' })).toBeFocused()
  await expect(
    page.getByText('Belum ada peserta. File unduhan hanya berisi judul kolom.'),
  ).toBeVisible()

  // Cookie 8 jam: muat ulang langsung ke rekap tanpa sandi.
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Rekap pendaftaran' })).toBeVisible()
})

test('batas percobaan: setelah 5 sandi salah, percobaan berikutnya ditahan', async ({ page }) => {
  await page.goto('/admin')
  for (let i = 0; i < 5; i++) {
    await passwordField(page).fill(`salah-${i}`)
    await loginButton(page).click()
    await expect(page.getByText('Sandi salah. Coba lagi.')).toBeVisible()
  }
  await passwordField(page).fill(E2E_ADMIN_PASSWORD)
  await loginButton(page).click()
  await expect(
    page.getByText('Terlalu banyak percobaan. Tunggu sebentar, lalu coba lagi.'),
  ).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Rekap pendaftaran' })).toHaveCount(0)
})

test('rekap cocok dengan database', async ({ page, db, request }) => {
  await db.addVip('vipa')
  await db.setEvent({ capacity: 3 })
  await registerViaApi(request, [person('a0', 'vipa'), person('a1')])
  await registerViaApi(request, [person('b0')])
  await registerViaApi(request, [person('c0')])
  await db.query(`UPDATE registrations SET status = 'cancelled' WHERE id IN
    (SELECT registration_id FROM attendees WHERE email = 'b0@email.com')`)

  await loginAdmin(page)
  const row = (label: string) => page.locator('dl > div').filter({ hasText: label })
  await expect(row('Terkonfirmasi')).toContainText('2 dari 3 peserta')
  await expect(row('Daftar tunggu')).toContainText('1 peserta')
  await expect(row('Batal')).toContainText('1 registrasi')
})

test('unduh Excel dan CSV dengan nama file dari server', async ({ page, request }, testInfo) => {
  await registerViaApi(request, [{ name: '=1+1', email: 'rumus@email.com', account: '@uji2' }])
  await loginAdmin(page)

  for (const [label, ext] of [
    ['Unduh Excel', 'xlsx'],
    ['Unduh CSV', 'csv'],
  ] as const) {
    const download = page.waitForEvent('download')
    await page.getByRole('button', { name: label }).click()
    const file = await download
    expect(file.suggestedFilename()).toMatch(
      new RegExp(`^nvm-doomsday-peserta-\\d{8}-\\d{4}\\.${ext}$`),
    )
    const path = testInfo.outputPath(file.suggestedFilename())
    await file.saveAs(path)
    const bytes = await readFile(path)
    if (ext === 'xlsx') {
      // XLSX adalah arsip ZIP. Error JSON yang tersimpan sebagai .xlsx akan diawali "{".
      expect(bytes.subarray(0, 2).toString()).toBe('PK')
    } else {
      expect(bytes.subarray(0, 3)).toEqual(Buffer.from([0xef, 0xbb, 0xbf]))
      const text = bytes.subarray(3).toString('utf8')
      expect(text.startsWith('Kode registrasi,Status,')).toBe(true)
      expect(text).toContain("'=1+1")
      expect(text).toContain("'@uji2")
      expect(text).toContain('\r\n')
    }
  }
})

test('sesi habis saat mengunduh: kembali ke form masuk, tanpa file rusak', async ({
  page,
  context,
}) => {
  await loginAdmin(page)
  let downloaded = false
  page.on('download', () => {
    downloaded = true
  })
  await context.clearCookies()
  await page.getByRole('button', { name: 'Unduh CSV' }).click()
  await expect(page.getByText('Sesi admin berakhir. Masuk lagi.')).toBeVisible()
  await expect(passwordField(page)).toBeFocused()
  await page.waitForTimeout(500)
  expect(downloaded).toBe(false)
})

test('keluar menghapus sesi', async ({ page, context }) => {
  await loginAdmin(page)
  await page.getByRole('button', { name: 'Keluar' }).click()
  await expect(page.getByText('Kamu sudah keluar.')).toBeVisible()
  await expect(passwordField(page)).toBeFocused()
  expect((await context.cookies()).filter((c) => c.name === 'doomsday_admin')).toEqual([])
  await page.reload()
  await expect(passwordField(page)).toBeVisible()
})

test('Daftar VIP: tempel dari Excel dengan tab, laporan, cari, nonaktifkan', async ({
  page,
  db,
}) => {
  await loginAdmin(page)
  const panel = page.getByRole('region', { name: 'Daftar VIP' })
  await expect(
    panel.getByText('Belum ada nama VIP. Semua pendaftar terbaca Solo sampai daftar ini diisi.'),
  ).toBeVisible()

  // Tiga kolom tersalin dari Excel: hanya kolom pertama yang dipakai.
  const textarea = panel.getByRole('textbox', { name: 'Nama akun', exact: true })
  await textarea.fill('budi\tBudi Santoso\t0812\r\n@Sari\tSari\r\n\r\n@\r\nBUDI\r\n')
  await panel.getByRole('button', { name: 'Tambah ke daftar VIP' }).click()
  const report = panel.getByRole('status').filter({ hasText: 'Ditambah' })
  await expect(report).toContainText('Ditambah: 2 nama.')
  await expect(report).toContainText(
    'Ditolak: 1 nama. Perbaiki di kotak di atas, lalu tambah lagi.',
  )
  await expect(report).toContainText('@: Nama akun wajib diisi.')
  // Hanya baris yang ditolak yang tersisa di kotak, supaya admin tinggal membetulkannya.
  await expect(textarea).toHaveValue('@')
  await expect(panel.getByText('2 aktif dari 2 nama')).toBeVisible()
  expect(await db.count('SELECT count(*) FROM vip_accounts')).toBe(2)

  await panel.getByRole('textbox', { name: 'Cari nama akun' }).fill(' @SAR')
  // Laporan nama yang ditolak juga berupa daftar, jadi baris nama VIP dikenali dari tombolnya.
  const rows = panel.getByRole('listitem').filter({ has: page.getByRole('button') })
  await expect(rows).toHaveCount(1)
  await expect(rows).toContainText('@Sari')
  await expect(rows).toContainText('Belum dipakai')

  // Nama akun ikut dibacakan, supaya pembaca layar tahu tombol ini untuk nama yang mana.
  await panel.getByRole('button', { name: 'Nonaktifkan @Sari' }).click()
  await expect(panel.getByRole('button', { name: 'Aktifkan lagi @Sari' })).toBeVisible()
  await expect(rows).toContainText('Nonaktif')
  expect(
    await db.count(
      `SELECT count(*) FROM vip_accounts WHERE account_key = 'sari' AND NOT is_active`,
    ),
  ).toBe(1)

  await panel.getByRole('textbox', { name: 'Cari nama akun' }).fill('tidak-ada')
  await expect(panel.getByText('Tidak ada nama yang cocok dengan pencarian.')).toBeVisible()
})

test('Daftar VIP menampilkan kode registrasi yang memakai nama itu', async ({
  page,
  db,
  request,
}) => {
  await db.addVip('vipku')
  const reg = await registerViaApi(request, [person('vina', 'vipku')])
  await loginAdmin(page)
  await expect(
    page.getByRole('region', { name: 'Daftar VIP' }).getByText(`Dipakai ${reg.code}`),
  ).toBeVisible()
})
