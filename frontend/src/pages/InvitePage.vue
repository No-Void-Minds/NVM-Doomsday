<script setup lang="ts">
import { useRecipient } from '@/composables/useRecipient'

const recipient = useRecipient()
</script>

<template>
  <main class="mx-auto max-w-xl px-4 py-10">
    <h1 class="text-display">
      {{ recipient ? `Ada surat panggilan untuk ${recipient}.` : 'Ada surat panggilan untukmu.' }}
    </h1>
    <p class="mt-4 text-fg-muted">Kerangka sementara. Isi undangan dibuat di Sprint 5.</p>
  </main>
</template>
