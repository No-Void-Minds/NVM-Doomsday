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
import { api } from '@/api/client'
import type { RegistrationResult } from '@/api/types'
import type { RegistrationDraft } from '@/components/invite/RegistrationForm.vue'
import type { FieldErrors } from '@/lib/validation'
import { newIdempotencyKey } from '@/lib/uuid'
import AccessCard from '@/components/invite/AccessCard.vue'

const OPENED_KEY = 'doomsday_opened'

const recipient = useRecipient()
const { state: eventState, reload, refresh } = useEvent()
const faq = computed(() =>
  buildFaq(eventState.value.kind === 'ready' ? eventState.value.event : null),
)
// Hanya path yang punya kolom di formulir yang ditempel ke kolom. Sisanya ditampilkan di atas tombol kirim.
const FORM_FIELD = /^(attendees\[\d+\]\.(full_name|email|account_name)|consent|attendees)$/

const submitting = ref(false)
const serverErrors = ref<FieldErrors>({})
const formError = ref<string | null>(null)
const CARD_KEY = 'doomsday_card'

interface StoredCard {
  result: RegistrationResult
  savedAt: string
}

// Isi localStorage bisa rusak atau diubah orang, jadi bentuknya diperiksa sebelum dipakai.
function isStoredCard(v: unknown): v is StoredCard {
  if (typeof v !== 'object' || v === null) {
    return false
  }
  const { result, savedAt } = v as Record<string, unknown>
  if (typeof savedAt !== 'string' || typeof result !== 'object' || result === null) {
    return false
  }
  const r = result as Record<string, unknown>
  return (
    typeof r.code === 'string' &&
    (r.type === 'solo' || r.type === 'vip') &&
    (r.status === 'confirmed' || r.status === 'waitlisted') &&
    (r.waitlist_position === null || typeof r.waitlist_position === 'number') &&
    Array.isArray(r.attendees) &&
    r.attendees.every((name) => typeof name === 'string')
  )
}

// localStorage bisa diblokir di mode privat atau browser bawaan aplikasi. Kalau gagal, kartu tetap tampil
// di sesi ini, hanya tidak muncul lagi saat link dibuka ulang.
function readCard(): StoredCard | null {
  try {
    const raw = localStorage.getItem(CARD_KEY)
    if (raw === null) {
      return null
    }
    const parsed: unknown = JSON.parse(raw)
    return isStoredCard(parsed) ? parsed : null
  } catch {
    return null
  }
}

function storeCard(r: RegistrationResult) {
  try {
    const card: StoredCard = { result: r, savedAt: new Date().toISOString() }
    localStorage.setItem(CARD_KEY, JSON.stringify(card))
  } catch {
    // Lihat readCard.
  }
}

const storedCard = readCard()
const result = ref<RegistrationResult | null>(storedCard?.result ?? null)
const restoredAt = ref<string | null>(storedCard?.savedAt ?? null)

// Satu perangkat bisa dipakai mendaftarkan teman. Kartu lama tetap tersimpan sampai pendaftaran baru berhasil.
async function registerAnother() {
  result.value = null
  restoredAt.value = null
  await nextTick()
  document.getElementById('formulir')?.scrollIntoView()
}
const accessCard = useTemplateRef<InstanceType<typeof AccessCard>>('accessCard')

// Key yang sama dipakai ulang selama isi formulir tidak berubah. Kalau kiriman pertama sebenarnya sudah
// sampai tapi jawabannya hilang karena sinyal putus, kiriman ulang mendapat pendaftaran yang sama dari server.
let attempt: { fingerprint: string; key: string } | null = null

async function onRegister(draft: RegistrationDraft) {
  const fingerprint = JSON.stringify(draft)
  if (attempt === null || attempt.fingerprint !== fingerprint) {
    attempt = { fingerprint, key: newIdempotencyKey() }
  }

  submitting.value = true
  formError.value = null
  serverErrors.value = {}
  const res = await api.register({ ...draft, idempotency_key: attempt.key })
  submitting.value = false

  if (res.ok) {
    result.value = res.data
    restoredAt.value = null
    storeCard(res.data)
    attempt = null
    refresh()
    await nextTick()
    accessCard.value?.focus()
    return
  }

  const { code, message, fields } = res.error
  if (code === 'registration_closed') {
    // Formulir disembunyikan dan panel kursi berganti jadi "Pendaftaran sudah ditutup." setelah data diperbarui.
    refresh()
    return
  }

  const known: FieldErrors = {}
  const other: string[] = []
  for (const f of fields) {
    if (FORM_FIELD.test(f.field)) {
      known[f.field] = f.message
    } else {
      other.push(f.message)
    }
  }
  serverErrors.value = known
  formError.value =
    other.length > 0 ? other.join(' ') : Object.keys(known).length > 0 ? null : message
}
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

// Pengunjung yang sudah punya kartu tidak perlu membuka amplop lagi.
const opened = ref(readOpened() || storedCard !== null)
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
          <!-- Link biasa supaya tetap jalan walau JavaScript terlambat. Di layar lebar formulir sudah terlihat
               di kolom kanan, jadi tombol ini disembunyikan. Gayanya sama dengan BaseButton primary. -->
          <a
            v-if="result || eventState.event.is_open"
            href="#formulir"
            class="inline-flex min-h-12 w-full items-center justify-center rounded-paper bg-paper px-5 font-semibold text-void transition-[transform,background-color] duration-150 hover:bg-fg active:translate-y-px active:bg-fg lg:hidden"
          >
            {{
              result
                ? 'Lihat kartu akses'
                : eventState.event.seats_left === 0
                  ? 'Masuk daftar tunggu'
                  : 'Konfirmasi kehadiran'
            }}
          </a>
        </template>
      </div>

      <div class="flex min-w-0 flex-col gap-8">
        <div id="formulir" class="scroll-mt-8">
          <div v-if="result" class="flex flex-col gap-4">
            <AccessCard
              ref="accessCard"
              :result="result"
              :film-title="
                eventState.kind === 'ready' ? eventState.event.film_title : 'Avengers: Doomsday'
              "
              :show-at="eventState.kind === 'ready' ? eventState.event.show_at : null"
              :saved-at="restoredAt"
            />
            <BaseButton variant="text" class="self-center" @click="registerAnother">
              Daftarkan orang lain
            </BaseButton>
          </div>
          <RegistrationForm
            v-else-if="eventState.kind === 'ready' && eventState.event.is_open"
            :event="eventState.event"
            :submitting="submitting"
            :server-errors="serverErrors"
            :form-error="formError"
            @submit="onRegister"
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
