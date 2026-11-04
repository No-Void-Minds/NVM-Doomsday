<script setup lang="ts">
withDefaults(defineProps<{ as?: 'section' | 'div' | 'aside' }>(), { as: 'section' })
</script>

<template>
  <!-- Padding mobile kira-kira setengah desktop, sesuai SRS 11.5. -->
  <component :is="as" class="min-w-0 rounded-paper bg-panel p-5 sm:p-8">
    <slot />
  </component>
</template>
