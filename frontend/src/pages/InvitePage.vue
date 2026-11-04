<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue'

import EventSeats from '@/components/invite/EventSeats.vue'
import BaseButton from '@/components/BaseButton.vue'
import BasePanel from '@/components/BasePanel.vue'
import BaseSkeleton from '@/components/BaseSkeleton.vue'
import EventSpecSheet from '@/components/invite/EventSpecSheet.vue'
import InviteOpening from '@/components/invite/InviteOpening.vue'
import { useEvent } from '@/composables/useEvent'
import { useRecipient } from '@/composables/useRecipient'
import EventCountdown from '@/components/invite/EventCountdown.vue'

const OPENED_KEY = 'doomsday_opened'

const recipient = useRecipient()
const { state: eventState, reload, refresh } = useEvent()

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

  <main v-else class="mx-auto flex max-w-xl flex-col gap-8 px-4 py-10">
    <div>
      <h1 ref="heading" tabindex="-1" class="text-display">Kamu diundang nonton bareng.</h1>
      <p class="mt-4 text-fg-muted">
        No Void Minds mengajak affiliator dan teman-teman nonton Avengers: Doomsday bareng di satu
        studio.
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
  </main>
</template>
