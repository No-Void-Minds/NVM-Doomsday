import { randomIp, expect, test, watchConsole } from './fixtures'
import {
  accountField,
  cardCode,
  cardHeading,
  CONSENT,
  fillPerson,
  guestGroup,
  openInvite,
  person,
  primaryGroup,
  registerSolo,
  registerViaApi,
  submitAccount,
  submitRegistration,
} from './helpers'
import { BASE_URL } from './env'

const VIP_NOTE = 'Nama akun ini ada di daftar VIP.'
const TAKEN = 'Nama akun ini sudah terdaftar. Hubungi admin NVM.'
const NOT_VIP = 'Nama akun ini tidak ada di daftar VIP, jadi pendaftaran hanya untuk kamu sendiri.'

test('Solo: nama akun, data, kartu akses, dan tercatat di database', async ({ page, db }) => {
  await openInvite(page)
  await submitAccount(page, 'budi_solo')
  // Setelah nama akun diterima, fokus langsung ke kolom pertama langkah berikutnya.
  await expect(primaryGroup(page).getByRole('textbox', { name: 'Nama lengkap' })).toBeFocused()
  await expect(page.getByText(VIP_NOTE)).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Tambah tamu' })).toHaveCount(0)

  await fillPerson(primaryGroup(page), { name: '  Budi Santoso ', email: ' Budi@Email.COM ' })
  await submitRegistration(page)
  await expect(cardHeading(page)).toBeFocused()
  await expect(page.getByRole('heading', { name: 'Kartu akses kamu' })).toBeVisible()
  const code = await cardCode(page)
  await expect(page.getByText('Solo', { exact: true })).toBeVisible()
  await expect(page.getByText('Budi Santoso', { exact: true })).toBeVisible()

  expect(
    await db.count(
      `SELECT count(*) FROM registrations r JOIN attendees a ON a.registration_id = r.id
       WHERE r.code = $1 AND r.type = 'solo' AND r.status = 'confirmed'
       AND a.email = 'budi@email.com' AND a.account_name = 'budi_solo'`,
      [code],
    ),
  ).toBe(1)
})

test('validasi di browser: pesan per kolom, ringkasan, fokus ke kolom salah pertama', async ({
  page,
}) => {
  await openInvite(page)
  await page.getByRole('button', { name: 'Lanjut isi data' }).click()
  await expect(page.getByText('Nama akun wajib diisi.')).toBeVisible()
  await expect(accountField(page)).toBeFocused()

  await submitAccount(page, '@@')
  await expect(page.getByText('Nama akun wajib diisi.')).toBeVisible()

  await submitAccount(page, 'budi')
  await primaryGroup(page).getByRole('textbox', { name: 'Email' }).fill('budi@email')
  await page.getByRole('button', { name: 'Kirim konfirmasi' }).click()
  await expect(page.getByText('Ada 3 isian yang perlu dicek.')).toBeVisible()
  await expect(page.getByText('Nama lengkap wajib diisi.')).toBeVisible()
  await expect(page.getByText('Format email belum benar, contoh: nama@email.com.')).toBeVisible()
  await expect(
    page.getByText('Centang persetujuan penggunaan data untuk melanjutkan.'),
  ).toBeVisible()
  await expect(primaryGroup(page).getByRole('textbox', { name: 'Nama lengkap' })).toBeFocused()
  // Pesan error terbaca sebagai deskripsi kolom oleh pembaca layar.
  await expect(
    primaryGroup(page).getByRole('textbox', { name: 'Email' }),
  ).toHaveAccessibleDescription('Format email belum benar, contoh: nama@email.com.')
})

