import type { Locator } from '@playwright/test'

import { E2E_ADMIN_PASSWORD } from './env'
import { expect, test } from './fixtures'
import { accountField, cardHeading, CONSENT, guestGroup, INVITE_HEADING, tabTo } from './helpers'

test.use({ viewport: { width: 320, height: 640 } })

// Cincin fokus global: outline krem 2px. Di atas kertas memakai warna tinta.
async function expectFocusRing(target: Locator) {
  await expect(target).toBeFocused()
  await expect(target).toHaveCSS('outline-style', 'solid')
  await expect(target).toHaveCSS('outline-width', '2px')
}

test('alur undangan sampai kartu akses hanya dengan keyboard di 320px', async ({ page, db }) => {
  await db.addVip('vipku')
  await page.goto('/')
  const open = page.getByRole('button', { name: 'Buka surat panggilan' })
  await tabTo(page, open)
  await expectFocusRing(open)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { level: 1, name: INVITE_HEADING })).toBeFocused()

  await tabTo(page, accountField(page))
  await expectFocusRing(accountField(page))
  await page.keyboard.type('vipku')
  await page.keyboard.press('Enter')

  const name = page
    .getByRole('group', { name: 'Data kamu' })
    .getByRole('textbox', { name: 'Nama lengkap' })
  await expect(name).toBeFocused()
  await page.keyboard.type('Vina')
  await page.keyboard.press('Tab')
  await page.keyboard.type('vina@email.com')

  const add = page.getByRole('button', { name: 'Tambah tamu' })
  await tabTo(page, add)
  await expectFocusRing(add)
  await page.keyboard.press('Enter')
  const guestName = guestGroup(page, 1).getByRole('textbox', { name: 'Nama lengkap' })
  await expect(guestName).toBeFocused()
  await page.keyboard.press('Enter')
  // Enter di kolom isian mengirim formulir: validasi di browser menahan dan memindah fokus ke kolom salah pertama.
  await expect(guestName).toBeFocused()
  await page.keyboard.type('Tono')
  await page.keyboard.press('Tab')
  await page.keyboard.type('tono@email.com')
  await page.keyboard.press('Tab')
  await page.keyboard.type('tono_nvm')

  // Tamu kedua ditambah lalu dihapus: fokus kembali ke tombol tambah tamu.
  await tabTo(page, add)
  await page.keyboard.press('Enter')
  const remove = page.getByRole('button', { name: 'Hapus Tamu 2' })
  await tabTo(page, remove)
  await page.keyboard.press('Enter')
  await expect(add).toBeFocused()

  const consent = page.getByRole('checkbox', { name: CONSENT })
  await tabTo(page, consent)
  await expectFocusRing(consent)
  await page.keyboard.press('Space')
  await expect(consent).toBeChecked()

  // Honeypot tidak boleh terjangkau Tab.
  const submit = page.getByRole('button', { name: 'Kirim konfirmasi' })
  await page.keyboard.press('Tab')
  await expect(submit).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(cardHeading(page)).toBeFocused()

  const save = page.getByRole('button', { name: 'Simpan kartu akses' })
  await tabTo(page, save)
  await expectFocusRing(save)
})

test('FAQ: Enter membuka, Escape menutup dan fokus tetap di pertanyaan', async ({ page }) => {
  await page.goto('/')
  await tabTo(page, page.getByRole('button', { name: 'Buka surat panggilan' }))
  await page.keyboard.press('Enter')
  const question = page.getByRole('button', { name: 'Ada biayanya?' })
  await tabTo(page, question)
  await expect(question).toHaveAttribute('aria-expanded', 'false')
  await page.keyboard.press('Enter')
  await expect(question).toHaveAttribute('aria-expanded', 'true')
  await expect(
    page.getByText('Tidak. Formulir ini hanya untuk mendata siapa yang datang.'),
  ).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(question).toHaveAttribute('aria-expanded', 'false')
  await expect(question).toBeFocused()
  await page.keyboard.press('Space')
  await expect(question).toHaveAttribute('aria-expanded', 'true')
})

test('admin hanya dengan keyboard di 320px: masuk, unduh, Daftar VIP, keluar', async ({ page }) => {
  await page.goto('/admin')
  const password = page.getByLabel('Sandi admin')
  await expect(password).toBeFocused()
  await page.keyboard.type(E2E_ADMIN_PASSWORD)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { name: 'Rekap pendaftaran' })).toBeFocused()

  const csv = page.getByRole('button', { name: 'Unduh CSV' })
  await tabTo(page, csv)
  await expectFocusRing(csv)
  const download = page.waitForEvent('download')
  await page.keyboard.press('Enter')
  await download

  const names = page
    .getByRole('region', { name: 'Daftar VIP' })
    .getByRole('textbox', { name: 'Nama akun', exact: true })
  await tabTo(page, names)
  await page.keyboard.type('budi')
  await tabTo(page, page.getByRole('button', { name: 'Tambah ke daftar VIP' }))
  await page.keyboard.press('Enter')
  const off = page.getByRole('button', { name: 'Nonaktifkan budi' })
  await tabTo(page, off)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: 'Aktifkan lagi budi' })).toBeVisible()

  // Keluar ada di atas daftar VIP, jadi dicapai mundur dengan Shift+Tab.
  const logout = page.getByRole('button', { name: 'Keluar' })
  await tabTo(page, logout, 80, 'Shift+Tab')
  await page.keyboard.press('Enter')
  await expect(page.getByText('Kamu sudah keluar.')).toBeVisible()
  await expect(password).toBeFocused()
})
