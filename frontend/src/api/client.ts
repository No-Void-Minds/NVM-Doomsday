import type {
  AdminSummary,
  ApiError,
  ApiResult,
  ErrorCode,
  EventInfo,
  FieldError,
  RegistrationRequest,
  RegistrationResult,
  AccountCheckResult,
  VipAddResult,
  VipAccount,
} from './types'

// Lebih lama dari waktu wajar pendaftaran di jaringan seluler yang lemah, tapi cukup pendek supaya
// pendaftar tidak menunggu tanpa kepastian.
const TIMEOUT_MS = 15_000

const MESSAGES: Record<'network_error' | 'timeout' | 'internal_error' | 'rate_limited', string> = {
  network_error: 'Tidak bisa terhubung ke server. Periksa koneksi internetmu, lalu coba lagi.',
  timeout: 'Server terlalu lama menjawab. Coba lagi sebentar.',
  internal_error: 'Terjadi kesalahan di server. Coba lagi sebentar.',
  rate_limited: 'Terlalu banyak percobaan. Tunggu sebentar, lalu coba lagi.',
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH'
  body?: string
  headers?: Record<string, string>
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<ApiResult<T>> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS)

  let res: Response
  try {
    res = await fetch(path, {
      method: options.method ?? 'GET',
      body: options.body,
      headers: { Accept: 'application/json', ...options.headers },
      signal: controller.signal,
    })
  } catch {
    const code = controller.signal.aborted ? 'timeout' : 'network_error'
    return failure(0, code, MESSAGES[code])
  } finally {
    clearTimeout(timer)
  }

  if (res.status === 204) {
    return { ok: true, status: 204, data: undefined as unknown as T }
  }

  const body: unknown = await res.json().catch(() => null)
  if (res.ok) {
    return body === null
      ? failure(res.status, 'internal_error', MESSAGES.internal_error)
      : { ok: true, status: res.status, data: body as T }
  }
  return { ok: false, status: res.status, error: toApiError(body, res.status) }
}

function sendJSON<T>(
  path: string,
  payload: unknown,
  method: 'POST' | 'PATCH' = 'POST',
): Promise<ApiResult<T>> {
  return request<T>(path, {
    method,
    // Backend menolak kiriman yang bukan JSON (415), sebagai pengganti pemeriksaan CORS.
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

// Respons error yang bukan dari backend kita, misalnya halaman HTML dari proxy saat backend mati,
// tetap diubah ke bentuk ApiError supaya halaman cukup menangani satu bentuk.
function toApiError(body: unknown, status: number): ApiError {
  if (isRecord(body) && isRecord(body.error)) {
    const { code, message, fields } = body.error
    if (typeof code === 'string' && typeof message === 'string') {
      return {
        code: code as ErrorCode,
        message,
        fields: Array.isArray(fields) ? fields.filter(isFieldError) : [],
      }
    }
  }
  if (status === 429) {
    return { code: 'rate_limited', message: MESSAGES.rate_limited, fields: [] }
  }
  return { code: 'internal_error', message: MESSAGES.internal_error, fields: [] }
}

function failure<T>(status: number, code: ErrorCode, message: string): ApiResult<T> {
  return { ok: false, status, error: { code, message, fields: [] } }
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null
}

function isFieldError(v: unknown): v is FieldError {
  return isRecord(v) && typeof v.field === 'string' && typeof v.message === 'string'
}

export const api = {
  getEvent: () => request<EventInfo>('/api/event'),
  checkAccount: (accountName: string) =>
    sendJSON<AccountCheckResult>('/api/account-check', { account_name: accountName }),
  register: (req: RegistrationRequest) => sendJSON<RegistrationResult>('/api/registrations', req),
  adminLogin: (password: string) => sendJSON<void>('/api/admin/login', { password }),
  adminLogout: () => request<void>('/api/admin/logout', { method: 'POST' }),
  adminSession: () => request<void>('/api/admin/session'),
  adminSummary: () => request<AdminSummary>('/api/admin/summary'),
  adminVipList: () => request<{ accounts: VipAccount[] }>('/api/admin/vip'),
  adminVipAdd: (names: string[]) => sendJSON<VipAddResult>('/api/admin/vip', { names }),
  adminVipSetActive: (id: string, active: boolean) =>
    sendJSON<void>(`/api/admin/vip/${encodeURIComponent(id)}`, { is_active: active }, 'PATCH'),
}

// Unduhan memakai link biasa, bukan fetch, supaya browser menangani file dan nama dari Content-Disposition.
// Cookie admin ikut terkirim karena path-nya /api/admin.
export function exportUrl(format: 'csv' | 'xlsx'): string {
  return `/api/admin/export?format=${format}`
}
