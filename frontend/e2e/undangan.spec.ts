import type { Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { INVITE_HEADING, openInvite, person, registerViaApi } from './helpers'

const seats = (page: Page, capacity: number) =>
  page.locator('p', { hasText: `dari ${capacity} kursi tersisa` })

test.describe('amplop pembuka', () => {
  test('menyapa nama dari ?to=, animasi sekitar 600 ms, lalu fokus ke judul undangan', async ({
    page,
  }) => {
    await page.goto('/?to=%20%20Budi%E2%80%AE%20%20Santoso%20')
    await expect(
      page.getByRole('heading', { level: 1, name: 'Ada surat panggilan untuk Budi Santoso.' }),
    ).toBeVisible()

    const open = page.getByRole('button', { name: 'Buka surat panggilan' })
    const started = Date.now()
    await open.click()
    // Selama animasi tombol dikunci, supaya ketukan ganda tidak membuka dua kali.
    await expect(open).toBeDisabled()
    const heading = page.getByRole('heading', { level: 1, name: INVITE_HEADING })
    await expect(heading).toBeVisible()
    const elapsed = Date.now() - started
    expect(elapsed).toBeGreaterThanOrEqual(550)
    expect(elapsed).toBeLessThan(3000)
    await expect(heading).toBeFocused()
  })

  test('tanpa animasi saat gerak dikurangi', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/')
    const started = Date.now()
    await page.getByRole('button', { name: 'Buka surat panggilan' }).click()
    const heading = page.getByRole('heading', { level: 1, name: INVITE_HEADING })
    await expect(heading).toBeFocused()
    expect(Date.now() - started).toBeLessThan(500)
  })

  test('diingat per sesi: muat ulang tidak menampilkan amplop, tab baru menampilkannya lagi', async ({
    page,
    context,
  }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/')
    await page.getByRole('button', { name: 'Buka surat panggilan' }).click()
    await page.reload()
    await expect(page.getByRole('heading', { level: 1, name: INVITE_HEADING })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Buka surat panggilan' })).toHaveCount(0)

    // sessionStorage tidak dibagi antar tab.
    const other = await context.newPage()
    await other.goto('/')
    await expect(other.getByRole('button', { name: 'Buka surat panggilan' })).toBeVisible()
  })

  test('nama ganda di ?to= diabaikan', async ({ page }) => {
    await page.goto('/?to=Budi&to=Sari')
    await expect(
      page.getByRole('heading', { level: 1, name: 'Ada surat panggilan untukmu.' }),
    ).toBeVisible()
  })
})

test.describe('lembar acara dan panel kursi', () => {
  test('data kosong tampil Menyusul, jadwal dan kapasitas dari database', async ({ page, db }) => {
    await openInvite(page)
    await expect(page.getByText('Menyusul', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('Jadwal tayang diumumkan menyusul.')).toBeVisible()
    await expect(page.getByText('200 kursi', { exact: true })).toBeVisible()

    await db.setEvent({ show_at: '2026-12-18 19:00+07', gather_at: '2026-12-18 18:15+07' })
    await page.reload()
    await expect(page.getByText('Jumat, 18 Desember 2026', { exact: true })).toBeVisible()
    await expect(page.getByText('19.00 WIB', { exact: true })).toBeVisible()
    await expect(page.getByText('18.15 WIB', { exact: true })).toBeVisible()
  })

  test('sebagian terisi, penuh dengan daftar tunggu, dan ditutup', async ({
    page,
    db,
    request,
  }) => {
    await db.setEvent({ capacity: 3 })
    await registerViaApi(request, [person('andi')])
    await openInvite(page)
    await expect(seats(page, 3)).toHaveText('2 dari 3 kursi tersisa')

    await registerViaApi(request, [person('budi')])
    await registerViaApi(request, [person('citra')])
    await registerViaApi(request, [person('dedi')])
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Kursi utama sudah penuh.' })).toBeVisible()
    await expect(page.getByText('1 orang di daftar tunggu')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Masuk daftar tunggu' })).toBeVisible()

    await db.setEvent({ is_open: false })
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Pendaftaran sudah ditutup.' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Lanjut isi data' })).toHaveCount(0)
  })

  test('sisa kursi diperbarui saat tab kembali terlihat', async ({ page, request }) => {
    await openInvite(page)
    await expect(seats(page, 200)).toHaveText('200 dari 200 kursi tersisa')
    await registerViaApi(request, [person('andi')])
    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await expect(seats(page, 200)).toHaveText('199 dari 200 kursi tersisa')
  })

  test('detail acara gagal dimuat, lalu pulih dengan Coba lagi', async ({
    page,
    allowConsoleError,
  }) => {
    // 502 sengaja dibuat test ini, jadi baris console dari Chrome untuk respons itu diizinkan.
    allowConsoleError(/status of 502/)
    let fail = true
    await page.route('**/api/event', (route) =>
      fail ? route.fulfill({ status: 502, body: 'Bad Gateway' }) : route.continue(),
    )
    await openInvite(page)
    await expect(page.getByRole('alert')).toContainText('Detail acara gagal dimuat.')
    fail = false
    await page.getByRole('button', { name: 'Coba lagi' }).click()
    await expect(page.getByText('dari 200 kursi tersisa')).toBeVisible()
  })
})
