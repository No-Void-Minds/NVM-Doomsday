<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'primary' | 'outline' | 'text'
    type?: 'button' | 'submit'
    loading?: boolean
    disabled?: boolean
    block?: boolean
  }>(),
  { variant: 'primary', type: 'button', loading: false, disabled: false, block: false },
)

// Tombol utama memakai krem NVM dengan teks hitam (15,54:1), satu-satunya bidang terang di halaman gelap.
// Setiap efek hover punya padanan :active untuk layar sentuh.
const variantClass = {
  primary: 'min-h-12 bg-paper px-5 text-void hover:bg-fg active:bg-fg',
  outline:
    'min-h-12 border border-line px-5 text-fg hover:border-paper hover:bg-panel active:border-paper active:bg-panel',
  text: 'min-h-11 px-2 text-fg underline underline-offset-4 hover:text-fg-muted active:text-fg-muted',
} as const
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
    class="inline-flex items-center justify-center gap-2 rounded-paper font-semibold transition-[transform,background-color,border-color,filter] duration-150 active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60 disabled:active:translate-y-0"
    :class="[variantClass[variant], block && 'w-full']"
  >
    <!-- Putaran hanya muncul setelah pengguna menekan tombol, dan berhenti sendiri bila gerak dikurangi. -->
    <svg
      v-if="loading"
      class="size-4 shrink-0 animate-spin"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-opacity="0.3" stroke-width="3" />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        stroke-width="3"
        stroke-linecap="round"
      />
    </svg>
    <!-- Teks saat memuat ditentukan pemakai komponen, misalnya "Mengirim...", supaya status tidak hanya dari ikon. -->
    <slot />
  </button>
</template>
