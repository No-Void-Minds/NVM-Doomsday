// Jam tayang selalu ditampilkan dalam WIB, termasuk untuk pengunjung di zona lain, karena itu jam yang berlaku di bioskop.
const TIME_ZONE = 'Asia/Jakarta'

const dateFormat = new Intl.DateTimeFormat('id-ID', {
  timeZone: TIME_ZONE,
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

const timeFormat = new Intl.DateTimeFormat('id-ID', {
  timeZone: TIME_ZONE,
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

function parse(iso: string | null): Date | null {
  if (iso === null) {
    return null
  }
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? null : d
}

// "Jumat, 18 Desember 2026". Null bila jadwal belum diisi admin, supaya pemanggil menampilkan "Menyusul".
export function formatDateWIB(iso: string | null): string | null {
  const d = parse(iso)
  return d ? dateFormat.format(d) : null
}

// "19.00 WIB".
export function formatTimeWIB(iso: string | null): string | null {
  const d = parse(iso)
  return d ? `${timeFormat.format(d)} WIB` : null
}
