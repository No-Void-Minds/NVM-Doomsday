<script setup lang="ts">
import { ref } from 'vue'

import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'
import BaseAccordion from '@/components/BaseAccordion.vue'
import BaseSkeleton from '@/components/BaseSkeleton.vue'
import BaseStamp from '@/components/BaseStamp.vue'
import VoidSlit from '@/components/VoidSlit.vue'
import type { AccordionItem } from '@/components/accordion'
import { useEvent } from '@/composables/useEvent'

const { state: eventState, reload, refresh } = useEvent()
const name = ref('')
const email = ref('nama@contoh')
const password = ref('')
const loading = ref(false)
const faq: AccordionItem[] = [
  {
    question: 'Siapa yang boleh daftar?',
    answer: 'Siapa saja yang membuka halaman ini boleh daftar.',
  },
  {
    question: 'Ada biayanya?',
    answer: 'Tidak. Formulir ini hanya untuk mendata siapa yang datang.',
  },
  {
    question: 'Kalau 200 kursi sudah penuh?',
    answer: 'Kamu tetap bisa daftar dan masuk daftar tunggu.',
  },
]

function simulateLoading() {
  loading.value = true
  setTimeout(() => (loading.value = false), 2000)
}
</script>

<template>
  <main class="mx-auto flex max-w-3xl flex-col gap-8 px-4 py-10">
    <div>
      <h1 class="text-display">Contoh komponen</h1>
      <p class="mt-2 text-fg-muted">Halaman ini hanya ada saat development.</p>
    </div>

    <BasePanel>
      <h2 class="text-2xl">Tombol</h2>
      <div class="mt-4 flex flex-wrap gap-3">
        <BaseButton>Primary</BaseButton>
        <BaseButton variant="outline">Outline</BaseButton>
        <BaseButton variant="text">Text</BaseButton>
        <BaseButton disabled>Nonaktif</BaseButton>
        <BaseButton :loading="loading" @click="simulateLoading">
          {{ loading ? 'Mengirim...' : 'Coba status memuat' }}
        </BaseButton>
      </div>
      <BaseButton class="mt-4" block>Lebar penuh</BaseButton>
    </BasePanel>

    <BasePanel>
      <h2 class="text-2xl">Kolom isian</h2>
      <div class="mt-4 grid gap-5">
        <BaseField
          v-model="name"
          label="Nama lengkap"
          required
          autocomplete="name"
          :maxlength="100"
          placeholder="Nama lengkap"
        />
        <BaseField
          v-model="email"
          label="Email"
          type="email"
          inputmode="email"
          autocomplete="email"
          required
          error="Format email belum benar, contoh: nama@email.com."
        />
        <BaseField
          v-model="password"
          label="Sandi admin"
          type="password"
          autocomplete="current-password"
          hint="Contoh petunjuk di bawah kolom."
        />
        <BaseField v-model="name" label="Nonaktif" disabled />
      </div>
    </BasePanel>
    <BasePanel>
      <h2 class="text-2xl">Kertas, stempel, celah Void</h2>
      <div class="mt-4 rounded-paper bg-paper p-5 text-ink shadow-paper">
        <div class="flex items-start justify-between gap-4">
          <p class="font-mono text-sm text-ink-faded">Dari: No Void Minds</p>
          <BaseStamp>Panggilan</BaseStamp>
        </div>
        <div class="my-5"><VoidSlit /></div>
        <p>Contoh isi di atas kertas.</p>
      </div>
      <div class="mt-6"><VoidSlit /></div>
    </BasePanel>

    <section>
      <h2 class="text-2xl">Accordion</h2>
      <div class="mt-4">
        <BaseAccordion :items="faq" :initially-open="0" />
      </div>
    </section>

    <BasePanel>
      <h2 class="text-2xl">Skeleton</h2>
      <div class="mt-4 flex flex-col gap-3" aria-busy="true">
        <span class="sr-only">Memuat...</span>
        <BaseSkeleton class="h-6 w-48 max-w-full" />
        <BaseSkeleton class="h-4 w-full" />
        <BaseSkeleton class="h-4 w-3/4" />
      </div>
    </BasePanel>
    <BasePanel>
      <h2 class="text-2xl">useEvent</h2>
      <div class="mt-4" :aria-busy="eventState.kind === 'loading'">
        <div v-if="eventState.kind === 'loading'" class="flex flex-col gap-3">
          <span class="sr-only">Memuat detail acara...</span>
          <BaseSkeleton class="h-6 w-56 max-w-full" />
          <BaseSkeleton class="h-4 w-32" />
        </div>
        <div v-else-if="eventState.kind === 'ready'">
          <p class="font-heading text-xl">{{ eventState.event.title }}</p>
          <p class="mt-1 text-fg-muted">
            {{ eventState.event.seats_left }} dari {{ eventState.event.capacity }} kursi tersisa
          </p>
        </div>
        <div v-else role="alert">
          <p class="font-heading text-xl">Detail acara gagal dimuat.</p>
          <p class="mt-1 text-fg-muted">{{ eventState.error.message }}</p>
        </div>
      </div>
      <div class="mt-4 flex flex-wrap gap-3">
        <BaseButton variant="outline" @click="reload">Coba lagi</BaseButton>
        <BaseButton variant="text" @click="refresh">Perbarui diam-diam</BaseButton>
      </div>
    </BasePanel>
  </main>
</template>
