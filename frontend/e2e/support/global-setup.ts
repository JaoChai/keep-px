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

  // Priority 1: Explicit access token from env (override if needed)
  let accessToken = process.env.E2E_ACCESS_TOKEN
  let refreshToken = process.env.E2E_REFRESH_TOKEN

  // Priority 1.5: Auto-refresh from refresh token (for production — no manual access token copy needed)
  if (!accessToken && refreshToken) {
    const apiURL = baseURL
      .replace('localhost:5173', 'localhost:8080')
      .replace('pixlinks.xyz', 'api.pixlinks.xyz')

    try {
      console.log('[global-setup] Attempting auto-refresh from refresh token...')
      const res = await fetch(`${apiURL}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })

      if (res.ok) {
        const json = await res.json()
        const data = json.data as { access_token?: string; refresh_token?: string }
        if (!data?.access_token) {
          console.warn('[global-setup] Auto-refresh response missing access_token')
        } else {
          accessToken = data.access_token
          const newRefreshToken = data.refresh_token
          if (newRefreshToken && newRefreshToken !== refreshToken) {
            refreshToken = newRefreshToken
            // Persist rotated refresh token back to .env.e2e.local
            const envPath = path.resolve(__dirname, '../../.env.e2e.local')
            try {
              let content = fs.readFileSync(envPath, 'utf-8')
              content = content.replace(
                /^E2E_REFRESH_TOKEN=.*/m,
                `E2E_REFRESH_TOKEN=${newRefreshToken}`
              )
              fs.writeFileSync(envPath, content)
              console.log('[global-setup] Rotated refresh token persisted to .env.e2e.local')
            } catch {
              console.warn('[global-setup] Could not persist rotated refresh token — update .env.e2e.local manually')
            }
          }
          console.log('[global-setup] Auto-refresh successful — fresh access token obtained')
        }
      } else {
        const body = await res.text().catch(() => '(no response body)')
        console.warn(`[global-setup] Auto-refresh failed (${res.status}): ${body}`)
        console.warn('[global-setup] Refresh token may be expired — obtain a new one from browser DevTools')
      }
    } catch (err) {
      console.warn(`[global-setup] Auto-refresh request failed: ${err}`)
    }
  }

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
