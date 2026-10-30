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
    <h1 class="text-display">nvm-doomsday</h1>
    <p class="mt-4 text-fg-muted" aria-live="polite">
      <template v-if="state.kind === 'loading'">Mengecek backend...</template>
      <template v-else-if="state.kind === 'ok'">
        <span class="text-ok">Backend tersambung.</span>
      </template>
      <template v-else>
        <span class="text-error">Backend tidak bisa dihubungi ({{ state.message }}).</span>
      </template>
    </p>
    <p class="mt-2 font-mono text-sm text-fg-muted">GET /api/health</p>
    <button
      v-if="state.kind === 'error'"
      type="button"
      class="mt-4 min-h-11 rounded-paper border border-line px-4 active:translate-y-px"
      @click="check"
    >
      Coba lagi
    </button>
  </main>
</template>