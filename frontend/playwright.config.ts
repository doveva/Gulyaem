import { defineConfig } from '@playwright/test'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { loadEnv } from 'vite'

const frontendDirectory = dirname(fileURLToPath(import.meta.url))
const repositoryEnvironment = loadEnv('development', resolve(frontendDirectory, '..'), '')
const apiURL = process.env.VITE_API_URL ?? repositoryEnvironment.VITE_API_URL ?? 'http://localhost:8080'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:5173',
    viewport: { width: 1280, height: 800 },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1',
    env: { VITE_API_URL: apiURL },
    url: 'http://localhost:5173/debug/geo',
    reuseExistingServer: true,
    timeout: 30_000,
  },
})
