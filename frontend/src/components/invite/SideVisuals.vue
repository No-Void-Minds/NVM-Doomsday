<script setup lang="ts">
import { sideAssets } from '@/content/assets'

const sides = (['left', 'right'] as const).flatMap((side) => {
  const asset = sideAssets[side]
  return asset ? [{ side, asset }] : []
})
</script>

<template>
  <!-- Hanya mulai 1360px: di bawah itu bidang kosongnya kurang dari 120px dan gambar jadi garis tipis.
       fixed, supaya gambar terlihat tanpa perlu menggulir. Lebarnya persis bidang di luar konten 70rem.
       loading="lazy" membuat browser tidak mengunduhnya selama tersembunyi, jadi kuota HP tidak terpakai. -->
  <figure
    v-for="{ side, asset } in sides"
    :key="side"
    class="pointer-events-none fixed inset-y-0 z-0 m-0 hidden w-[calc((100vw_-_70rem)/2)] overflow-hidden min-[1360px]:block"
    :class="side === 'left' ? 'left-0' : 'right-0'"
  >
    <img
      :src="asset.src"
      :alt="asset.alt"
      :width="asset.width"
      :height="asset.height"
      loading="lazy"
      decoding="async"
      class="block size-full object-cover opacity-70"
    />
    <!-- Tepi yang menghadap konten memudar ke warna latar, supaya gambar tidak bersaing dengan isi undangan. -->
    <div
      class="absolute inset-0 from-void/0 from-55% to-void"
      :class="side === 'left' ? 'bg-linear-to-r' : 'bg-linear-to-l'"
      aria-hidden="true"
    />
    <figcaption
      v-if="asset.caption"
      class="absolute inset-x-0 bottom-0 bg-void/85 p-3 text-xs text-fg-muted"
    >
      {{ asset.caption }}
    </figcaption>
  </figure>
</template>
