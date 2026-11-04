<script setup lang="ts">
import { computed } from 'vue'

import type { EventInfo } from '@/api/types'
import BasePanel from '@/components/BasePanel.vue'

const props = defineProps<{ event: EventInfo }>()

// Bar menunjukkan sisa kursi, jadi makin pendek saat kursi makin sedikit.
const remainingPercent = computed(() => {
  const { capacity, seats_left } = props.event
  return capacity > 0 ? Math.min(100, (seats_left / capacity) * 100) : 0
})
</script>

<template>
  <BasePanel v-if="!event.is_open" as="div">
    <h2 class="text-2xl">Pendaftaran sudah ditutup.</h2>
    <p class="mt-2 text-fg-muted">
      Terima kasih sudah tertarik. Pantau info selanjutnya dari No Void Minds.
    </p>
  </BasePanel>

  <div
    v-else-if="event.seats_left === 0"
    class="rounded-paper border-l-4 border-wait bg-panel p-5 sm:p-8"
  >
    <h2 class="text-2xl">Kursi utama sudah penuh.</h2>
    <p class="mt-2 text-fg-muted">
      Kamu masih bisa daftar dan masuk daftar tunggu. Admin akan menghubungi kalau ada kursi kosong.
    </p>
    <p v-if="event.waitlisted_attendees > 0" class="mt-3 font-mono text-sm text-wait">
      {{ event.waitlisted_attendees }} orang di daftar tunggu
    </p>
  </div>

  <BasePanel v-else as="div">
    <p>
      <span class="font-heading text-3xl text-fg tabular-nums">{{ event.seats_left }}</span>
      <span class="text-fg-muted"> dari {{ event.capacity }} kursi tersisa</span>
    </p>
    <!-- Angka di atas sudah menyampaikan isinya, jadi bar hanya penegas visual. -->
    <div class="mt-3 h-1 w-full rounded-paper bg-line/40" aria-hidden="true">
      <div class="h-full rounded-paper bg-fg" :style="{ width: `${remainingPercent}%` }" />
    </div>
  </BasePanel>
</template>
