import type { AttendeeInput } from '@/api/types'

// Aturan dan pesan disamakan dengan backend (internal/registration/validate.go), supaya pendaftar melihat
// kalimat yang sama di mana pun kesalahannya tertangkap. Backend tetap penentu akhir.

const CONTROL = /\p{Cc}/u
// Selaras dengan mail.ParseAddress di backend untuk alamat polos: tanpa titik di awal, di akhir, atau berurutan,
// tanpa karakter khusus RFC 5322, dan domain wajib bertitik.
const ATOM = String.raw`[^\s@."(),:;<>[\]\\]+`
const EMAIL = new RegExp(`^${ATOM}(\\.${ATOM})*@${ATOM}(\\.${ATOM})+$`)

export type FieldErrors = Record<string, string>

export function cleanAttendee(a: AttendeeInput): AttendeeInput {
  return {
    full_name: a.full_name.trim(),
    email: a.email.trim().toLowerCase(),
    account_name: a.account_name.trim(),
  }
}

function textProblem(value: string, max: number, label: string): string | null {
  if (value === '') {
    return `${label} wajib diisi.`
  }
  if (CONTROL.test(value)) {
    return `${label} berisi karakter yang tidak valid.`
  }
  // Dihitung per karakter, sama seperti backend dan varchar di Postgres.
  if (Array.from(value).length > max) {
    return `${label} maksimal ${max} karakter.`
  }
  return null
}

function emailProblem(email: string): string | null {
  if (email === '') {
    return 'Email wajib diisi.'
  }
  if (email.length > 254) {
    return 'Email terlalu panjang.'
  }
  if (!EMAIL.test(email)) {
    return 'Format email belum benar, contoh: nama@email.com.'
  }
  return null
}

// Sama dengan NormalizeAccount di backend (internal/registration/account.go): bentuk inilah yang
// dicocokkan dengan daftar VIP. \u0085 ditambahkan karena Go menganggapnya spasi, sedangkan \s di JavaScript tidak.
export function normalizeAccount(value: string): string {
  return value
    .replace(/^[\s\u0085@]+/, '')
    .replace(/[\s\u0085]+$/, '')
    .toLowerCase()
}

// Langkah pertama formulir. "@" saja lolos aturan teks biasa, tapi tidak bisa dicocokkan.
export function validateAccount(raw: string): string | null {
  const value = raw.trim()
  const msg = textProblem(value, 50, 'Nama akun')
  if (msg) {
    return msg
  }
  return normalizeAccount(value) === '' ? 'Nama akun wajib diisi.' : null
}

function rowLabel(i: number): string {
  return i === 0 ? 'data kamu' : `Tamu ${i}`
}

export function validateRegistration(input: {
  attendees: AttendeeInput[]
  consent: boolean
}): FieldErrors {
  const errors: FieldErrors = {}
  const seen = new Map<string, number>()

  input.attendees.forEach((raw, i) => {
    const a = cleanAttendee(raw)
    const prefix = `attendees[${i}]`

    const nameMsg = textProblem(a.full_name, 100, 'Nama lengkap')
    if (nameMsg) errors[`${prefix}.full_name`] = nameMsg

    const emailMsg = emailProblem(a.email)
    const firstUse = seen.get(a.email)
    if (emailMsg) {
      errors[`${prefix}.email`] = emailMsg
    } else if (firstUse !== undefined) {
      errors[`${prefix}.email`] = `Email ini sudah dipakai di ${rowLabel(firstUse)}.`
    } else {
      seen.set(a.email, i)
    }

    const accountMsg = textProblem(a.account_name, 50, 'Nama akun')
    if (accountMsg) errors[`${prefix}.account_name`] = accountMsg
  })

  if (!input.consent) {
    errors.consent = 'Centang persetujuan penggunaan data untuk melanjutkan.'
  }
  return errors
}
