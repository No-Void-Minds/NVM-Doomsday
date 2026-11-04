<script setup lang="ts">
import { ref, useId } from 'vue'

import type { AccordionItem } from './accordion'

const props = withDefaults(
  defineProps<{ items: AccordionItem[]; initiallyOpen?: number | null }>(),
  {
    initiallyOpen: null,
  },
)

const baseId = useId()
const open = ref(new Set<number>(props.initiallyOpen === null ? [] : [props.initiallyOpen]))
const buttons: (HTMLButtonElement | null)[] = []

function toggle(i: number) {
  const next = new Set(open.value)
  if (next.has(i)) {
    next.delete(i)
  } else {
    next.add(i)
  }
  open.value = next
}

// Escape menutup jawaban yang sedang dibaca dan mengembalikan fokus ke pertanyaannya,
// supaya pengguna keyboard tidak kehilangan posisi.
function close(i: number) {
  if (!open.value.has(i)) {
    return
  }
  const next = new Set(open.value)
  next.delete(i)
  open.value = next
  buttons[i]?.focus()
}
</script>

<template>
  <ul class="divide-y divide-line/40 rounded-paper bg-panel">
    <li v-for="(item, i) in items" :key="item.question">
      <h3>
        <button
          :id="`${baseId}-q${i}`"
          :ref="(el) => (buttons[i] = el as HTMLButtonElement | null)"
          type="button"
          class="flex min-h-14 w-full items-center justify-between gap-4 px-5 py-4 text-left font-heading text-lg text-fg transition-colors hover:bg-void/40 active:bg-void/40 sm:px-6"
          :aria-expanded="open.has(i)"
          :aria-controls="`${baseId}-a${i}`"
          @click="toggle(i)"
          @keydown.esc="close(i)"
        >
          <span>{{ item.question }}</span>
          <svg
            class="size-5 shrink-0 transition-transform duration-200"
            :class="open.has(i) && 'rotate-180'"
            viewBox="0 0 20 20"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M5 7.5 10 12.5 15 7.5"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </h3>
      <div
        v-show="open.has(i)"
        :id="`${baseId}-a${i}`"
        role="region"
        :aria-labelledby="`${baseId}-q${i}`"
        class="px-5 pb-5 text-fg-muted sm:px-6"
        @keydown.esc="close(i)"
      >
        <p>{{ item.answer }}</p>
      </div>
    </li>
  </ul>
</template>
