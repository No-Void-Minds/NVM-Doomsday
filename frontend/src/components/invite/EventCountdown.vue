<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId } from 'vue'

import VoidSlit from '@/components/VoidSlit.vue'
import { formatDateWIB, formatTimeWIB } from '@/lib/datetime'

const props = defineProps<{ showAt: string | null }>()

const titleId = useId()

// Memakai jam perangkat pengunjung. Kalau jam perangkat meleset, hitung mundur ikut meleset,
// tapi jam tayang resmi tetap tertulis di lembar spesifikasi.
const now = ref(Date.now())
let timer: number | undefined

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onBeforeUnmount(() => {
  window.clearInterval(timer)
})

const target = computed(() => {
  if (props.showAt === null) {
    return null
  }
  const t = new Date(props.showAt).getTime()
  return Number.isNaN(t) ? null : t
})

const secondsLeft = computed(() =>
  target.value === null ? null : Math.max(0, Math.floor((target.value - now.value) / 1000)),
)

const parts = computed(() => {
  const s = secondsLeft.value ?? 0
  return [
    { label: 'hari', value: Math.floor(s / 86400) },
    { label: 'jam', value: Math.floor((s % 86400) / 3600) },
    { label: 'menit', value: Math.floor((s % 3600) / 60) },
    { label: 'detik', value: s % 60 },
  ]
})

// Kalimat tetap untuk pembaca layar, karena angka yang berubah setiap detik tidak boleh dibacakan terus-menerus.
const spokenSchedule = computed(() =>
  props.showAt
    ? `Jam tayang ${formatDateWIB(props.showAt)} pukul ${formatTimeWIB(props.showAt)}.`
    : '',
)

function pad(n: number): string {
  return String(n).padStart(2, '0')
}
</script>

<template>
  <section :aria-labelledby="titleId" class="flex flex-col gap-4">
    <VoidSlit />
    <div class="py-2 text-center">
      <h2 :id="titleId" class="font-mono text-xs uppercase tracking-[0.08em] text-fg-muted">
        Menuju jam tayang
      </h2>

      <p v-if="target === null" class="mt-3 text-lg">Jadwal tayang diumumkan menyusul.</p>
      <p v-else-if="secondsLeft === 0" class="mt-3 text-lg">Pemutaran sudah dimulai.</p>
      <template v-else>
        <p class="sr-only">{{ spokenSchedule }}</p>
        <div class="mt-4 grid grid-cols-4 gap-2" aria-hidden="true">
          <div v-for="part in parts" :key="part.label">
            <p
              class="font-heading text-[clamp(2.25rem,1.5rem+4vw,4rem)] leading-none text-paper tabular-nums"
            >
              {{ pad(part.value) }}
            </p>
            <p class="mt-2 font-mono text-xs text-fg-muted">{{ part.label }}</p>
          </div>
        </div>
      </template>
    </div>
    <VoidSlit />
  </section>
</template>
