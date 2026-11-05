import { readFileSync } from 'node:fs'

import { describe, expect, it } from 'vitest'

import type { AttendeeInput } from '@/api/types'
import {
  cleanAttendee,
  normalizeAccount,
  validateAccount,
  validateRegistration,
} from '@/lib/validation'

// File kasus yang sama dibaca test Go di backend/internal/registration, jadi kedua sisi diuji dengan data
// yang persis sama. Kalau salah satu sisi berubah, test di sisi lain ikut gagal.
function readShared<T>(name: string): T {
  return JSON.parse(readFileSync(new URL(`../../../../testdata/${name}`, import.meta.url), 'utf8'))
}

interface ValidationCase {
  nama: string
  attendees: AttendeeInput[]
  consent: boolean
  errors: Record<string, string>
}

const validation = readShared<{ kasus: ValidationCase[] }>('validasi-pendaftaran.json').kasus
interface NormalizationCase {
  input: string
  hasil: string
  catatan?: string
  // Hanya diuji di Go. Alasannya tertulis di field alasan pada file kasus.
  go_only?: boolean
  alasan?: string
}

const normalization = readShared<{ kasus: NormalizationCase[] }>('normalisasi-nama-akun.json').kasus

describe('validateRegistration sama dengan Validate di backend', () => {
  it.each(validation.map((c) => [c.nama, c] as const))('%s', (_, c) => {
    expect(validateRegistration({ attendees: c.attendees, consent: c.consent })).toEqual(c.errors)
  })
})

// Banyak kasus berisi karakter tak terlihat, jadi nama test ditulis sebagai escape \uXXXX.
const visible = (s: string) =>
  JSON.stringify(s).replace(
    /[^ -~]/g,
    (ch) => `\\u${ch.charCodeAt(0).toString(16).padStart(4, '0')}`,
  )

describe('normalizeAccount sama dengan NormalizeAccount di backend', () => {
  const shared = normalization.filter((c) => !c.go_only)
  it.each(shared.map((c) => [visible(c.input), c] as const))('%s', (_, c) => {
    expect(normalizeAccount(c.input)).toBe(c.hasil)
  })

  // Dilewati, bukan dihapus, supaya tetap terlihat di laporan Vitest beserta alasannya.
  it.skip.each(normalization.filter((c) => c.go_only).map((c) => [visible(c.input), c.alasan]))(
    '%s (khusus Go: %s)',
    () => {},
  )
})

describe('validateAccount', () => {
  // Pasangan input dan pesan sama dengan TestValidateAccount di backend.
  it.each([
    ['', 'Nama akun wajib diisi.'],
    ['   ', 'Nama akun wajib diisi.'],
    ['@', 'Nama akun wajib diisi.'],
    ['@@', 'Nama akun wajib diisi.'],
    ['a'.repeat(51), 'Nama akun maksimal 50 karakter.'],
    ['bu\u0007di', 'Nama akun berisi karakter yang tidak valid.'],
  ])('%j ditolak', (input, message) => {
    expect(validateAccount(input)).toBe(message)
  })

  it.each([' budi ', ' @Budi ', 'a'.repeat(50), '\u00e9'.repeat(50)])('%j lolos', (input) => {
    expect(validateAccount(input)).toBeNull()
  })
})

describe('panjang dihitung per karakter seperti backend', () => {
  const base: AttendeeInput = { full_name: 'Budi', email: 'budi@email.com', account_name: 'budi' }
  const check = (a: Partial<AttendeeInput>) =>
    validateRegistration({ attendees: [{ ...base, ...a }], consent: true })

  it('nama 100 huruf beraksen lolos, 101 ditolak', () => {
    expect(check({ full_name: '\u00e9'.repeat(100) })).toEqual({})
    expect(check({ full_name: 'a'.repeat(101) })).toEqual({
      'attendees[0].full_name': 'Nama lengkap maksimal 100 karakter.',
    })
  })

  it('emoji dihitung satu karakter, bukan dua unit UTF-16', () => {
    expect(check({ full_name: '\u{1F600}'.repeat(100) })).toEqual({})
  })

  it('nama akun 51 ditolak', () => {
    expect(check({ account_name: 'a'.repeat(51) })).toEqual({
      'attendees[0].account_name': 'Nama akun maksimal 50 karakter.',
    })
  })

  it('email 255 karakter terlalu panjang', () => {
    const email = `${'a'.repeat(64)}@${'b'.repeat(186)}.com`
    expect(email).toHaveLength(255)
    expect(check({ email })).toEqual({ 'attendees[0].email': 'Email terlalu panjang.' })
  })
})

describe('cleanAttendee', () => {
  it('memangkas spasi dan mengecilkan email, nama akun tetap seperti diketik', () => {
    expect(
      cleanAttendee({ full_name: '  Budi ', email: ' Budi@Email.COM ', account_name: ' @Budi ' }),
    ).toEqual({ full_name: 'Budi', email: 'budi@email.com', account_name: '@Budi' })
  })
})
