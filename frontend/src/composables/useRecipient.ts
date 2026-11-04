import { computed } from 'vue'
import { useRoute } from 'vue-router'

const MAX_LENGTH = 60

// Karakter kontrol (Cc) dan karakter format (Cf), termasuk penanda arah teks, dibuang supaya nama
// di link tidak bisa membalik atau menyembunyikan teks di sekitarnya.
const UNSAFE = /[\p{Cc}\p{Cf}]/gu

export function sanitizeRecipient(raw: unknown): string | null {
  if (typeof raw !== 'string') {
    return null
  }
  const cleaned = raw.replace(UNSAFE, '').replace(/\s+/g, ' ').trim()
  if (cleaned === '') {
    return null
  }
  // Dihitung per karakter Unicode, bukan per unit UTF-16, supaya huruf beraksen tidak terpotong di tengah.
  const chars = Array.from(cleaned)
  return chars.length > MAX_LENGTH ? chars.slice(0, MAX_LENGTH).join('').trimEnd() : cleaned
}

export function useRecipient() {
  const route = useRoute()
  return computed(() => sanitizeRecipient(route.query.to))
}
