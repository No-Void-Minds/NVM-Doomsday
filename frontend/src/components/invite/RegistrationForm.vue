<script setup lang="ts">
import { computed, nextTick, reactive, ref, useId, useTemplateRef, watch } from 'vue'

import { api } from '@/api/client'
import type { AttendeeInput, EventInfo, RegistrationType } from '@/api/types'
import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'
import {
  cleanAttendee,
  validateAccount,
  validateRegistration,
  type FieldErrors,
} from '@/lib/validation'

export interface RegistrationDraft {
  attendees: AttendeeInput[]
  consent: boolean
  website: string
}

const props = withDefaults(
  defineProps<{
    event: EventInfo
    submitting?: boolean
    // Error dari server yang punya kolom, misalnya email yang sudah terdaftar.
    serverErrors?: FieldErrors
    // Error dari server tanpa kolom, misalnya sinyal putus atau terlalu banyak percobaan.
    formError?: string | null
  }>(),
  { submitting: false, serverErrors: () => ({}), formError: null },
)
const emit = defineEmits<{ submit: [draft: RegistrationDraft] }>()

interface PersonDraft extends AttendeeInput {
  key: number
}

let nextKey = 0
function emptyPerson(): PersonDraft {
  return { key: nextKey++, full_name: '', email: '', account_name: '' }
}

const titleId = useId()

// Langkah pertama: nama akun. Hasil cek hanya menentukan tampilan formulir, karena server memutuskan
// ulang VIP atau Solo saat formulir dikirim.
const type = ref<RegistrationType | null>(null)
const accountInput = ref('')
const accountError = ref<string | null>(null)
const checking = ref(false)
const accountField = useTemplateRef<InstanceType<typeof BaseField>>('accountField')

const primary = reactive(emptyPerson())
const guests = ref<PersonDraft[]>([])
const consent = ref(false)
// Honeypot: kolom tersembunyi yang hanya diisi bot pengisi formulir otomatis.
const website = ref('')
const errors = ref<FieldErrors>({})

const maxGuests = computed(() => props.event.max_guests_vip)
const canAddGuest = computed(() => guests.value.length < maxGuests.value)

async function checkAccount() {
  const msg = validateAccount(accountInput.value)
  if (msg) {
    accountError.value = msg
    accountField.value?.focus()
    return
  }

  const name = accountInput.value.trim()
  checking.value = true
  accountError.value = null
  const res = await api.checkAccount(name)
  checking.value = false

  if (!res.ok) {
    const field = res.error.fields.find((f) => f.field === 'account_name')
    accountError.value = field?.message ?? res.error.message
    accountField.value?.focus()
    return
  }

  primary.account_name = name
  type.value = res.data.type
  errors.value = {}
  await nextTick()
  focusables.get('attendees[0].full_name')?.focus()
}

// Kembali ke langkah nama akun. Data yang sudah diketik tetap disimpan, termasuk tamu.
async function changeAccount(message: string | null = null) {
  type.value = null
  accountError.value = message
  errors.value = {}
  await nextTick()
  accountField.value?.focus()
}

const addGuestButton = useTemplateRef<InstanceType<typeof BaseButton>>('addGuest')

