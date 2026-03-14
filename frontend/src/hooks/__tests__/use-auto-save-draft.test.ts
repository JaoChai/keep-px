import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAutoSaveDraft, loadDraft, hasDraft } from '../use-auto-save-draft'

// Mock localStorage (jsdom localStorage.clear may not be available)
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value }),
    removeItem: vi.fn((key: string) => { delete store[key] }),
    clear: vi.fn(() => { store = {} }),
    get length() { return Object.keys(store).length },
    key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
  }
})()
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock })

describe('useAutoSaveDraft', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorageMock.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should save to localStorage after debounce', () => {
    const key = 'test-draft'
    const data = { name: 'Hello' }

    // First render — skipped (isFirstRender)
    const { rerender } = renderHook(
      ({ data }) => useAutoSaveDraft(key, data),
      { initialProps: { data } },
    )

    // Second render triggers the effect
    rerender({ data: { name: 'Updated' } })

    // Before debounce fires
    expect(localStorageMock.setItem).not.toHaveBeenCalled()

    // Advance past the 2000ms debounce
    act(() => { vi.advanceTimersByTime(2500) })

    expect(localStorageMock.setItem).toHaveBeenCalledWith(key, JSON.stringify({ name: 'Updated' }))
  })

  it('should clear draft from localStorage', () => {
    const key = 'test-draft'
    localStorageMock.setItem(key, JSON.stringify({ name: 'Saved' }))
    vi.clearAllMocks() // clear the setItem call above from mock history

    const { result } = renderHook(() => useAutoSaveDraft(key, { name: 'Saved' }))

    act(() => { result.current.clearDraft() })

    expect(localStorageMock.removeItem).toHaveBeenCalledWith(key)
  })
})

describe('loadDraft', () => {
  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllMocks()
  })

  it('should return parsed data from localStorage', () => {
    const data = { name: 'Test', count: 42 }
    localStorageMock.setItem('draft-key', JSON.stringify(data))

    const result = loadDraft<{ name: string; count: number }>('draft-key')
    expect(result).toEqual(data)
  })

  it('should return null for missing key', () => {
    const result = loadDraft('nonexistent-key')
    expect(result).toBeNull()
  })

  it('should return null for invalid JSON', () => {
    localStorageMock.setItem('bad-json', '{invalid')
    const result = loadDraft('bad-json')
    expect(result).toBeNull()
  })
})

describe('hasDraft', () => {
  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllMocks()
  })

  it('should return true when draft exists', () => {
    localStorageMock.setItem('draft-key', JSON.stringify({ foo: 'bar' }))
    expect(hasDraft('draft-key')).toBe(true)
  })

  it('should return false when draft does not exist', () => {
    expect(hasDraft('nonexistent-key')).toBe(false)
  })
})
