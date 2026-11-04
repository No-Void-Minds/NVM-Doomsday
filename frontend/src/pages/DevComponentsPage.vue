<script setup lang="ts">
import { ref } from 'vue'

import BaseButton from '@/components/BaseButton.vue'
import BaseField from '@/components/BaseField.vue'
import BasePanel from '@/components/BasePanel.vue'

const name = ref('')
const email = ref('nama@contoh')
const password = ref('')
const loading = ref(false)

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
  </main>
</template>
