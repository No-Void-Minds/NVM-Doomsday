<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, useId, useTemplateRef } from 'vue'

import { api } from '@/api/client'
import type { ApiError, VipAccount, VipAddResult } from '@/api/types'
import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'
import BaseSkeleton from '@/components/BaseSkeleton.vue'
import { normalizeAccount } from '@/lib/validation'

const emit = defineEmits<{ unauthorized: [message: string] }>()

// Sama dengan vip.MaxBatch di backend.
const MAX_BATCH = 500

type ListState =
  | { kind: 'loading' }
  | { kind: 'ready'; accounts: VipAccount[] }
  | { kind: 'error'; message: string }

const list = shallowRef<ListState>({ kind: 'loading' })
const actionError = ref<string | null>(null)

// Sesi admin bisa habis di tengah jalan. Halaman admin yang memutuskan kembali ke form masuk.
function handleError(error: ApiError): string | null {
  if (error.code === 'unauthorized') {
    emit('unauthorized', error.message)
    return null
  }
  return error.fields[0]?.message ?? error.message
}

// Muat ulang setelah tambah atau ubah status dilakukan diam-diam, supaya daftar tidak berkedip.
async function load(silent = false) {
  if (!silent) {
    list.value = { kind: 'loading' }
  }
  const res = await api.adminVipList()
  if (res.ok) {
    list.value = { kind: 'ready', accounts: res.data.accounts }
    return
  }
  const message = handleError(res.error)
  if (message !== null) {
    if (silent && list.value.kind === 'ready') {
      actionError.value = message
    } else {
      list.value = { kind: 'error', message }
    }
  }
}

onMounted(() => load())

const accounts = computed(() => (list.value.kind === 'ready' ? list.value.accounts : []))
const activeCount = computed(() => accounts.value.filter((a) => a.is_active).length)

const query = ref('')
const filtered = computed(() => {
  const q = normalizeAccount(query.value)
  return q === ''
    ? accounts.value
    : accounts.value.filter((a) => normalizeAccount(a.account_name).includes(q))
})

const namesText = ref('')
const addError = ref<string | null>(null)
const adding = ref(false)
const report = shallowRef<VipAddResult | null>(null)
const textareaId = useId()
const textarea = useTemplateRef<HTMLTextAreaElement>('textarea')

// Salinan beberapa kolom dari Excel dipisah tab. Hanya kolom pertama yang dipakai sebagai nama akun.
function parseNames(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.split('\t')[0] ?? '')
    .filter((line) => line.trim() !== '')
}

async function add() {
  const names = parseNames(namesText.value)
  if (names.length === 0) {
    addError.value = 'Isi minimal satu nama akun.'
    textarea.value?.focus()
    return
  }
  if (names.length > MAX_BATCH) {
    addError.value = `Maksimal ${MAX_BATCH} nama sekali tambah. Bagi jadi beberapa kali.`
    textarea.value?.focus()
    return
  }

  adding.value = true
  addError.value = null
  actionError.value = null
  const res = await api.adminVipAdd(names)
  adding.value = false

  if (!res.ok) {
    addError.value = handleError(res.error)
    return
  }
  report.value = res.data
  // Hanya nama yang ditolak yang tersisa, supaya admin tinggal membetulkannya lalu menekan tombol lagi.
  namesText.value = res.data.rejected.map((r) => r.account_name).join('\n')
  await load(true)
}

const busyId = ref<string | null>(null)

async function toggle(account: VipAccount) {
  busyId.value = account.id
  actionError.value = null
  const res = await api.adminVipSetActive(account.id, !account.is_active)
  busyId.value = null

  if (!res.ok) {
    actionError.value = handleError(res.error)
    return
  }
  await load(true)
}

function statusText(a: VipAccount): string {
  if (!a.is_active) {
    return a.registration_code ? `Nonaktif, tetap dipakai ${a.registration_code}` : 'Nonaktif'
  }
  return a.registration_code ? `Dipakai ${a.registration_code}` : 'Belum dipakai'
}
</script>

