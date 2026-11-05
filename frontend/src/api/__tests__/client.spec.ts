import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, exportUrl } from '@/api/client'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

function respond(status: number, body: string | null, contentType = 'application/json') {
  const fetch = vi.fn(
    async () => new Response(body, { status, headers: { 'Content-Type': contentType } }),
  )
  vi.stubGlobal('fetch', fetch)
  return fetch
}

describe('pemetaan error klien API', () => {
  it('error JSON dari backend diteruskan, field yang rusak dibuang', async () => {
    respond(
      409,
      JSON.stringify({
        error: {
          code: 'duplicate_email',
          message: 'Ada email yang sudah terdaftar.',
          fields: [
            { field: 'attendees[1].email', message: 'Email ini sudah terdaftar.' },
            { field: 3 },
            'rusak',
          ],
        },
      }),
    )
    const res = await api.register({
      attendees: [],
      consent: true,
      idempotency_key: 'x',
      website: '',
    })
    expect(res).toEqual({
      ok: false,
      status: 409,
      error: {
        code: 'duplicate_email',
        message: 'Ada email yang sudah terdaftar.',
        fields: [{ field: 'attendees[1].email', message: 'Email ini sudah terdaftar.' }],
      },
    })
  })

  it('jaringan putus jadi network_error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new TypeError('Failed to fetch')
      }),
    )
    const res = await api.getEvent()
    expect(res).toEqual({
      ok: false,
      status: 0,
      error: {
        code: 'network_error',
        message: 'Tidak bisa terhubung ke server. Periksa koneksi internetmu, lalu coba lagi.',
        fields: [],
      },
    })
  })

  it('tidak ada jawaban dalam 15 detik jadi timeout', async () => {
    vi.useFakeTimers()
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_: string, init: RequestInit) =>
          new Promise((_, reject) => {
            init.signal?.addEventListener('abort', () => reject(new DOMException('', 'AbortError')))
          }),
      ),
    )
    const pending = api.getEvent()
    await vi.advanceTimersByTimeAsync(15_000)
    const res = await pending
    expect(res.ok).toBe(false)
    expect(!res.ok && res.error.code).toBe('timeout')
  })

  it('halaman HTML dari proxy saat backend mati jadi internal_error', async () => {
    respond(502, '<html>Bad Gateway</html>', 'text/html')
    const res = await api.getEvent()
    expect(!res.ok && res.error).toEqual({
      code: 'internal_error',
      message: 'Terjadi kesalahan di server. Coba lagi sebentar.',
      fields: [],
    })
  })

  it('429 tanpa JSON tetap jadi rate_limited', async () => {
    respond(429, 'Too Many Requests', 'text/plain')
    const res = await api.checkAccount('budi')
    expect(!res.ok && res.error.code).toBe('rate_limited')
    expect(!res.ok && res.error.message).toBe(
      'Terlalu banyak percobaan. Tunggu sebentar, lalu coba lagi.',
    )
  })

  it('200 dengan isi bukan JSON dianggap internal_error', async () => {
    respond(200, 'ok', 'text/plain')
    const res = await api.getEvent()
    expect(!res.ok && res.error.code).toBe('internal_error')
  })

  it('204 berhasil tanpa isi', async () => {
    respond(204, null)
    expect(await api.adminSession()).toEqual({ ok: true, status: 204, data: undefined })
  })
})

describe('kiriman JSON', () => {
  it('selalu dengan Content-Type application/json, karena backend menolak yang lain dengan 415', async () => {
    const fetch = respond(200, JSON.stringify({ type: 'vip' }))
    const res = await api.checkAccount('budi')
    expect(res).toEqual({ ok: true, status: 200, data: { type: 'vip' } })
    expect(fetch).toHaveBeenCalledWith(
      '/api/account-check',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ account_name: 'budi' }),
        headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
      }),
    )
  })

  it('ID VIP di-escape di path PATCH', async () => {
    const fetch = respond(204, null)
    await api.adminVipSetActive('a/b?c', false)
    expect(fetch).toHaveBeenCalledWith(
      '/api/admin/vip/a%2Fb%3Fc',
      expect.objectContaining({ method: 'PATCH' }),
    )
  })

  it('link unduhan', () => {
    expect(exportUrl('csv')).toBe('/api/admin/export?format=csv')
    expect(exportUrl('xlsx')).toBe('/api/admin/export?format=xlsx')
  })
})