test('VIP: tambah dan hapus tamu dengan fokus yang benar, batas tamu, kartu VIP', async ({
  page,
  db,
}) => {
  await db.addVip('VipKu')
  await openInvite(page)
  await submitAccount(page, ' @vipku ')
  await expect(page.getByText(`${VIP_NOTE} Kamu boleh membawa sampai 3 tamu.`)).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Tamu (0 dari 3)' })).toBeVisible()

  const add = page.getByRole('button', { name: 'Tambah tamu' })
  for (let n = 1; n <= 3; n++) {
    await add.click()
    await expect(guestGroup(page, n).getByRole('textbox', { name: 'Nama lengkap' })).toBeFocused()
  }
  await expect(add).toBeDisabled()
  await expect(page.getByText('Sudah maksimal 3 tamu.')).toBeVisible()

  await page.getByRole('button', { name: 'Hapus Tamu 2' }).click()
  await expect(add).toBeFocused()
  await page.getByRole('button', { name: 'Hapus Tamu 2' }).click()
  await expect(page.getByRole('heading', { name: 'Tamu (1 dari 3)' })).toBeVisible()

  await fillPerson(primaryGroup(page), person('Vina'))
  await fillPerson(guestGroup(page, 1), person('Tono', 'tono_nvm'))
  await submitRegistration(page)
  await expect(page.getByRole('heading', { name: 'Kartu akses kamu' })).toBeFocused()
  await expect(page.getByText('VIP', { exact: true })).toBeVisible()
  await expect(page.getByRole('listitem').filter({ hasText: 'Tono' })).toBeVisible()
  const code = await cardCode(page)
  expect(
    await db.count(
      `SELECT count(*) FROM attendees a JOIN registrations r ON r.id = a.registration_id
       WHERE r.code = $1 AND r.type = 'vip' AND r.vip_account_id IS NOT NULL`,
      [code],
    ),
  ).toBe(2)
})

