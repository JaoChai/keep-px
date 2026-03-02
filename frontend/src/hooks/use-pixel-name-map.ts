import { useMemo } from 'react'
import { usePixels } from './use-pixels'

export function usePixelNameMap() {
  const { data: pixels } = usePixels()
  return useMemo(() => {
    const map = new Map<string, string>()
    if (pixels) {
      for (const p of pixels) {
        map.set(p.id, p.name)
      }
    }
    return map
  }, [pixels])
}
