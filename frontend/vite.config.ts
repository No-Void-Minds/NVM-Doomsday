import { fileURLToPath, URL } from 'node:url'

import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '127.0.0.1',
    // 5173 dipakai NVM-Internal; strictPort supaya Vite gagal jelas, bukan diam-diam pindah port.
    port: 5174,
    strictPort: true,
    proxy: {
      // Frontend dan API satu origin di dev, sama seperti nanti di belakang Nginx,
      // jadi cookie admin dan CORS berperilaku sama di dua lingkungan.
      '/api': 'http://127.0.0.1:8081',
    },
  },
})