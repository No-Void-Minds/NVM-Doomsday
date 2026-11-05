<script setup lang="ts">
import { computed, nextTick, reactive, ref, useId, useTemplateRef, watch } from 'vue'

import type { AttendeeInput, EventInfo, RegistrationType } from '@/api/types'
import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'
import { cleanAttendee, validateRegistration, type FieldErrors } from '@/lib/validation'

export interface RegistrationDraft {
  type: RegistrationType
  attendees: AttendeeInput[]
  consent: boolean
  website: string
}

const props = defineProps<{ event: EventInfo }>()
const emit = defineEmits<{ submit: [draft: RegistrationDraft] }>()

interface PersonDraft extends AttendeeInput {
  key: number
}

let nextKey = 0
function emptyPerson(): PersonDraft {
  return { key: nextKey++, full_name: '', email: '', account_name: '' }
}

const titleId = useId()
const type = ref<RegistrationType>('solo')
const primary = reactive(emptyPerson())
const guests = ref<PersonDraft[]>([])
const consent = ref(false)
// Honeypot: kolom tersembunyi yang hanya diisi bot pengisi formulir otomatis.
const website = ref('')
const errors = ref<FieldErrors>({})

const maxGuests = computed(() => props.event.max_guests_vip)
const canAddGuest = computed(() => guests.value.length < maxGuests.value)

// VIP minimal membawa satu tamu, jadi satu baris tamu disiapkan saat VIP dipilih pertama kali.
// Pindah ke Solo tidak menghapus tamu, supaya data yang sudah diketik tidak hilang kalau pendaftar berubah pikiran.
watch(type, (t) => {
  if (t === 'vip' && guests.value.length === 0) {
    guests.value.push(emptyPerson())
  }
  clearError('attendees')
})

const addGuestButton = useTemplateRef<InstanceType<typeof BaseButton>>('addGuest')

function addGuest() {
  if (canAddGuest.value) {
    guests.value.push(emptyPerson())
  }
}

async function removeGuest(key: number) {
  guests.value = guests.value.filter((g) => g.key !== key)
  // Kesalahan tamu disimpan per indeks, jadi setelah ada yang dihapus indeksnya bergeser. Dibersihkan semua
  // supaya pesan tidak menempel di baris yang salah; pemeriksaan berikutnya dibuat ulang saat kirim.
  errors.value = Object.fromEntries(
    Object.entries(errors.value).filter(
      ([path]) => !path.startsWith('attendees[') || path.startsWith('attendees[0]'),
    ),
  )
  await nextTick()
  // Fokus dipindah ke tombol tambah tamu, supaya pengguna keyboard tidak terlempar ke awal halaman.
  ;(addGuestButton.value?.$el as HTMLElement | undefined)?.focus()
}

const people = computed<PersonDraft[]>(() =>
  type.value === 'vip' ? [primary, ...guests.value] : [primary],
)

const fields = [
  { name: 'full_name', label: 'Nama lengkap', autocomplete: 'name', max: 100 },
  { name: 'email', label: 'Email', autocomplete: 'email', max: 254 },
  { name: 'account_name', label: 'Nama akun', autocomplete: 'off', max: 50 },
] as const

function clearError(path: string) {
  if (path in errors.value) {
    const next = { ...errors.value }
    delete next[path]
    errors.value = next
  }
}

// Kolom didaftarkan berurutan seperti tampilannya, supaya fokus bisa dipindah ke kolom salah pertama.
const focusables = new Map<string, { focus: () => void }>()
function register(path: string, el: unknown) {
  if (el && typeof (el as { focus?: unknown }).focus === 'function') {
    focusables.set(path, el as { focus: () => void })
  } else {
    focusables.delete(path)
  }
}

const consentInput = useTemplateRef<HTMLInputElement>('consentInput')

const errorCount = computed(() => Object.keys(errors.value).length)

async function onSubmit() {
  const attendees = people.value.map(({ full_name, email, account_name }) => ({
    full_name,
    email,
    account_name,
  }))
  errors.value = validateRegistration({ type: type.value, attendees, consent: consent.value })

  if (errorCount.value > 0) {
    await nextTick()
    const order = [
      ...people.value.flatMap((_, i) => fields.map((f) => `attendees[${i}].${f.name}`)),
      'consent',
    ]
    const first = order.find((path) => path in errors.value)
    if (first === 'consent') {
      consentInput.value?.focus()
    } else if (first) {
      focusables.get(first)?.focus()
    }
    return
  }

  emit('submit', {
    type: type.value,
    attendees: attendees.map(cleanAttendee),
    consent: consent.value,
    website: website.value,
  })
}
</script>

