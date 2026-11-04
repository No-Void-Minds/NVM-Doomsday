import { onMounted, shallowRef } from 'vue'

import { api } from '@/api/client'
import type { ApiError, EventInfo } from '@/api/types'

export type EventState =
  { kind: 'loading' } | { kind: 'ready'; event: EventInfo } | { kind: 'error'; error: ApiError }

export function useEvent() {
  const state = shallowRef<EventState>({ kind: 'loading' })
  let latest = 0

  // silent: data lama tetap tampil selama memuat, dan dipertahankan bila gagal.
  // Dipakai untuk memperbarui sisa kursi setelah pendaftaran tanpa mengosongkan halaman.
  async function load(options: { silent?: boolean } = {}) {
    const silent = options.silent === true && state.value.kind === 'ready'
    const id = ++latest
    if (!silent) {
      state.value = { kind: 'loading' }
    }

    const res = await api.getEvent()
    // Jawaban dari permintaan yang sudah digantikan permintaan lebih baru diabaikan.
    if (id !== latest) {
      return
    }
    if (res.ok) {
      state.value = { kind: 'ready', event: res.data }
    } else if (!silent) {
      state.value = { kind: 'error', error: res.error }
    }
  }

  onMounted(() => load())

  return { state, reload: () => load(), refresh: () => load({ silent: true }) }
}