<template>
  <BasePanel aria-labelledby="vip-title">
    <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h2 id="vip-title" class="text-2xl">Daftar VIP</h2>
      <p v-if="list.kind === 'ready'" class="font-mono text-sm tabular-nums text-fg-muted">
        {{ activeCount }} aktif dari {{ accounts.length }} nama
      </p>
    </div>
    <p class="mt-2 text-fg-muted">
      Nama akun di daftar ini mendapat formulir VIP. Nama lain mendaftar sebagai Solo.
    </p>

    <form class="mt-6 flex flex-col gap-3" novalidate @submit.prevent="add">
      <label :for="textareaId" class="text-sm font-medium text-fg">Nama akun</label>
      <textarea
        :id="textareaId"
        ref="textarea"
        v-model="namesText"
        rows="5"
        :disabled="adding"
        :aria-invalid="addError ? true : undefined"
        :aria-describedby="`${textareaId}-hint${addError ? ` ${textareaId}-error` : ''}`"
        class="w-full min-w-0 rounded-paper border bg-void px-4 py-3 font-mono text-fg transition-colors placeholder:text-fg-muted focus:border-paper disabled:cursor-not-allowed disabled:opacity-60"
        :class="addError ? 'border-error' : 'border-line'"
        @input="addError = null"
      />
      <p :id="`${textareaId}-hint`" class="text-sm text-fg-muted">
        Satu nama per baris, maksimal {{ MAX_BATCH }} sekali tambah. Bisa ditempel langsung dari
        satu kolom Excel.
      </p>
      <p v-if="addError" :id="`${textareaId}-error`" class="text-sm text-error">{{ addError }}</p>
      <BaseButton type="submit" :loading="adding" class="sm:self-start">
        {{ adding ? 'Menambahkan...' : 'Tambah ke daftar VIP' }}
      </BaseButton>
    </form>

    <div role="status">
      <div v-if="report" class="mt-4 rounded-paper border border-line/60 p-4 text-sm">
        <p v-if="report.added.length > 0">Ditambah: {{ report.added.length }} nama.</p>
        <p v-if="report.reactivated.length > 0">
          Diaktifkan lagi: {{ report.reactivated.length }} nama.
        </p>
        <p v-if="report.existing.length > 0" class="text-fg-muted">
          Sudah ada di daftar: {{ report.existing.length }} nama.
        </p>
        <template v-if="report.rejected.length > 0">
          <p class="mt-2 text-error">
            Ditolak: {{ report.rejected.length }} nama. Perbaiki di kotak di atas, lalu tambah lagi.
          </p>
          <ul class="mt-1 flex flex-col gap-1">
            <li v-for="r in report.rejected" :key="r.account_name" class="break-all">
              <span class="font-mono">{{ r.account_name }}</span>
              <span class="text-fg-muted">: {{ r.message }}</span>
            </li>
          </ul>
        </template>
      </div>
    </div>

    <div class="mt-8 border-t border-line/40 pt-6">
      <div v-if="list.kind === 'loading'" aria-busy="true" class="flex flex-col gap-3">
        <span class="sr-only">Memuat daftar VIP...</span>
        <BaseSkeleton class="h-12 w-full" />
        <BaseSkeleton class="h-6 w-full" />
        <BaseSkeleton class="h-6 w-full" />
      </div>

      <div v-else-if="list.kind === 'error'" role="alert">
        <p class="font-heading text-xl font-bold">Daftar VIP gagal dimuat.</p>
        <p class="mt-2 text-fg-muted">{{ list.message }}</p>
        <BaseButton variant="outline" class="mt-6" block @click="load()">Coba lagi</BaseButton>
      </div>

      <p v-else-if="accounts.length === 0" class="text-fg-muted">
        Belum ada nama VIP. Semua pendaftar terbaca Solo sampai daftar ini diisi.
      </p>

      <template v-else>
        <BaseField v-model="query" label="Cari nama akun" autocomplete="off" :maxlength="50" />
        <p v-if="filtered.length === 0" class="mt-4 text-fg-muted">
          Tidak ada nama yang cocok dengan pencarian.
        </p>
        <ul v-else class="mt-4 border-t border-line/40">
          <li
            v-for="a in filtered"
            :key="a.id"
            class="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b border-line/40 py-2"
          >
            <div class="min-w-0">
              <p class="break-all font-mono" :class="a.is_active ? 'text-fg' : 'text-fg-muted'">
                {{ a.account_name }}
              </p>
              <p class="text-sm text-fg-muted">{{ statusText(a) }}</p>
            </div>
            <BaseButton
              variant="text"
              :loading="busyId === a.id"
              :disabled="busyId !== null && busyId !== a.id"
              @click="toggle(a)"
            >
              {{ a.is_active ? 'Nonaktifkan' : 'Aktifkan lagi' }}
              <span class="sr-only">{{ a.account_name }}</span>
            </BaseButton>
          </li>
        </ul>
      </template>

      <div role="status">
        <p v-if="actionError" class="mt-4 text-sm text-error">{{ actionError }}</p>
      </div>
    </div>
  </BasePanel>
</template>