test('ganti nama akun: kembali ke langkah 1, data tamu tetap tersimpan', async ({ page, db }) => {
  await db.addVip('vipku')
  await openInvite(page)
  await submitAccount(page, 'vipku')
  await page.getByRole('button', { name: 'Tambah tamu' }).click()
  await fillPerson(guestGroup(page, 1), person('Tono'))

  await page.getByRole('button', { name: 'Ganti nama akun' }).click()
  await expect(accountField(page)).toBeFocused()
  await expect(accountField(page)).toHaveValue('vipku')

  await submitAccount(page, 'bukan_vip')
  await expect(page.getByRole('group', { name: 'Tamu 1' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Ganti nama akun' }).click()
  await submitAccount(page, 'vipku')
  await expect(
    guestGroup(page, 1).getByRole('textbox', { name: 'Nama lengkap', exact: true }),
  ).toHaveValue('Tono')
})

test('nama VIP yang sudah dipakai ditolak di langkah 1', async ({ page, db, request }) => {
  await db.addVip('vipku')
  await registerViaApi(request, [person('vina', 'vipku')])
  await openInvite(page)
  await submitAccount(page, '@VIPKU')
  await expect(page.getByText(TAKEN)).toBeVisible()
  await expect(accountField(page)).toBeFocused()
  await expect(primaryGroup(page)).toHaveCount(0)
})

test('nama VIP dinonaktifkan admin di tengah jalan: kembali ke langkah 1 dengan pesan', async ({
  page,
  db,
}) => {
  await db.addVip('vipku')
  await openInvite(page)
  await submitAccount(page, 'vipku')
  await page.getByRole('button', { name: 'Tambah tamu' }).click()
  await fillPerson(primaryGroup(page), person('Vina'))
  await fillPerson(guestGroup(page, 1), person('Tono'))

  await db.setVipActive('vipku', false)
  await submitRegistration(page)
  await expect(page.getByText(NOT_VIP)).toBeVisible()
  await expect(accountField(page)).toBeFocused()
  expect(await db.count('SELECT count(*) FROM registrations')).toBe(0)

  await page.getByRole('button', { name: 'Lanjut isi data' }).click()
  await expect(page.getByText(VIP_NOTE)).toHaveCount(0)
})

test('dua tab dengan nama VIP sama: satu dapat kartu, yang lain kembali ke langkah 1', async ({
  browser,
  db,
}, testInfo) => {
  await db.addVip('rebutan')
  const problems: string[] = []
  const tabs = await Promise.all(
    [0, 1].map(async () => {
      const context = await browser.newContext({
        baseURL: BASE_URL,
        extraHTTPHeaders: { 'X-Forwarded-For': randomIp() },
      })
      const page = await context.newPage()
      watchConsole(page, testInfo, problems)
      await openInvite(page)
      await submitAccount(page, 'rebutan')
      await expect(page.getByText(VIP_NOTE)).toBeVisible()
      return page
    }),
  )
  const [first, second] = tabs as [(typeof tabs)[0], (typeof tabs)[0]]
  await fillPerson(primaryGroup(first), person('Pertama'))
  await fillPerson(primaryGroup(second), person('Kedua'))

  await submitRegistration(first)
  await expect(first.getByRole('heading', { name: 'Kartu akses kamu' })).toBeVisible()
  await submitRegistration(second)
  await expect(second.getByText(TAKEN)).toBeVisible()
  await expect(accountField(second)).toBeFocused()
  expect(await db.count('SELECT count(*) FROM registrations')).toBe(1)
  expect(problems).toEqual([])
  await Promise.all(tabs.map((p) => p.context().close()))
})

test('email yang sudah terdaftar: pesan di kolomnya, fokus ke sana, tanpa baris baru', async ({
  page,
  db,
  request,
}) => {
  await registerViaApi(request, [person('budi')])
  await openInvite(page)
  await submitAccount(page, 'budi2')
  await fillPerson(primaryGroup(page), { name: 'Budi Dua', email: 'BUDI@email.com' })
  await submitRegistration(page)
  const email = primaryGroup(page).getByRole('textbox', { name: 'Email' })
  await expect(email).toHaveAccessibleDescription('Email ini sudah terdaftar.')
  await expect(email).toBeFocused()
  expect(await db.count('SELECT count(*) FROM registrations')).toBe(1)
})

test('mode daftar tunggu dan peringatan rombongan lebih besar dari sisa kursi', async ({
  page,
  db,
  request,
}) => {
  await db.addVip('vipku')
  await db.setEvent({ capacity: 2 })
  await registerViaApi(request, [person('andi')])
  await openInvite(page)
  await submitAccount(page, 'vipku')
  await page.getByRole('button', { name: 'Tambah tamu' }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Sisa 1 kursi' })).toHaveText(
    'Sisa 1 kursi, rombonganmu 2 orang. Kalau tetap dikirim, kalian masuk daftar tunggu bersama.',
  )
  await page.getByRole('button', { name: 'Hapus Tamu 1' }).click()
  await expect(page.getByText('Sisa 1 kursi')).toHaveCount(0)

  await registerViaApi(request, [person('budi')])
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Masuk daftar tunggu' })).toBeVisible()
  await submitAccount(page, 'citra')
  await fillPerson(primaryGroup(page), person('Citra'))
  await page.getByRole('checkbox', { name: CONSENT }).check()
  await page.getByRole('button', { name: 'Kirim ke daftar tunggu' }).click()
  await expect(page.getByRole('heading', { name: 'Kamu di daftar tunggu' })).toBeFocused()
  await expect(page.getByText('Nomor urut 1 saat mendaftar')).toBeVisible()
})

test('jawaban hilang karena sinyal putus: kirim ulang memberi kode yang sama', async ({
  page,
  db,
  allowConsoleError,
}) => {
  allowConsoleError(/ERR_INTERNET_DISCONNECTED/)
  let dropped = false
  await page.route('**/api/registrations', async (route) => {
    if (dropped) {
      return route.continue()
    }
    dropped = true
    // Server sempat memproses kiriman pertama, tapi jawabannya tidak pernah sampai ke browser.
    await route.fetch()
    return route.abort('internetdisconnected')
  })
  await openInvite(page)
  await submitAccount(page, 'budi')
  await fillPerson(primaryGroup(page), person('Budi'))
  await submitRegistration(page)
  await expect(
    page.getByText('Tidak bisa terhubung ke server. Periksa koneksi internetmu, lalu coba lagi.'),
  ).toBeVisible()
  expect(await db.count('SELECT count(*) FROM registrations')).toBe(1)

  await page.getByRole('button', { name: 'Kirim konfirmasi' }).click()
  const code = await cardCode(page)
  expect(await db.count('SELECT count(*) FROM registrations')).toBe(1)
  expect(await db.count('SELECT count(*) FROM registrations WHERE code = $1', [code])).toBe(1)
})

test('daftarkan orang lain memakai formulir baru', async ({ page }) => {
  await openInvite(page)
  await registerSolo(page, person('Budi'))
  await page.getByRole('button', { name: 'Daftarkan orang lain' }).click()
  await expect(accountField(page)).toBeVisible()
  await registerSolo(page, person('Sari'))
  await expect(page.getByText('Sari', { exact: true })).toBeVisible()
})
