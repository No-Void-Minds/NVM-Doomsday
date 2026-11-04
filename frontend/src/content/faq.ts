import type { EventInfo } from '@/api/types'
import type { AccordionItem } from '@/components/accordion'
import { formatTimeWIB } from '@/lib/datetime'

// Angka dibaca dari data acara, bukan ditulis mati, supaya FAQ ikut berubah saat admin mengubah kapasitas lewat CLI.
// Selama data acara belum dimuat, kalimatnya dibuat tanpa angka.
export function buildFaq(event: EventInfo | null): AccordionItem[] {
  const maxGuests = event?.max_guests_vip
  const gatherAt = event ? formatTimeWIB(event.gather_at) : null

  return [
    {
      question: 'Siapa yang boleh daftar?',
      answer: 'Siapa saja yang membuka halaman ini boleh daftar.',
    },
    {
      question: 'Apa beda Solo dan VIP?',
      answer: maxGuests
        ? `Solo untuk kamu sendiri. VIP untuk kamu ditambah maksimal ${maxGuests} orang. Data setiap orang wajib diisi, termasuk tamu.`
        : 'Solo untuk kamu sendiri. VIP untuk kamu ditambah beberapa tamu. Data setiap orang wajib diisi, termasuk tamu.',
    },
    {
      question: 'Ada biayanya?',
      answer: 'Tidak. Formulir ini hanya untuk mendata siapa yang datang.',
    },
    {
      question: event ? `Kalau ${event.capacity} kursi sudah penuh?` : 'Kalau kursi sudah penuh?',
      answer:
        'Kamu tetap bisa daftar dan masuk daftar tunggu. Admin akan menghubungi kalau ada kursi kosong.',
    },
    {
      question: 'Salah ketik nama atau mau ganti tamu?',
      // Kontak admin belum diputuskan NVM (SRS bagian 10). Ganti kalimat ini setelah kontaknya ada.
      answer: 'Hubungi admin NVM. Kontak admin diumumkan menyusul.',
    },
    {
      question: 'Jam berapa harus datang?',
      answer: gatherAt
        ? `Kumpul pukul ${gatherAt}, sebelum film mulai.`
        : 'Menyusul, mengikuti jadwal tayang resmi.',
    },
  ]
}
