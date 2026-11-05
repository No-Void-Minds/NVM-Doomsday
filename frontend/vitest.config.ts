import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vitest/config'

// Terpisah dari vite.config.ts supaya plugin Tailwind dan Vue tidak ikut jalan: yang diuji hanya modul TypeScript.
export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/__tests__/*.spec.ts'],
  },
})
