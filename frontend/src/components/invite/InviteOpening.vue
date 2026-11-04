<script setup lang="ts">
import { ref } from 'vue'

import BaseButton from '@/components/BaseButton.vue'
import BaseStamp from '@/components/BaseStamp.vue'

defineProps<{ recipient: string | null }>()
const emit = defineEmits<{ open: [] }>()

// Tutup terangkat (300 ms), lalu amplop memudar (300 ms).
const OPEN_DURATION_MS = 600

const opening = ref(false)

function open() {
  if (opening.value) {
    return
  }
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    emit('open')
    return
  }
  opening.value = true
  window.setTimeout(() => emit('open'), OPEN_DURATION_MS)
}
</script>

<template>
  <main class="flex min-h-dvh flex-col justify-center px-4 py-10">
    <div class="mx-auto flex w-full max-w-md flex-col gap-8">
      <div>
        <h1 class="text-display">
          {{
            recipient ? `Ada surat panggilan untuk ${recipient}.` : 'Ada surat panggilan untukmu.'
          }}
        </h1>
        <p class="mt-4 text-fg-muted">
          Undangan nonton bareng Avengers: Doomsday. Terbuka untuk affiliator dan siapa saja yang
          menerima link ini.
        </p>
      </div>

      <div
        class="transition-[opacity,transform] delay-300 duration-300 ease-in"
        :class="opening ? 'translate-y-4 opacity-0' : ''"
      >
        <div class="relative aspect-[3/2] w-full -rotate-2" aria-hidden="true">
          <div class="absolute inset-0 rounded-paper bg-paper shadow-paper" />

          <!-- Celah Void: kegelapan dari dalam surat yang terlihat di bibir amplop saat tutupnya terangkat. -->
          <div class="absolute inset-x-4 top-0 h-[2px] bg-void/80" />

          <div class="absolute inset-x-0 top-0 h-[55%] [perspective:800px]">
            <div
              class="h-full origin-top transition-transform duration-300 ease-out [filter:drop-shadow(0_1px_0_rgb(26_28_32/0.25))]"
              :class="opening ? '[transform:rotateX(180deg)]' : ''"
            >
              <div class="h-full bg-paper-shade [clip-path:polygon(0_0,100%_0,50%_100%)]" />
            </div>
          </div>

          <div class="absolute inset-x-5 bottom-5 flex items-end justify-between gap-4">
            <div>
              <p class="font-mono text-xs uppercase tracking-[0.08em] text-ink-faded">Dari</p>
              <!-- Logo NVM putih dibalik jadi hitam di atas kertas, cara yang sama dengan tema Shopify mode terang. -->
              <img src="/brand/nvm-logo.png" alt="" class="mt-1 h-6 w-auto invert" />
            </div>
            <BaseStamp>Panggilan</BaseStamp>
          </div>
        </div>
      </div>

      <BaseButton block :disabled="opening" @click="open">Buka surat panggilan</BaseButton>
    </div>
  </main>
</template>
