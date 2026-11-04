<script setup lang="ts">
import { computed, useId } from 'vue'

import type { EventInfo } from '@/api/types'
import { formatDateWIB, formatTimeWIB } from '@/lib/datetime'

const props = defineProps<{ event: EventInfo }>()

const titleId = useId()

// Null berarti belum diisi admin dan ditampilkan sebagai "Menyusul".
const rows = computed(() => [
  { label: 'Bioskop', value: props.event.cinema_name },
  { label: 'Kota', value: props.event.city },
  { label: 'Studio', value: props.event.studio },
  { label: 'Tanggal', value: formatDateWIB(props.event.show_at) },
  { label: 'Jam kumpul', value: formatTimeWIB(props.event.gather_at) },
  { label: 'Jam tayang', value: formatTimeWIB(props.event.show_at) },
  { label: 'Kapasitas', value: `${props.event.capacity} kursi` },
])

const hasPending = computed(() => rows.value.some((row) => row.value === null))
</script>

<template>
  <article
    class="rounded-paper bg-paper p-5 text-ink shadow-paper sm:p-8"
    :aria-labelledby="titleId"
  >
    <!-- Label mono kapital dengan spasi renggang mengikuti gaya label di tema Shopify NVM. -->
    <p class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Nonton bareng</p>
    <h2 :id="titleId" class="mt-1 text-3xl text-ink">{{ event.film_title }}</h2>

    <dl class="mt-6 border-t border-ink/15">
      <!-- Di HP label di atas nilai, supaya nilai panjang tidak terjepit di layar 320px. -->
      <div
        v-for="row in rows"
        :key="row.label"
        class="grid gap-1 border-b border-ink/15 py-3 sm:grid-cols-[9rem_minmax(0,1fr)] sm:gap-4"
      >
        <dt class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded sm:pt-1">
          {{ row.label }}
        </dt>
        <dd class="min-w-0" :class="row.value === null ? 'text-ink-faded' : 'text-ink'">
          {{ row.value ?? 'Menyusul' }}
        </dd>
      </div>
    </dl>

    <!-- Cincin fokus global berwarna krem dan tidak terlihat di atas kertas, jadi di sini memakai warna tinta. -->
    <a
      v-if="event.map_url"
      :href="event.map_url"
      target="_blank"
      rel="noopener noreferrer"
      class="mt-4 inline-flex min-h-11 items-center gap-2 underline underline-offset-4 focus-visible:outline-ink active:text-ink-faded"
    >
      <svg class="size-4 shrink-0" viewBox="0 0 16 16" fill="none" aria-hidden="true">
        <path
          d="M8 14.5s4.5-4.2 4.5-8a4.5 4.5 0 1 0-9 0c0 3.8 4.5 8 4.5 8Z"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linejoin="round"
        />
        <circle cx="8" cy="6.5" r="1.6" stroke="currentColor" stroke-width="1.5" />
      </svg>
      Lihat lokasi di Google Maps
      <span class="sr-only">(membuka tab baru)</span>
    </a>

    <p v-if="hasPending" class="mt-4 text-sm text-ink-faded">
      Bagian yang masih "Menyusul" diisi setelah jadwal resmi bioskop diumumkan.
    </p>
  </article>
</template>
