<script setup lang="ts">
import type { DoomsdayAsset } from '@/content/assets'

defineProps<{ asset: DoomsdayAsset }>()
</script>

<template>
  <figure class="min-w-0">
    <!-- Lebih pipih mulai layar lebar, supaya judul dan hitung mundur tetap di layar pertama.
         object-cover memotong tepi atas dan bawah, jadi pusat perhatian foto sebaiknya di tengah. -->
    <img
      :src="asset.src"
      :alt="asset.alt"
      :width="asset.width"
      :height="asset.height"
      decoding="async"
      fetchpriority="high"
      class="block aspect-video h-auto w-full rounded-paper border border-line/40 bg-panel object-cover lg:aspect-[21/9]"
    />
    <figcaption v-if="asset.caption" class="mt-2 text-sm text-fg-muted">
      {{ asset.caption }}
    </figcaption>
  </figure>
</template>
