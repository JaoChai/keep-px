import { useEffect, useCallback, useRef } from 'react'
import { useBlocker } from 'react-router'

export function useUnsavedChanges(isDirty: boolean) {
  const allowNextRef = useRef(false)

  // Browser close/refresh
  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [isDirty])

  // In-app navigation — function form reads ref inside callback, not during render
  const blocker = useBlocker(() => {
    if (allowNextRef.current) return false
    return isDirty
  })

  const confirmLeave = useCallback(() => {
    if (blocker.state === 'blocked') {
      blocker.proceed()
    }
  }, [blocker])

  const cancelLeave = useCallback(() => {
    if (blocker.state === 'blocked') {
      blocker.reset()
    }
  }, [blocker])

  // Call before programmatic navigate() to bypass the blocker
  const allowNavigation = useCallback(() => {
    allowNextRef.current = true
  }, [])

  return {
    isBlocked: blocker.state === 'blocked',
    confirmLeave,
    cancelLeave,
    allowNavigation,
  }
}
