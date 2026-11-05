<script setup lang="ts">
import { toBlob } from 'html-to-image'
import { computed, onBeforeUnmount, ref, useId, useTemplateRef } from 'vue'

import type { RegistrationResult } from '@/api/types'
import BaseButton from '@/components/BaseButton.vue'
import BaseStamp from '@/components/BaseStamp.vue'
import { copyText } from '@/lib/clipboard'
import { formatDateWIB, formatTimeWIB } from '@/lib/datetime'

const props = defineProps<{
  result: RegistrationResult
  filmTitle: string
  showAt: string | null
  // Terisi bila kartu dipulihkan dari perangkat, bukan hasil pendaftaran barusan.
  savedAt?: string | null
}>()

const titleId = useId()
const heading = useTemplateRef<HTMLHeadingElement>('heading')
const capture = useTemplateRef<HTMLDivElement>('capture')

// Dipanggil halaman setelah pendaftaran berhasil, supaya halaman bergulir ke kartu dan pembaca layar membacakannya.
defineExpose({ focus: () => heading.value?.focus() })

const confirmed = computed(() => props.result.status === 'confirmed')

const schedule = computed(() => {
  const date = formatDateWIB(props.showAt)
  const time = formatTimeWIB(props.showAt)
  return date && time ? `${date}, ${time}` : 'Menyusul'
})

const saving = ref(false)
const previewUrl = ref<string | null>(null)
const saveError = ref<string | null>(null)

// URL blob memegang gambar di memori sampai dilepas.
onBeforeUnmount(() => {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
  }
})

async function saveCard() {
  if (!capture.value || saving.value) {
    return
  }
  saving.value = true
  saveError.value = null
  const fileName = `nvm-doomsday-${props.result.code}.png`

  try {
    // toBlob, bukan toPng lalu fetch(dataUrl): CSP connect-src 'self' di Nginx menolak fetch ke URL data:,
    // sehingga di server cara lama selalu gagal. Di dev tidak terlihat karena Vite tidak memasang CSP.
    // pixelRatio 2 supaya kode tetap tajam saat gambar diperbesar di HP. Margin dinolkan karena mx-auto
    // yang dihitung browser jadi piksel akan ikut terbawa dan mendorong kartu keluar dari bingkai.
    const blob = await toBlob(capture.value, {
      pixelRatio: 2,
      backgroundColor: '#08090b',
      style: { margin: '0' },
    })
    if (!blob) {
      throw new Error('gambar kartu kosong')
    }
    const file = new File([blob], fileName, { type: 'image/png' })

    // Menu bagikan bawaan HP hanya tersedia di HTTPS. Di sana pendaftar bisa langsung memilih "Simpan gambar".
    if (navigator.canShare?.({ files: [file] })) {
      try {
        await navigator.share({ files: [file], title: 'Kartu akses No Void Minds Doomsday' })
        return
      } catch (err) {
        // Pendaftar menutup menu bagikan tanpa memilih apa pun. Itu bukan kegagalan.
        if (err instanceof DOMException && err.name === 'AbortError') {
          return
        }
        // Di HP, menggambar kartu bisa lebih lama dari batas aktivasi tombol, sehingga browser menolak
        // membuka menu bagikan (NotAllowedError). Gambarnya sudah jadi, jadi lanjut ke unduhan biasa.
      }
    }

    if (previewUrl.value) {
      URL.revokeObjectURL(previewUrl.value)
    }
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    link.click()
    // Browser bawaan WhatsApp dan Instagram sering mengabaikan unduhan, jadi gambarnya juga ditampilkan
    // supaya bisa disimpan dengan tekan lama.
    previewUrl.value = url
  } catch {
    saveError.value = 'Gambar kartu gagal dibuat. Screenshot layar ini sebagai gantinya.'
  } finally {
    saving.value = false
  }
}

const copyStatus = ref<'idle' | 'copied' | 'failed'>('idle')
let copyTimer: number | undefined

async function copyCode() {
  copyStatus.value = (await copyText(props.result.code)) ? 'copied' : 'failed'
  window.clearTimeout(copyTimer)
  if (copyStatus.value === 'copied') {
    copyTimer = window.setTimeout(() => (copyStatus.value = 'idle'), 2500)
  }
}
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
      <p v-if="savedAt" class="mt-2 text-sm text-fg-muted">
        Kartu ini tersimpan di perangkat ini sejak {{ formatDateWIB(savedAt) }}. Kalau statusnya
        berubah, admin mengabari lewat email.
      </p>
    </div>

    <!-- Wadah ini yang digambar jadi PNG. Latar void dan padding ikut terekam, supaya takik dan bayangan
         kartu terlihat sama seperti di layar. -->
    <div ref="capture" class="mx-auto w-full max-w-sm bg-void p-4">
      <!-- Hangtag: label karton yang tergantung di pakaian NVM. Takik dan lubang berwarna void supaya terlihat
           berlubang, jadi kartu harus selalu diletakkan di atas latar void, bukan di dalam panel. -->
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
                <li v-for="(name, i) in result.attendees" :key="i">
                  {{ name }}
                </li>
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

    <div class="mx-auto flex w-full max-w-sm flex-col gap-3">
      <BaseButton block :loading="saving" @click="saveCard">
        {{ saving ? 'Membuat gambar...' : 'Simpan kartu akses' }}
      </BaseButton>
      <BaseButton variant="outline" block @click="copyCode">
        {{ copyStatus === 'copied' ? 'Kode tersalin' : 'Salin kode' }}
      </BaseButton>

      <!-- Wadah status selalu ada supaya pembaca layar mengumumkan perubahannya. -->
      <div role="status" class="text-sm">
        <p v-if="copyStatus === 'failed'" class="text-error">
          Kode tidak bisa disalin otomatis. Tulis atau screenshot kode di atas.
        </p>
        <p v-if="saveError" class="text-error">{{ saveError }}</p>
      </div>

      <figure v-if="previewUrl" class="mt-2 flex flex-col gap-2">
        <img
          :src="previewUrl"
          alt="Gambar kartu akses yang siap disimpan"
          class="w-full rounded-paper"
        />
        <figcaption class="text-sm text-fg-muted">
          Kalau unduhan tidak berjalan, tekan lama gambar ini lalu pilih simpan gambar.
        </figcaption>
      </figure>
    </div>
  </section>
</template>
