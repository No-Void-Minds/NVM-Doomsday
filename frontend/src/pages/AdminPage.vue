<script setup lang="ts">
import { computed, nextTick, onMounted, ref, shallowRef, useTemplateRef } from 'vue'

import { api, exportUrl } from '@/api/client'
import type { AdminSummary } from '@/api/types'
import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'
import BaseSkeleton from '@/components/BaseSkeleton.vue'
import VipList from '@/components/admin/VipList.vue'

type View = 'checking' | 'login' | 'ready'
type SummaryState =
  | { kind: 'loading' }
  | { kind: 'ready'; summary: AdminSummary }
  | { kind: 'error'; message: string }

const view = ref<View>('checking')
const notice = ref<string | null>(null)

const password = ref('')
const loginError = ref<string | null>(null)
const loggingIn = ref(false)
const passwordField = useTemplateRef<InstanceType<typeof BaseField>>('passwordField')

const summary = shallowRef<SummaryState>({ kind: 'loading' })
const heading = useTemplateRef<HTMLHeadingElement>('heading')
const actionError = ref<string | null>(null)
const downloading = ref<'csv' | 'xlsx' | null>(null)

const isEmpty = computed(
  () =>
    summary.value.kind === 'ready' &&
    summary.value.summary.confirmed_attendees === 0 &&
    summary.value.summary.waitlisted_attendees === 0 &&
    summary.value.summary.cancelled_registrations === 0,
)

async function toLogin(message: string | null) {
  view.value = 'login'
  notice.value = message
  password.value = ''
  loginError.value = null
  await nextTick()
  passwordField.value?.focus()
}

async function loadSummary() {
  summary.value = { kind: 'loading' }
  const res = await api.adminSummary()
  if (res.ok) {
    summary.value = { kind: 'ready', summary: res.data }
  } else if (res.error.code === 'unauthorized') {
    toLogin(res.error.message)
  } else {
    summary.value = { kind: 'error', message: res.error.message }
  }
}

async function enterDashboard() {
  view.value = 'ready'
  notice.value = null
  actionError.value = null
  await nextTick()
  heading.value?.focus()
  loadSummary()
}

// Cookie admin berlaku 8 jam. Kalau masih berlaku, admin langsung ke ringkasan tanpa mengetik sandi lagi.
onMounted(async () => {
  const res = await api.adminSession()
  if (res.ok) {
    enterDashboard()
  } else {
    toLogin(res.error.code === 'unauthorized' ? null : res.error.message)
  }
})

async function login() {
  if (password.value === '') {
    loginError.value = 'Isi sandi admin.'
    passwordField.value?.focus()
    return
  }
  loggingIn.value = true
  loginError.value = null
  const res = await api.adminLogin(password.value)
  loggingIn.value = false

  if (res.ok) {
    password.value = ''
    enterDashboard()
    return
  }
  loginError.value = res.error.message
  passwordField.value?.focus()
}

// File diunduh lewat link biasa supaya browser memakai nama dari Content-Disposition. Kalau sesi sudah berakhir,
// link itu akan menyimpan pesan error JSON sebagai file rusak, jadi sesi dicek dulu.
async function download(format: 'csv' | 'xlsx') {
  downloading.value = format
  actionError.value = null
  const res = await api.adminSession()
  downloading.value = null

  if (res.ok) {
    window.location.assign(exportUrl(format))
  } else if (res.error.code === 'unauthorized') {
    toLogin(res.error.message)
  } else {
    actionError.value = res.error.message
  }
}

async function logout() {
  actionError.value = null
  const res = await api.adminLogout()
  if (res.ok) {
    toLogin('Kamu sudah keluar.')
  } else {
    actionError.value = res.error.message
  }
}
</script>

