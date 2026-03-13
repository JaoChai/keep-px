import { TEST_USER } from '../fixtures/test-data'
import path from 'path'
import fs from 'fs'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

async function fetchWithRetry(
  url: string,
  options: RequestInit,
  retries = 5
): Promise<Response> {
  const backoffMs = [3000, 6000, 12000, 24000, 48000]
  let lastError: Error | null = null

  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const res = await fetch(url, options)

      // Only retry on 5xx server errors
      if (res.status >= 500 && attempt < retries) {
        const delay = backoffMs[attempt] ?? 48000
        console.log(
          `[global-setup] ${url} returned ${res.status}, retrying in ${delay / 1000}s (attempt ${attempt + 1}/${retries})`
        )
        await new Promise((r) => setTimeout(r, delay))
        continue
      }

      return res
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err))
      if (attempt < retries) {
        const delay = backoffMs[attempt] ?? 48000
        console.log(
          `[global-setup] ${url} fetch error: ${lastError.message}, retrying in ${delay / 1000}s (attempt ${attempt + 1}/${retries})`
        )
        await new Promise((r) => setTimeout(r, delay))
      }
    }
  }

  throw lastError ?? new Error(`fetchWithRetry failed after ${retries} retries`)
}

async function globalSetup() {
  const authDir = path.resolve(__dirname, '../.auth')
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true })
  }

  const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
  const apiBase = process.env.E2E_API_URL
    ? `${process.env.E2E_API_URL}/api/v1`
    : `${baseURL}/api/v1`
  const storagePath = path.resolve(authDir, 'user.json')

  let tokens: { access_token: string; refresh_token: string }

  // Try to register via API (may fail if user already exists)
  try {
    const res = await fetchWithRetry(`${apiBase}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: TEST_USER.name,
        email: TEST_USER.email,
        password: TEST_USER.password,
      }),
    })
    if (!res.ok) throw new Error('register failed')
    const body = await res.json()
    tokens = body.data
  } catch {
    // Registration failed (user exists) — login instead
    const res = await fetchWithRetry(`${apiBase}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: TEST_USER.email,
        password: TEST_USER.password,
      }),
    })
    if (!res.ok) throw new Error(`E2E auth setup failed: login returned ${res.status}`)
    const body = await res.json()
    tokens = body.data
  }

  // Write storageState with tokens in localStorage
  const storageState = {
    cookies: [],
    origins: [
      {
        origin: baseURL,
        localStorage: [
          { name: 'access_token', value: tokens.access_token },
          { name: 'refresh_token', value: tokens.refresh_token },
        ],
      },
    ],
  }

  fs.writeFileSync(storagePath, JSON.stringify(storageState, null, 2))
}

export default globalSetup
