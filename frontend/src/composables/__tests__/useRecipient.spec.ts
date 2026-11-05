import { reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import { sanitizeRecipient, useRecipient } from '@/composables/useRecipient'

// useRoute diganti route palsu yang reaktif, supaya composable bisa diuji tanpa DOM dan router sungguhan.
const route = reactive<{ query: Record<string, unknown> }>({ query: {} })
vi.mock('vue-router', () => ({ useRoute: () => route }))

describe('sanitizeRecipient', () => {
  it.each([
    ['Budi', 'Budi'],
    ['  Budi   Santoso  ', 'Budi Santoso'],
    // Tag tidak dibuang di sini. Vue menampilkannya sebagai teks, bukan HTML.
    ['<b>Budi</b>', '<b>Budi</b>'],
    ['\u202eBudi\u202c', 'Budi'],
    ['\u200fBu\u200bdi\u2066', 'Budi'],
    ['\ufeffBudi', 'Budi'],
    ['Budi\u0007', 'Budi'],
    ['Budi\u00a0Santoso', 'Budi Santoso'],
  ])('%j menjadi %j', (input, want) => {
    expect(sanitizeRecipient(input)).toBe(want)
  })

  it.each([undefined, null, '', '   ', '\u202e\u202c', ['Budi', 'Sari'], 42])(
    '%j dianggap kosong',
    (input) => {
      expect(sanitizeRecipient(input)).toBeNull()
    },
  )

  it('dipotong 60 karakter tanpa spasi di ujung', () => {
    const got = sanitizeRecipient(`${'a'.repeat(59)} ${'b'.repeat(20)}`)
    expect(got).toBe('a'.repeat(59))
    expect(sanitizeRecipient('x'.repeat(80))).toBe('x'.repeat(60))
  })

  it('dihitung per karakter Unicode, emoji tidak terbelah', () => {
    const got = sanitizeRecipient('\u{1F600}'.repeat(61))
    expect(Array.from(got ?? '')).toHaveLength(60)
    expect(got).toBe('\u{1F600}'.repeat(60))
  })
})

describe('useRecipient', () => {
  it('membaca ?to= dan ikut berubah saat query berubah', () => {
    route.query = { to: '  Budi  ' }
    const recipient = useRecipient()
    expect(recipient.value).toBe('Budi')

    route.query = { to: ['Budi', 'Sari'] }
    expect(recipient.value).toBeNull()

    route.query = {}
    expect(recipient.value).toBeNull()
  })
})
