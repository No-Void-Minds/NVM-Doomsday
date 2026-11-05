<script setup lang="ts">
import { computed, useId, useTemplateRef } from 'vue'

import type { RegistrationResult } from '@/api/types'
import BaseStamp from '@/components/BaseStamp.vue'
import { formatDateWIB, formatTimeWIB } from '@/lib/datetime'

const props = defineProps<{
  result: RegistrationResult
  filmTitle: string
  showAt: string | null
}>()

const titleId = useId()
const heading = useTemplateRef<HTMLHeadingElement>('heading')

// Dipanggil halaman setelah pendaftaran berhasil, supaya halaman bergulir ke kartu dan pembaca layar membacakannya.
defineExpose({ focus: () => heading.value?.focus() })

const confirmed = computed(() => props.result.status === 'confirmed')

const schedule = computed(() => {
  const date = formatDateWIB(props.showAt)
  const time = formatTimeWIB(props.showAt)
  return date && time ? `${date}, ${time}` : 'Menyusul'
})
</script>

<template>
  <section :aria-labelledby="titleId" class="flex flex-col gap-6">
    <div>
      <h2 :id="titleId" ref="heading" tabindex="-1" class="text-3xl">
        {{ confirmed ? 'Kartu akses kamu' : 'Kamu di daftar tunggu' }}
      </h2>
      <p class="mt-2 text-fg-muted">
        {{
          confirmed
            ? 'Simpan kartu ini. Tunjukkan kodenya saat hari H.'
            : 'Admin akan menghubungi lewat email kalau ada kursi kosong.'
        }}
      </p>
    </div>

    <!-- Hangtag: label karton yang tergantung di pakaian NVM. Takik dan lubang berwarna void supaya terlihat
         berlubang, jadi kartu harus selalu diletakkan di atas latar void, bukan di dalam panel. -->
    <div class="mx-auto w-full max-w-sm">
      <article class="relative rounded-paper bg-paper px-6 pb-6 pt-12 text-ink shadow-paper">
        <span
          class="absolute left-1/2 top-4 size-4 -translate-x-1/2 rounded-full bg-void ring-2 ring-ink/20"
          aria-hidden="true"
        />

        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <!-- Logo NVM putih dibalik jadi hitam di atas kertas, cara yang sama dengan tema Shopify mode terang. -->
            <img src="/brand/nvm-logo.png" alt="No Void Minds" class="h-6 w-auto invert" />
            <p class="mt-2 font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">
              Pas akses nonton bareng
            </p>
          </div>
          <BaseStamp>{{ confirmed ? 'Terkonfirmasi' : 'Daftar tunggu' }}</BaseStamp>
        </div>

        <!-- Celah Void sebagai garis sobek, dengan takik di kedua sisi seperti tiket. -->
        <div class="relative -mx-6 my-6 h-4" aria-hidden="true">
          <span class="absolute -left-2 top-0 size-4 rounded-full bg-void" />
          <span class="absolute -right-2 top-0 size-4 rounded-full bg-void" />
          <span class="absolute inset-x-4 top-1/2 h-[2px] -translate-y-1/2 bg-void" />
        </div>

        <!-- Kuning langsung di atas krem tidak terbaca (sekitar 1,3:1), jadi strip ini berlatar tinta. -->
        <p v-if="!confirmed" class="-mx-6 mb-6 bg-ink px-6 py-2 font-mono text-sm text-wait">
          Nomor urut {{ result.waitlist_position }} saat mendaftar
        </p>

        <p class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Kode registrasi</p>
        <p class="mt-1 break-all font-mono text-2xl font-bold tracking-wide sm:text-3xl">
          {{ result.code }}
        </p>

        <dl class="mt-6 border-t border-ink/15 text-sm">
          <div class="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 border-b border-ink/15 py-2.5">
            <dt class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Tipe</dt>
            <dd>{{ result.type === 'vip' ? 'VIP' : 'Solo' }}</dd>
          </div>
          <div class="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 border-b border-ink/15 py-2.5">
            <dt class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Peserta</dt>
            <dd>
              <ul>
                <li v-for="name in result.attendees" :key="name" class="break-words">{{ name }}</li>
              </ul>
            </dd>
          </div>
          <div class="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 border-b border-ink/15 py-2.5">
            <dt class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Acara</dt>
            <dd>Nonton bareng {{ filmTitle }}</dd>
          </div>
          <div class="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 py-2.5">
            <dt class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Jadwal</dt>
            <dd>{{ schedule }}</dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>
