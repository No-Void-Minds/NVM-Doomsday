// Aset Doomsday di halaman undangan. File ditaruh di frontend/public/doomsday/, lalu didaftarkan di sini.
// Mengganti gambar: taruh file baru, ubah src, alt, width, dan height, lalu commit dan build ulang.

export interface DoomsdayAsset {
  // Path dari folder public, diawali /doomsday/.
  src: string
  // Isi gambar untuk pembaca layar. Kosongkan ('') kalau gambarnya sama dengan sisi lain,
  // supaya tidak dibacakan dua kali.
  alt: string
  // Ukuran asli dalam piksel.
  width: number
  height: number
  // Keterangan atau kredit, misalnya sumber materi resmi.
  caption?: string
}

const placeholder: DoomsdayAsset = {
  src: '/doomsday/studio-side-placeholder.svg',
  alt: 'Ilustrasi studio bioskop gelap dengan layar menyala dan deretan kursi menghadap layar.',
  width: 900,
  height: 1600,
}

// Bidang kosong di kiri dan kanan halaman, hanya di layar mulai 1360px. Bidangnya tinggi dan sempit,
// jadi gambar potret paling pas, misalnya poster 2:3. Isi null untuk mengosongkan satu sisi.
export const sideAssets: { left: DoomsdayAsset | null; right: DoomsdayAsset | null } = {
  left: {
    ...placeholder,
    caption: 'Ilustrasi sementara. Diganti dengan materi resmi setelah disetujui NVM.',
  },
  right: { ...placeholder, alt: '' },
}
