<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue'

import BaseAccordion from '@/components/BaseAccordion.vue'
import { buildFaq } from '@/content/faq'
import EventSeats from '@/components/invite/EventSeats.vue'
import BaseButton from '@/components/BaseButton.vue'
import BasePanel from '@/components/BasePanel.vue'
import BaseSkeleton from '@/components/BaseSkeleton.vue'
import EventSpecSheet from '@/components/invite/EventSpecSheet.vue'
import InviteOpening from '@/components/invite/InviteOpening.vue'
import { useEvent } from '@/composables/useEvent'
import { useRecipient } from '@/composables/useRecipient'
import EventCountdown from '@/components/invite/EventCountdown.vue'
import RegistrationForm from '@/components/invite/RegistrationForm.vue'

const OPENED_KEY = 'doomsday_opened'

const recipient = useRecipient()
const { state: eventState, reload, refresh } = useEvent()
const faq = computed(() =>
  buildFaq(eventState.value.kind === 'ready' ? eventState.value.event : null),
)
// sessionStorage bisa diblokir di mode privat atau browser bawaan aplikasi. Kalau gagal, amplop tetap jalan
// dan hanya muncul lagi saat halaman dimuat ulang.
function readOpened(): boolean {
  try {
    return sessionStorage.getItem(OPENED_KEY) === '1'
  } catch {
    return false
  }
}

function rememberOpened() {
  try {
    sessionStorage.setItem(OPENED_KEY, '1')
  } catch {
    // Lihat readOpened.
  }
}

const opened = ref(readOpened())
const heading = useTemplateRef<HTMLHeadingElement>('heading')

async function onOpen() {
  opened.value = true
  rememberOpened()
  await nextTick()
  // Pengguna keyboard dan pembaca layar langsung berada di awal isi undangan, bukan di awal halaman.
  heading.value?.focus()
}
// Affiliator sering membuka link, pindah ke aplikasi lain, lalu kembali. Sisa kursi diambil ulang
// diam-diam saat tab kembali terlihat, supaya angkanya tidak basi.
function onVisibilityChange() {
  if (document.visibilityState === 'visible') {
    refresh()
  }
}

onMounted(() => document.addEventListener('visibilitychange', onVisibilityChange))
onBeforeUnmount(() => document.removeEventListener('visibilitychange', onVisibilityChange))
</script>

<template>
  <InviteOpening v-if="!opened" :recipient="recipient" @open="onOpen" />

  <main v-else class="mx-auto w-full max-w-xl px-4 py-10 lg:max-w-[70rem] lg:px-8 lg:py-16">
    <!-- Kolom kiri tidak menempel saat digulir: tingginya melebihi layar laptop kebanyakan,
         dan kolom menempel yang lebih tinggi dari layar menyembunyikan bagian bawahnya. -->
    <div class="grid gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
      <div class="flex min-w-0 flex-col gap-8">
        <div>
          <h1 ref="heading" tabindex="-1" class="text-display">Kamu diundang nonton bareng.</h1>
          <p class="mt-4 text-fg-muted">
            No Void Minds mengajak affiliator dan teman-teman nonton Avengers: Doomsday bareng di
            satu studio.
          </p>
        </div>

        <div v-if="eventState.kind === 'loading'" aria-busy="true">
          <span class="sr-only">Memuat detail acara...</span>
          <BasePanel>
            <div class="flex flex-col gap-3">
              <BaseSkeleton class="h-4 w-24" />
              <BaseSkeleton class="h-8 w-56 max-w-full" />
              <BaseSkeleton class="mt-4 h-4 w-full" />
              <BaseSkeleton class="h-4 w-full" />
              <BaseSkeleton class="h-4 w-3/4" />
            </div>
          </BasePanel>
        </div>

        <BasePanel v-else-if="eventState.kind === 'error'" role="alert">
          <h2 class="text-2xl">Detail acara gagal dimuat.</h2>
          <p class="mt-2 text-fg-muted">{{ eventState.error.message }}</p>
          <BaseButton variant="outline" class="mt-6" block @click="reload">Coba lagi</BaseButton>
        </BasePanel>

        <template v-else>
          <EventSpecSheet :event="eventState.event" />
          <EventCountdown :show-at="eventState.event.show_at" />
          <EventSeats :event="eventState.event" />
        </template>
      </div>

      <div class="flex min-w-0 flex-col gap-8">
        <div id="formulir" class="scroll-mt-8">
          <RegistrationForm
            v-if="eventState.kind === 'ready' && eventState.event.is_open"
            :event="eventState.event"
          />
        </div>
        <section aria-labelledby="faq-title" class="flex flex-col gap-4">
          <h2 id="faq-title" class="text-2xl">Pertanyaan yang sering muncul</h2>
          <BaseAccordion :items="faq" :initially-open="0" />
        </section>
      </div>
    </div>
  </main>
</template>
