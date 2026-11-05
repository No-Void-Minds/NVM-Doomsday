// Kontrak dengan backend (SRS bagian 7). Nama field mengikuti JSON dari Go, jadi memakai snake_case.

export interface EventInfo {
  title: string
  film_title: string
  cinema_name: string | null
  city: string | null
  studio: string | null
  // ISO 8601 dalam UTC. Ditampilkan sebagai WIB oleh frontend.
  gather_at: string | null
  show_at: string | null
  map_url: string | null
  capacity: number
  max_guests_vip: number
  is_open: boolean
  seats_left: number
  waitlisted_attendees: number
}

export type RegistrationType = 'solo' | 'vip'
export type RegistrationStatus = 'confirmed' | 'waitlisted'

export interface AttendeeInput {
  full_name: string
  email: string
  account_name: string
}

// Tipe Solo atau VIP tidak dikirim. Server menentukannya dari nama akun pendaftar (attendees[0]).
export interface RegistrationRequest {
  // Indeks 0 selalu pendaftar, sisanya tamu.
  attendees: AttendeeInput[]
  consent: boolean
  idempotency_key: string
  // Honeypot, selalu dikirim kosong oleh formulir asli.
  website: string
}

// Hasil langkah pertama formulir. Hanya petunjuk tampilan, karena server memutuskan ulang saat mendaftar.
export interface AccountCheckResult {
  type: RegistrationType
}

export interface RegistrationResult {
  code: string
  type: RegistrationType
  status: RegistrationStatus
  waitlist_position: number | null
  attendees: string[]
}

export interface AdminSummary {
  capacity: number
  confirmed_attendees: number
  waitlisted_attendees: number
  cancelled_registrations: number
}

// Kode dari backend, ditambah dua kode yang hanya muncul di sisi browser (network_error, timeout).
export type ErrorCode =
  | 'validation_failed'
  | 'not_found'
  | 'registration_closed'
  | 'duplicate_email'
  | 'account_taken'
  | 'rate_limited'
  | 'internal_error'
  | 'unauthorized'
  | 'invalid_password'
  | 'network_error'
  | 'timeout'

export interface FieldError {
  // Path sama dengan body JSON, misalnya "attendees[1].email".
  field: string
  message: string
}

export interface ApiError {
  code: ErrorCode
  message: string
  fields: FieldError[]
}

export type ApiResult<T> =
  { ok: true; status: number; data: T } | { ok: false; status: number; error: ApiError }
