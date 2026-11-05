import { defineConfig, devices } from '@playwright/test'

import { BACKEND_URL, BASE_URL, E2E_ADMIN_PASSWORD_HASH, testDatabaseUrls } from './e2e/env'

// Gagal di sini, sebelum server apa pun dijalankan, kalau URL database uji kosong atau bukan *_test.
const db = testDatabaseUrls()

export default defineConfig({
  testDir: './e2e',
  // Semua test memakai satu database uji yang dikosongkan sebelum setiap test, jadi tidak boleh paralel.
  fullyParallel: false,
  workers: 1,
  forbidOnly: true,
  retries: 0,
  timeout: 60_000,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  globalSetup: './e2e/global-setup.ts',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: BASE_URL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command: 'go run ./cmd/server',
      cwd: '../backend',
      url: `${BACKEND_URL}/api/health`,
      // Port 8081 yang sudah terpakai biasanya backend dev yang tersambung ke nvm_doomsday. Jangan dipakai ulang.
      reuseExistingServer: false,
      timeout: 180_000,
      stdout: 'ignore',
      stderr: 'pipe',
      env: {
        APP_ENV: 'development',
        HTTP_ADDR: '127.0.0.1:8081',
        DATABASE_URL: db.app,
        // Proxy Vite di 127.0.0.1 meneruskan X-Forwarded-For dari test, supaya tiap test punya IP sendiri.
        TRUSTED_PROXIES: '127.0.0.1',
        ADMIN_PASSWORD_HASH: E2E_ADMIN_PASSWORD_HASH,
        SESSION_SECRET: 'e2e-session-secret-hanya-untuk-uji-otomatis',
        COOKIE_SECURE: 'false',
      },
    },
    {
      command: 'npx vite',
      url: BASE_URL,
      reuseExistingServer: false,
      timeout: 120_000,
      stdout: 'ignore',
    },
  ],
})
