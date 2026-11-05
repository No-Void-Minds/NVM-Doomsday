import type { APIRequestContext, Page } from '@playwright/test'

import type { TestDb } from './db'
import { E2E_ADMIN_PASSWORD } from './env'
import { expect } from './fixtures'
import {
  fillPerson,
  loginAdmin,
  openInvite,
  person,
  primaryGroup,
  registerSolo,
  registerViaApi,
  submitAccount,
} from './helpers'

interface StateContext {
  page: Page
  db: TestDb
  request: APIRequestContext
}

// Keadaan layar yang diperiksa di setiap lebar. Masing-masing dimulai dari database kosong.
export const states: Record<string, (c: StateContext) => Promise<void>> = {
  amplop: async ({ page }) => {
    await page.goto('/?to=Budi%20Santoso')
    await expect(page.getByRole('button', { name: 'Buka surat panggilan' })).toBeVisible()
  },
  undangan: async ({ page, db }) => {
    await db.setEvent({ show_at: '2026-12-18 19:00+07', gather_at: '2026-12-18 18:15+07' })
    await openInvite(page)
    await expect(page.getByRole('button', { name: 'Lanjut isi data' })).toBeVisible()
  },
  'formulir-vip-error': async ({ page, db }) => {
    await db.addVip('vipku')
    await openInvite(page)
    await submitAccount(page, 'vipku')
    await page.getByRole('button', { name: 'Tambah tamu' }).click()
    await fillPerson(primaryGroup(page), person('Vina'))
    await page.getByRole('button', { name: 'Kirim konfirmasi' }).click()
    await expect(page.getByText(/Ada \d isian yang perlu dicek\./)).toBeVisible()
  },
  'kartu-tunggu': async ({ page, db, request }) => {
    await db.setEvent({ capacity: 1 })
    await registerViaApi(request, [person('andi')])
    await openInvite(page)
    await registerSolo(page, person('Budi Santoso Wicaksono Panjang Sekali Namanya'))
    await expect(page.getByText('Nomor urut 1 saat mendaftar')).toBeVisible()
  },
  'admin-masuk': async ({ page }) => {
    await page.goto('/admin')
    await page.getByLabel('Sandi admin').fill(`${E2E_ADMIN_PASSWORD}-salah`)
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()
    await expect(page.getByText('Sandi salah. Coba lagi.')).toBeVisible()
  },
  'admin-rekap': async ({ page, db, request }) => {
    await db.addVip('vipku', 'nama_akun_vip_yang_cukup_panjang_untuk_layar_kecil')
    await registerViaApi(request, [person('vina', 'vipku')])
    await loginAdmin(page)
    await expect(page.getByText('Dipakai NVM-DD-', { exact: false })).toBeVisible()
  },
  'tidak-ditemukan': async ({ page }) => {
    await page.goto('/halaman-yang-tidak-ada')
    await expect(page.getByRole('heading', { name: 'Halaman tidak ditemukan.' })).toBeVisible()
  },
}

// Kontrol yang bisa ditekan atau difokus. Checkbox di dalam label diukur dari labelnya, karena label itulah
// bidang sentuhnya. Elemen di dalam aria-hidden (honeypot) dan yang tidak terlihat dilewati.
export async function smallTargets(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const selector =
      'a[href], button, input:not([type="hidden"]), select, textarea, summary, [role="button"], [tabindex]:not([tabindex="-1"])'
    const out: string[] = []
    for (const el of Array.from(document.querySelectorAll<HTMLElement>(selector))) {
      if (el.closest('[aria-hidden="true"]')) {
        continue
      }
      const style = getComputedStyle(el)
      if (style.display === 'none' || style.visibility === 'hidden') {
        continue
      }
      const input = el instanceof HTMLInputElement ? el : null
      const label = input && ['checkbox', 'radio'].includes(input.type) ? el.closest('label') : null
      const rect = (label ?? el).getBoundingClientRect()
      if (rect.width === 0 && rect.height === 0) {
        continue
      }
      if (rect.width < 43.5 || rect.height < 43.5) {
        const name = (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 40)
        out.push(
          `<${el.tagName.toLowerCase()}> "${name}" ${rect.width.toFixed(1)}x${rect.height.toFixed(1)}`,
        )
      }
    }
    return out
  })
}

export async function horizontalOverflow(page: Page): Promise<number> {
  return page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )
}
