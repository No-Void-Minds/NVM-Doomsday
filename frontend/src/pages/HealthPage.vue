<script setup lang="ts">
import { onMounted, ref } from 'vue'

type HealthState = { kind: 'loading' } | { kind: 'ok' } | { kind: 'error'; message: string }

const state = ref<HealthState>({ kind: 'loading' })

async function check() {
  state.value = { kind: 'loading' }
  try {
    const res = await fetch('/api/health')
    if (!res.ok) {
      throw new Error(`status ${res.status}`)
    }
    state.value = { kind: 'ok' }
  } catch (err) {
    state.value = {
      kind: 'error',
      message: err instanceof Error ? err.message : 'penyebab tidak diketahui',
    }
  }
}

onMounted(check)
</script>

<template>
  <main class="mx-auto max-w-md p-6">
    <h1 class="text-2xl font-bold">nvm-doomsday</h1>
    <p class="mt-4" aria-live="polite">
      <template v-if="state.kind === 'loading'">Mengecek backend...</template>
      <template v-else-if="state.kind === 'ok'">Backend tersambung.</template>
      <template v-else>Backend tidak bisa dihubungi ({{ state.message }}).</template>
    </p>
    <button
      v-if="state.kind === 'error'"
      type="button"
      class="mt-4 min-h-11 border px-4"
      @click="check"
    >
      Coba lagi
    </button>
  </main>
</template>