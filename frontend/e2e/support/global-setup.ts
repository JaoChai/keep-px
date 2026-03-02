import { TEST_USER } from '../fixtures/test-data'
import path from 'path'
import fs from 'fs'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

async function globalSetup() {
  const authDir = path.resolve(__dirname, '../.auth')
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true })
  }

  const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
  const apiBase = `${baseURL}/api/v1`
  const storagePath = path.resolve(authDir, 'user.json')

  let tokens: { access_token: string; refresh_token: string }

  // Try to register via API (may fail if user already exists)
  try {
    const res = await fetch(`${apiBase}/auth/register`, {
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
    const res = await fetch(`${apiBase}/auth/login`, {
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
