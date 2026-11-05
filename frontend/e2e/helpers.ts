import { randomUUID } from 'node:crypto'

import type { APIRequestContext, Locator, Page } from '@playwright/test'

import { E2E_ADMIN_PASSWORD } from './env'
import { expect } from './fixtures'

export const INVITE_HEADING = 'Kamu diundang nonton bareng.'
export const CONSENT = 'Data ini hanya dipakai untuk acara nonton bareng ini.'

export interface Person {
  name: string
  email: string
  account?: string
}

export function person(name: string, account = name): Person {
  return { name, email: `${name.toLowerCase().replace(/\s+/g, '.')}@email.com`, account }
}

// Amplop dilewati lewat sessionStorage, sama seperti pengunjung yang sudah membukanya di sesi ini.
export async function openInvite(page: Page, path = '/') {
  await page.addInitScript(() => sessionStorage.setItem('doomsday_opened', '1'))
  await page.goto(path)
  await expect(page.getByRole('heading', { level: 1, name: INVITE_HEADING })).toBeVisible()
}

export const accountField = (page: Page) =>
  page.getByRole('textbox', { name: 'Nama akun', exact: true })

export async function submitAccount(page: Page, account: string) {
  await accountField(page).fill(account)
  await page.getByRole('button', { name: 'Lanjut isi data' }).click()
}

export const primaryGroup = (page: Page) => page.getByRole('group', { name: 'Data kamu' })
export const guestGroup = (page: Page, n: number) =>
  page.getByRole('group', { name: `Tamu ${n}`, exact: true })

export async function fillPerson(scope: Locator, p: Person) {
  await scope.getByRole('textbox', { name: 'Nama lengkap', exact: true }).fill(p.name)
  await scope.getByRole('textbox', { name: 'Email', exact: true }).fill(p.email)
  const account = scope.getByRole('textbox', { name: 'Nama akun', exact: true })
  if (p.account !== undefined && (await account.count()) > 0) {
    await account.fill(p.account)
  }
}

export async function submitRegistration(page: Page) {
  await page.getByRole('checkbox', { name: CONSENT }).check()
  await page.getByRole('button', { name: /^Kirim (konfirmasi|ke daftar tunggu)$/ }).click()
}

export const cardHeading = (page: Page) =>
  page.getByRole('heading', { name: /^(Kartu akses kamu|Kamu di daftar tunggu)$/ })

export async function cardCode(page: Page): Promise<string> {
  const code = page.getByText(/^\s*NVM-DD-[A-HJKMNP-Z2-9]{6}\s*$/)
  await expect(code).toBeVisible()
  return (await code.textContent())?.trim() ?? ''
}

// Seluruh pendaftaran Solo dari langkah nama akun sampai kartu akses.
export async function registerSolo(page: Page, p: Person): Promise<string> {
  await submitAccount(page, p.account ?? p.name)
  await fillPerson(primaryGroup(page), p)
  await submitRegistration(page)
  await expect(cardHeading(page)).toBeFocused()
  return cardCode(page)
}

// Untuk menyiapkan keadaan lewat API asli, dengan header yang sama dengan halaman (termasuk IP uji).
export async function registerViaApi(request: APIRequestContext, people: Person[]) {
  const res = await request.post('/api/registrations', {
    data: {
      attendees: people.map((p) => ({
        full_name: p.name,
        email: p.email,
        account_name: p.account ?? p.name,
      })),
      consent: true,
      idempotency_key: randomUUID(),
      website: '',
    },
  })
  expect(res.status(), await res.text()).toBe(201)
  return (await res.json()) as { code: string; status: string; type: string }
}

export async function loginAdmin(page: Page) {
  await page.goto('/admin')
  await page.getByLabel('Sandi admin').fill(E2E_ADMIN_PASSWORD)
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Rekap pendaftaran' })).toBeFocused()
}

// Menekan Tab sampai target terfokus. Gagal kalau target tidak terjangkau keyboard.
export async function tabTo(page: Page, target: Locator, max = 80, key = 'Tab') {
  for (let i = 0; i < max; i++) {
    if (await target.evaluate((el) => el === document.activeElement).catch(() => false)) {
      return
    }
    await page.keyboard.press(key)
  }
  throw new Error(`tidak terjangkau dengan ${key} dalam ${max} langkah: ${target}`)
}
