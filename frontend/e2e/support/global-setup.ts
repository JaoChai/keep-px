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
  const storagePath = path.resolve(authDir, 'user.json')

  // Auth uses Google OAuth only — E2E tokens must be provided via env vars.
  // Generate them via the backend or use a long-lived test token.
  const accessToken = process.env.E2E_ACCESS_TOKEN
  const refreshToken = process.env.E2E_REFRESH_TOKEN

  if (!accessToken || !refreshToken) {
    console.warn(
      '[global-setup] E2E_ACCESS_TOKEN and E2E_REFRESH_TOKEN not set — writing empty auth state. ' +
        'Tests requiring authentication will fail.'
    )
    // Write empty storage state so Playwright config doesn't error on missing file
    fs.writeFileSync(storagePath, JSON.stringify({ cookies: [], origins: [] }, null, 2))
    return
  }

  // Write storageState with tokens in localStorage
  const storageState = {
    cookies: [],
    origins: [
      {
        origin: baseURL,
        localStorage: [
          { name: 'access_token', value: accessToken },
          { name: 'refresh_token', value: refreshToken },
        ],
      },
    ],
  }

  fs.writeFileSync(storagePath, JSON.stringify(storageState, null, 2))
}

export default globalSetup
