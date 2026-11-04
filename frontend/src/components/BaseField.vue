<script setup lang="ts">
import { computed, useId, useTemplateRef } from 'vue'

const props = withDefaults(
  defineProps<{
    label: string
    type?: 'text' | 'email' | 'password'
    hint?: string
    error?: string
    required?: boolean
    disabled?: boolean
    autocomplete?: string
    inputmode?: 'text' | 'email'
    maxlength?: number
    placeholder?: string
  }>(),
  { type: 'text', required: false, disabled: false },
)

const model = defineModel<string>({ required: true })

const id = useId()
const hintId = `${id}-hint`
const errorId = `${id}-error`

// Pembaca layar membacakan petunjuk dan pesan error setelah label, karena keduanya ditautkan lewat aria-describedby.
const describedBy = computed(
  () => [props.hint && hintId, props.error && errorId].filter(Boolean).join(' ') || undefined,
)

const input = useTemplateRef<HTMLInputElement>('input')

// Dipakai formulir untuk memindahkan fokus ke kolom salah pertama setelah validasi gagal.
defineExpose({ focus: () => input.value?.focus() })
</script>

<template>
  <div class="flex min-w-0 flex-col gap-2">
    <label :for="id" class="text-sm font-medium text-fg">
      {{ label }}<span v-if="required" class="text-fg-muted" aria-hidden="true"> *</span>
    </label>
    <input
      :id="id"
      ref="input"
      v-model="model"
      :type="type"
      :required="required"
      :disabled="disabled"
      :autocomplete="autocomplete"
      :inputmode="inputmode"
      :maxlength="maxlength"
      :placeholder="placeholder"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="describedBy"
      class="min-h-12 w-full min-w-0 rounded-paper border bg-void px-4 text-fg transition-colors placeholder:text-fg-muted focus:border-paper disabled:cursor-not-allowed disabled:opacity-60"
      :class="error ? 'border-error' : 'border-line'"
    />
    <p v-if="hint" :id="hintId" class="text-sm text-fg-muted">{{ hint }}</p>
    <p v-if="error" :id="errorId" class="flex items-start gap-1.5 text-sm text-error">
      <svg class="mt-0.5 size-4 shrink-0" viewBox="0 0 16 16" fill="none" aria-hidden="true">
        <circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.5" />
        <path d="M8 4.5v4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        <circle cx="8" cy="11" r="0.9" fill="currentColor" />
      </svg>
      <span>{{ error }}</span>
    </p>
  </div>
</template>