async function addGuest() {
  if (!canAddGuest.value) {
    return
  }
  guests.value.push(emptyPerson())
  clearError('attendees')
  await nextTick()
  focusables.get(`attendees[${guests.value.length}].full_name`)?.focus()
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
// Keputusan akhir terkonfirmasi atau daftar tunggu tetap di server. Ini hanya pemberitahuan awal,
// dan angka sisa kursinya bisa sedikit basi kalau ada pendaftar lain yang lebih dulu.
const headcount = computed(() => people.value.length)
const isFull = computed(() => props.event.seats_left === 0)
const exceedsSeats = computed(() => !isFull.value && headcount.value > props.event.seats_left)

// Nama akun pendaftar sudah dikunci di langkah pertama, jadi hanya tamu yang mengisinya di sini.
const primaryFields = [
  { name: 'full_name', label: 'Nama lengkap', autocomplete: 'name', max: 100 },
  { name: 'email', label: 'Email', autocomplete: 'email', max: 254 },
] as const
const guestFields = [
  ...primaryFields,
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

async function focusFirstError() {
  await nextTick()
  const order = [
    ...primaryFields.map((f) => `attendees[0].${f.name}`),
    ...guests.value.flatMap((_, i) => guestFields.map((f) => `attendees[${i + 1}].${f.name}`)),
    'consent',
  ]
  const first = order.find((path) => path in errors.value)
  if (first === 'consent') {
    consentInput.value?.focus()
  } else if (first) {
    focusables.get(first)?.focus()
  }
}

watch(
  () => props.serverErrors,
  (serverErrors) => {
    // Nama VIP keburu dipakai orang lain di antara cek dan kirim. Kolomnya ada di langkah pertama.
    const taken = serverErrors['attendees[0].account_name']
    if (taken) {
      changeAccount(taken)
      return
    }
    // Admin menonaktifkan nama ini di antara cek dan kirim, jadi server membacanya sebagai Solo.
    // Batas tamu sudah dijaga di formulir, sehingga pesan di kolom attendees saat VIP berarti hal ini.
    if (serverErrors.attendees && type.value === 'vip' && guests.value.length <= maxGuests.value) {
      changeAccount(serverErrors.attendees)
      return
    }
    if (Object.keys(serverErrors).length > 0) {
      errors.value = { ...serverErrors }
      focusFirstError()
    }
  },
)

function onSubmit() {
  const attendees = people.value.map(({ full_name, email, account_name }) => ({
    full_name,
    email,
    account_name,
  }))
  errors.value = validateRegistration({ attendees, consent: consent.value })

  if (errorCount.value > 0) {
    focusFirstError()
    return
  }

  emit('submit', {
    attendees: attendees.map(cleanAttendee),
    consent: consent.value,
    website: website.value,
  })
}
</script>

<template>
  <BasePanel :aria-labelledby="titleId">
    <h2 :id="titleId" class="text-2xl">
      {{ isFull ? 'Masuk daftar tunggu' : 'Konfirmasi kehadiran' }}
    </h2>

    <div v-if="isFull" class="mt-6 rounded-paper border-l-4 border-wait bg-void p-4">
      <p class="font-heading text-lg font-bold">Kursi utama sudah penuh</p>
      <p class="mt-1 text-fg-muted">
        Kamu masih bisa daftar. Registrasimu masuk daftar tunggu, dan admin akan menghubungi kalau
        ada kursi kosong.
      </p>
    </div>

    <form
      v-if="type === null"
      class="mt-6 flex flex-col gap-5"
      novalidate
      @submit.prevent="checkAccount"
    >
      <p class="text-fg-muted">
        Mulai dari nama akunmu. Formulir berikutnya menyesuaikan dengan nama akun itu.
      </p>
      <BaseField
        ref="accountField"
        v-model="accountInput"
        label="Nama akun"
        hint="Tulis persis seperti nama akunmu. Tanda @ di depan boleh ditulis atau tidak."
        autocomplete="off"
        :maxlength="50"
        placeholder="Nama akun"
        :error="accountError ?? undefined"
        :disabled="checking"
        required
        @update:model-value="accountError = null"
      />
      <BaseButton type="submit" block :loading="checking">
        {{ checking ? 'Memeriksa...' : 'Lanjut isi data' }}
      </BaseButton>
    </form>

    <form v-else class="mt-6" novalidate @submit.prevent="onSubmit">
      <!-- Satu fieldset disabled mengunci semua kolom dan tombol sekaligus selama pengiriman. -->
      <fieldset :disabled="submitting" class="flex min-w-0 flex-col gap-8">
        <div class="flex flex-col gap-3 rounded-paper border border-line/60 p-4 sm:p-5">
          <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <p class="min-w-0 break-words">
              <span class="text-sm text-fg-muted">Nama akun</span>
              <span class="block font-mono">{{ primary.account_name }}</span>
            </p>
            <BaseButton variant="text" @click="changeAccount()">Ganti nama akun</BaseButton>
          </div>
          <p class="text-sm" :class="type === 'vip' ? 'text-fg' : 'text-fg-muted'">
            {{
              type === 'vip'
                ? `Nama akun ini ada di daftar VIP. Kamu boleh membawa sampai ${maxGuests} tamu.`
                : 'Pendaftaran ini untuk kamu sendiri.'
            }}
          </p>
        </div>

        <fieldset class="flex flex-col gap-5">
          <legend class="font-heading text-xl font-bold">Data kamu</legend>
          <BaseField
            v-for="f in primaryFields"
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
          <p v-if="guests.length === 0" class="text-sm text-fg-muted">
            Datang sendiri juga boleh. Tambahkan tamu kalau ada yang ikut.
          </p>
          <p v-if="errors.attendees" class="text-sm text-error">{{ errors.attendees }}</p>

          <fieldset
            v-for="(guest, gi) in guests"
            :key="guest.key"
            class="flex flex-col gap-5 rounded-paper border border-line/60 p-4 sm:p-5"
          >
            <legend class="sr-only">Tamu {{ gi + 1 }}</legend>
            <p class="font-heading text-lg font-bold" aria-hidden="true">Tamu {{ gi + 1 }}</p>
            <BaseField
              v-for="f in guestFields"
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
            <BaseButton variant="text" class="self-start" @click="removeGuest(guest.key)">
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
          <p class="text-sm text-fg-muted">
            {{
              type === 'vip'
                ? 'Semua isian wajib diisi, termasuk data tamu.'
                : 'Semua isian wajib diisi.'
            }}
          </p>
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
          <!-- role="status" supaya pembaca layar mengumumkan peringatan ini saat tamu ditambah, tanpa memotong. -->
          <div role="status">
            <p v-if="exceedsSeats" class="rounded-paper border-l-4 border-wait bg-void p-4 text-sm">
              Sisa {{ event.seats_left }} kursi, rombonganmu {{ headcount }} orang. Kalau tetap
              dikirim, kalian masuk daftar tunggu bersama.
            </p>
          </div>
          <p v-if="errorCount > 0" role="alert" class="text-sm text-error">
            Ada {{ errorCount }} isian yang perlu dicek.
          </p>
          <p v-if="formError" role="alert" class="text-sm text-error">{{ formError }}</p>
          <BaseButton type="submit" block :loading="submitting">
            {{
              submitting ? 'Mengirim...' : isFull ? 'Kirim ke daftar tunggu' : 'Kirim konfirmasi'
            }}
          </BaseButton>
        </div>
      </fieldset>
    </form>
  </BasePanel>
</template>