<template>
  <BasePanel :aria-labelledby="titleId">
    <h2 :id="titleId" class="text-2xl">Konfirmasi kehadiran</h2>
    <p class="mt-2 text-fg-muted">
      Isi data semua orang yang akan datang. Gratis, ini hanya untuk pendataan.
    </p>

    <form class="mt-6 flex flex-col gap-8" novalidate @submit.prevent="onSubmit">
      <fieldset>
        <legend class="text-sm font-medium">Pilih tipe</legend>
        <div class="mt-2 grid grid-cols-2 gap-3">
          <label
            v-for="opt in [
              { value: 'solo', label: 'Solo', hint: 'Kamu sendiri' },
              { value: 'vip', label: 'VIP', hint: `Kamu + maksimal ${maxGuests} orang` },
            ]"
            :key="opt.value"
            class="relative flex min-h-16 cursor-pointer flex-col justify-center rounded-paper border px-4 py-3 transition-colors has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-paper"
            :class="type === opt.value ? 'border-paper' : 'border-line hover:border-fg-muted'"
          >
            <input
              v-model="type"
              type="radio"
              name="registration-type"
              :value="opt.value"
              class="sr-only"
            />
            <span class="pr-6 font-heading text-lg">{{ opt.label }}</span>
            <span class="text-sm text-fg-muted">{{ opt.hint }}</span>
            <svg
              v-if="type === opt.value"
              class="absolute right-3 top-3 size-4 text-paper"
              viewBox="0 0 16 16"
              fill="none"
              aria-hidden="true"
            >
              <path
                d="m3.5 8.5 3 3 6-7"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </label>
        </div>
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="font-heading text-xl">Data kamu</legend>
        <BaseField
          v-for="f in fields"
          :key="f.name"
          :ref="(el) => register(`attendees[0].${f.name}`, el)"
          v-model="primary[f.name]"
          :label="f.label"
          :type="f.name === 'email' ? 'email' : 'text'"
          :inputmode="f.name === 'email' ? 'email' : undefined"
          :autocomplete="f.autocomplete"
          :maxlength="f.max"
          :placeholder="f.name === 'email' ? 'nama@email.com' : f.label"
          :error="errors[`attendees[0].${f.name}`]"
          required
          @update:model-value="clearError(`attendees[0].${f.name}`)"
        />
      </fieldset>

      <section v-if="type === 'vip'" class="flex flex-col gap-4" aria-labelledby="guests-title">
        <h3 id="guests-title" class="font-heading text-xl">
          Tamu ({{ guests.length }} dari {{ maxGuests }})
        </h3>
        <p v-if="errors.attendees" class="text-sm text-error">{{ errors.attendees }}</p>

        <fieldset
          v-for="(guest, gi) in guests"
          :key="guest.key"
          class="flex flex-col gap-5 rounded-paper border border-line/60 p-4 sm:p-5"
        >
          <legend class="sr-only">Tamu {{ gi + 1 }}</legend>
          <div class="flex items-center justify-between gap-4" aria-hidden="true">
            <p class="font-heading text-lg">Tamu {{ gi + 1 }}</p>
          </div>
          <BaseField
            v-for="f in fields"
            :key="f.name"
            :ref="(el) => register(`attendees[${gi + 1}].${f.name}`, el)"
            v-model="guest[f.name]"
            :label="f.label"
            :type="f.name === 'email' ? 'email' : 'text'"
            :inputmode="f.name === 'email' ? 'email' : undefined"
            autocomplete="off"
            :maxlength="f.max"
            :placeholder="f.name === 'email' ? 'nama@email.com' : f.label"
            :error="errors[`attendees[${gi + 1}].${f.name}`]"
            required
            @update:model-value="clearError(`attendees[${gi + 1}].${f.name}`)"
          />
          <BaseButton
            v-if="guests.length > 1"
            variant="text"
            class="self-start"
            @click="removeGuest(guest.key)"
          >
            <svg class="size-4" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path
                d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            Hapus Tamu {{ gi + 1 }}
          </BaseButton>
        </fieldset>

        <BaseButton
          ref="addGuest"
          variant="outline"
          block
          :disabled="!canAddGuest"
          @click="addGuest"
        >
          <svg class="size-4" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path
              d="M8 3v10M3 8h10"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
            />
          </svg>
          Tambah tamu
        </BaseButton>
        <p v-if="!canAddGuest" class="text-sm text-fg-muted">
          Sudah maksimal {{ maxGuests }} tamu.
        </p>
      </section>

      <div class="flex flex-col gap-2">
        <p class="text-sm text-fg-muted">Semua isian wajib diisi, termasuk data tamu.</p>
        <label class="flex min-h-11 cursor-pointer items-start gap-3 pt-2">
          <input
            ref="consentInput"
            v-model="consent"
            type="checkbox"
            class="mt-0.5 size-5 shrink-0 accent-paper"
            :aria-invalid="errors.consent ? true : undefined"
            :aria-describedby="errors.consent ? 'consent-error' : undefined"
            @change="clearError('consent')"
          />
          <span>Data ini hanya dipakai untuk acara nonton bareng ini.</span>
        </label>
        <p v-if="errors.consent" id="consent-error" class="text-sm text-error">
          {{ errors.consent }}
        </p>
      </div>

      <!-- Honeypot: tidak terlihat dan tidak bisa dicapai keyboard. Manusia tidak pernah mengisinya. -->
      <div class="absolute -left-[9999px] size-px overflow-hidden" aria-hidden="true">
        <label>
          Jangan diisi
          <input v-model="website" type="text" name="website" tabindex="-1" autocomplete="off" />
        </label>
      </div>

      <div class="flex flex-col gap-3">
        <p v-if="errorCount > 0" role="alert" class="text-sm text-error">
          Ada {{ errorCount }} isian yang perlu dicek.
        </p>
        <BaseButton type="submit" block>Kirim konfirmasi</BaseButton>
      </div>
    </form>
  </BasePanel>
</template>
