// Clipboard API hanya tersedia di HTTPS atau localhost. Saat halaman dibuka lewat IP LAN dengan HTTP (Sprint 8),
// cara lama execCommand masih bisa menyalin, jadi dipakai sebagai cadangan.
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Ditolak browser, lanjut ke cara lama di bawah.
    }
  }

  const previousFocus = document.activeElement as HTMLElement | null
  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.opacity = '0'
  document.body.appendChild(area)
  area.select()

  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  }
  area.remove()
  // select() memindahkan fokus ke textarea sementara, jadi fokus dikembalikan ke tombol yang ditekan.
  previousFocus?.focus()
  return copied
}
