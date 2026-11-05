// Aset Doomsday di halaman undangan. File ditaruh di frontend/public/doomsday/, lalu didaftarkan di sini.
// Mengganti foto: taruh file baru, ubah src, alt, width, dan height, lalu commit dan build ulang.
// Daftar kosong menyembunyikan bagiannya.

export interface DoomsdayAsset {
  // Path dari folder public, diawali /doomsday/.
  src: string
  // Isi gambar untuk pembaca layar. Jangan diisi "gambar" atau nama file.
  alt: string
  // Ukuran asli dalam piksel, supaya browser menyiapkan ruang sebelum gambar termuat.
  width: number
  height: number
  // Keterangan atau kredit, misalnya sumber materi resmi.
  caption?: string
}

// Gambar pertama jadi key visual di atas isi undangan. Gambar berikutnya disimpan untuk galeri nanti.
export const doomsdayAssets: DoomsdayAsset[] = [
  {
    src: '/doomsday/studio-placeholder.svg',
    alt: 'Ilustrasi studio bioskop gelap dengan layar menyala dan deretan kursi menghadap layar.',
    width: 1600,
    height: 900,
    caption: 'Ilustrasi sementara. Diganti dengan materi resmi setelah disetujui NVM.',
  },
]
