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

  // Priority 1: Explicit tokens from env vars (for production/CI)
  let accessToken = process.env.E2E_ACCESS_TOKEN
  let refreshToken = process.env.E2E_REFRESH_TOKEN

  // Priority 2: Auto dev-login (for local development — no manual token needed)
  if (!accessToken || !refreshToken) {
    const email = process.env.E2E_USER_EMAIL || 'anugooltippon@gmail.com'
    const apiURL = baseURL
      .replace('localhost:5173', 'localhost:8080')
      .replace('pixlinks.xyz', 'api.pixlinks.xyz')

    try {
      console.log(`[global-setup] Attempting dev-login for ${email}...`)
      const res = await fetch(`${apiURL}/api/v1/auth/dev-login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })

      if (res.ok) {
        const { data } = (await res.json()) as {
          data: { access_token: string; refresh_token: string }
        }
        accessToken = data.access_token
        refreshToken = data.refresh_token
        console.log('[global-setup] Dev-login successful — fresh tokens obtained')
      } else {
        const body = await res.text()
        console.warn(
          `[global-setup] Dev-login failed (${res.status}): ${body}. ` +
            'This is expected on production — provide E2E_ACCESS_TOKEN and E2E_REFRESH_TOKEN env vars instead.'
        )
      }
    } catch (err) {
      console.warn(
        `[global-setup] Dev-login request failed: ${err}. ` +
          'Backend may not be running yet or dev-login is disabled (ENV != development).'
      )
    }
  }

  if (!accessToken || !refreshToken) {
    console.warn(
      '[global-setup] No tokens available — writing empty auth state. ' +
        'Tests requiring authentication will fail.'
    )
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