<template>
  <main class="mx-auto w-full max-w-[40rem] px-4 py-10 sm:py-16">
    <div v-if="view === 'checking'" aria-busy="true">
      <span class="sr-only">Memeriksa sesi admin...</span>
      <BasePanel>
        <div class="flex flex-col gap-3">
          <BaseSkeleton class="h-8 w-48 max-w-full" />
          <BaseSkeleton class="h-12 w-full" />
        </div>
      </BasePanel>
    </div>

    <BasePanel v-else-if="view === 'login'">
      <h1 class="text-3xl">Admin Doomsday</h1>
      <div role="status">
        <p v-if="notice" class="mt-2 text-fg-muted">{{ notice }}</p>
      </div>
      <form class="mt-6 flex flex-col gap-5" novalidate @submit.prevent="login">
        <BaseField
          ref="passwordField"
          v-model="password"
          label="Sandi admin"
          type="password"
          autocomplete="current-password"
          :error="loginError ?? undefined"
          :disabled="loggingIn"
          required
          @update:model-value="loginError = null"
        />
        <BaseButton type="submit" block :loading="loggingIn">
          {{ loggingIn ? 'Masuk...' : 'Masuk' }}
        </BaseButton>
      </form>
    </BasePanel>

    <div v-else class="flex flex-col gap-6">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <h1 ref="heading" tabindex="-1" class="text-3xl">Rekap pendaftaran</h1>
        <BaseButton variant="text" @click="logout">Keluar</BaseButton>
      </div>

      <BasePanel>
        <div v-if="summary.kind === 'loading'" aria-busy="true" class="flex flex-col gap-4">
          <span class="sr-only">Memuat rekap...</span>
          <BaseSkeleton class="h-6 w-full" />
          <BaseSkeleton class="h-6 w-full" />
          <BaseSkeleton class="h-6 w-full" />
        </div>

        <div v-else-if="summary.kind === 'error'" role="alert">
          <p class="font-heading text-xl font-bold">Rekap gagal dimuat.</p>
          <p class="mt-2 text-fg-muted">{{ summary.message }}</p>
          <BaseButton variant="outline" class="mt-6" block @click="loadSummary"
            >Coba lagi</BaseButton
          >
        </div>

        <template v-else>
          <dl class="border-t border-line/40">
            <div class="flex items-baseline justify-between gap-4 border-b border-line/40 py-3">
              <dt>Terkonfirmasi</dt>
              <dd class="font-mono tabular-nums">
                {{ summary.summary.confirmed_attendees }} dari
                {{ summary.summary.capacity }} peserta
              </dd>
            </div>
            <div class="flex items-baseline justify-between gap-4 border-b border-line/40 py-3">
              <dt>Daftar tunggu</dt>
              <dd class="font-mono tabular-nums">
                {{ summary.summary.waitlisted_attendees }} peserta
              </dd>
            </div>
            <div class="flex items-baseline justify-between gap-4 border-b border-line/40 py-3">
              <dt>Batal</dt>
              <dd class="font-mono tabular-nums">
                {{ summary.summary.cancelled_registrations }} registrasi
              </dd>
            </div>
          </dl>

          <p v-if="isEmpty" class="mt-4 text-fg-muted">
            Belum ada peserta. File unduhan hanya berisi judul kolom.
          </p>

          <div class="mt-6 grid gap-3 sm:grid-cols-2">
            <BaseButton :loading="downloading === 'xlsx'" @click="download('xlsx')">
              Unduh Excel
            </BaseButton>
            <BaseButton variant="outline" :loading="downloading === 'csv'" @click="download('csv')">
              Unduh CSV
            </BaseButton>
          </div>
          <p class="mt-3 text-sm text-fg-muted">
            File berisi satu baris per peserta, urut terkonfirmasi lalu daftar tunggu.
          </p>

          <BaseButton variant="text" class="mt-4" @click="loadSummary">Muat ulang angka</BaseButton>
        </template>

        <div role="status">
          <p v-if="actionError" class="mt-4 text-sm text-error">{{ actionError }}</p>
        </div>
      </BasePanel>
      <VipList @unauthorized="toLogin" />
    </div>
  </main>
</template>
