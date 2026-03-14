import { useEffect, useCallback } from 'react'
import { useBlocker } from 'react-router'

export function useUnsavedChanges(isDirty: boolean) {
  // Browser close/refresh
  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [isDirty])

  // In-app navigation
  const blocker = useBlocker(isDirty)

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

  return {
    isBlocked: blocker.state === 'blocked',
    confirmLeave,
    cancelLeave,
  }
}
