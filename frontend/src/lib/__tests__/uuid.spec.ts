import { afterEach, describe, expect, it, vi } from 'vitest'

import { newIdempotencyKey } from '@/lib/uuid'

// Bentuk yang diterima uuid.Parse di backend: versi 4, varian RFC 4122.
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('newIdempotencyKey', () => {
  it('berbentuk UUID v4', () => {
    for (let i = 0; i < 1000; i++) {
      expect(newIdempotencyKey()).toMatch(UUID_V4)
    }
  })

  it('unik', () => {
    const keys = new Set(Array.from({ length: 10_000 }, () => newIdempotencyKey()))
    expect(keys.size).toBe(10_000)
  })

  // Di HTTP lewat IP LAN, crypto.randomUUID tidak ada. Yang tersedia hanya getRandomValues.
  it('jalan tanpa crypto.randomUUID', () => {
    const getRandomValues = vi.fn(<T extends ArrayBufferView>(b: T) => {
      new Uint8Array(b.buffer).fill(0xff)
      return b
    })
    vi.stubGlobal('crypto', { getRandomValues })
    expect(newIdempotencyKey()).toBe('ffffffff-ffff-4fff-bfff-ffffffffffff')
    expect(getRandomValues).toHaveBeenCalledOnce()
  })

  it('bit versi dan varian dipasang walau acakan bernilai nol', () => {
    vi.stubGlobal('crypto', { getRandomValues: <T extends ArrayBufferView>(b: T) => b })
    expect(newIdempotencyKey()).toBe('00000000-0000-4000-8000-000000000000')
  })
})
