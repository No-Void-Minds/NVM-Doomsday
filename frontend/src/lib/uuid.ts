// crypto.randomUUID hanya tersedia di secure context (HTTPS atau localhost). Saat halaman dibuka lewat IP LAN
// dengan HTTP biasa (Sprint 8), fungsi itu tidak ada, sedangkan getRandomValues tetap tersedia.
export function newIdempotencyKey(): string {
  const b = crypto.getRandomValues(new Uint8Array(16))
  // Penanda UUID versi 4 dan varian RFC 4122, supaya diterima uuid.Parse di backend.
  b[6] = ((b[6] ?? 0) & 0x0f) | 0x40
  b[8] = ((b[8] ?? 0) & 0x3f) | 0x80
  const h = Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('')
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`
}
