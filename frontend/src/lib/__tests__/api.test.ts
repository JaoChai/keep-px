import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { AxiosAdapter, InternalAxiosRequestConfig } from 'axios'

// ใน jsdom ตัวนี้ localStorage ไม่มี setItem จริง → zustand persist เขียนไม่ได้ตอน setState
// mock ไว้ก่อน store ถูกสร้าง (เหมือนที่ use-auto-save-draft.test.ts mock localStorage)
vi.hoisted(() => {
  const s: Record<string, string> = {}
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    writable: true,
    value: {
      getItem: (k: string) => s[k] ?? null,
      setItem: (k: string, v: string) => {
        s[k] = String(v)
      },
      removeItem: (k: string) => {
        delete s[k]
      },
      clear: () => {
        Object.keys(s).forEach((k) => delete s[k])
      },
      key: (i: number) => Object.keys(s)[i] ?? null,
      get length() {
        return Object.keys(s).length
      },
    },
  })
})

// mock neon-auth เพื่อคุม token + ดัก signOut ไม่ให้ยิง Neon จริง
vi.mock('@/lib/neon-auth', () => ({
  authClient: { signOut: vi.fn(() => Promise.resolve()) },
  getAccessToken: vi.fn(() => Promise.resolve('jwt-token')),
  clearAccessToken: vi.fn(),
}))

import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { authClient } from '@/lib/neon-auth'

// response สำเร็จรูปที่ axios ยอมรับ
function ok(data: unknown, config: unknown) {
  return { data, status: 200, statusText: 'OK', headers: {}, config }
}

// error ที่ response interceptor อ่านได้ (มี response.status / data.error / config)
function fail(status: number, error: string, config: unknown) {
  return { response: { status, data: { error } }, config, message: `${status}` }
}

describe('api response interceptor — customer_not_provisioned retry', () => {
  beforeEach(() => {
    useAuthStore.setState({ customer: null, isAuthenticated: false })
    vi.clearAllMocks()
    // กิ่ง logout ใน interceptor ทำ window.location.href = '/login' → ทำให้เป็น no-op ในเทสต์
    Object.defineProperty(window, 'location', {
      value: { href: '' },
      writable: true,
      configurable: true,
    })
  })

  it('provisions via POST /auth/session, retries once, then setCustomer', async () => {
    let xCalls = 0
    let sessionCalls = 0
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      if (config.url === '/x') {
        xCalls++
        if (xCalls === 1) throw fail(401, 'customer_not_provisioned', config)
        return ok({ ok: true }, config)
      }
      if (config.url === '/auth/session') {
        sessionCalls++
        return ok({ data: { id: 'c1', email: 'a@b.c', name: 'A', plan: 'free', is_admin: false } }, config)
      }
      throw new Error('unexpected url ' + config.url)
    }) as AxiosAdapter

    const res = await api.get('/x')

    expect(res.data).toEqual({ ok: true })
    expect(xCalls).toBe(2) // original + 1 retry
    expect(sessionCalls).toBe(1) // POST /auth/session ครั้งเดียว
    expect(useAuthStore.getState().customer?.id).toBe('c1')
  })

  it('does not loop infinitely when the retried request still fails', async () => {
    let xCalls = 0
    let sessionCalls = 0
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      if (config.url === '/x') {
        xCalls++
        throw fail(401, 'customer_not_provisioned', config) // พังทุกครั้ง
      }
      if (config.url === '/auth/session') {
        sessionCalls++
        return ok({ data: { id: 'c1' } }, config)
      }
      throw new Error('unexpected url ' + config.url)
    }) as AxiosAdapter

    await expect(api.get('/x')).rejects.toBeDefined()

    expect(xCalls).toBe(2) // original + 1 retry เท่านั้น — ไม่ loop
    expect(sessionCalls).toBe(1)
    expect(authClient.signOut).toHaveBeenCalledTimes(1) // วิ่งกิ่ง logout ไม่ใช่ retry ซ้ำ
  })
})
