import type { EventInfo } from '@/api/types'
import type { AccordionItem } from '@/components/accordion'
import { formatTimeWIB } from '@/lib/datetime'

// Jam kumpul dibaca dari data acara, bukan ditulis mati, supaya FAQ ikut berubah saat admin mengubahnya lewat CLI.
// Selama data acara belum dimuat, jawabannya dibuat tanpa jam.
export function buildFaq(event: EventInfo | null): AccordionItem[] {
  const gatherAt = event ? formatTimeWIB(event.gather_at) : null

  return [
    {
      question: 'Siapa yang boleh daftar?',
      answer: 'Siapa saja yang membuka halaman ini boleh daftar.',
    },
    {
      question: 'Ada biayanya?',
      answer: 'Tidak. Formulir ini hanya untuk mendata siapa yang datang.',
    },
    {
      question: 'Jam berapa harus datang?',
      answer: gatherAt
        ? `Kumpul pukul ${gatherAt}, sebelum film mulai.`
        : 'Menyusul, mengikuti jadwal tayang resmi.',
    },
  ]
}
