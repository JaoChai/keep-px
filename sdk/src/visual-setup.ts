interface VisualSetupConfig {
  apiKey: string
  endpoint?: string
}

interface ElementInfo {
  tagName: string
  id: string
  className: string
  text: string
  cssSelector: string
  xpath: string
  rect: { top: number; left: number; width: number; height: number }
}

export class VisualSetup {
  private config: Required<VisualSetupConfig>
  private overlay: HTMLDivElement | null = null
  private highlight: HTMLDivElement | null = null
  private active = false

  constructor(config: VisualSetupConfig) {
    this.config = {
      apiKey: config.apiKey,
      endpoint: config.endpoint || 'https://api.pixlinks.io',
    }
  }

  activate(): void {
    if (this.active) return
    this.active = true

    this.overlay = document.createElement('div')
    this.overlay.id = 'pixlinks-visual-overlay'
    this.overlay.style.cssText = 'position:fixed;inset:0;z-index:99998;cursor:crosshair;'
    document.body.appendChild(this.overlay)

    this.highlight = document.createElement('div')
    this.highlight.id = 'pixlinks-highlight'
    this.highlight.style.cssText = 'position:fixed;z-index:99999;pointer-events:none;border:2px solid #4f46e5;background:rgba(79,70,229,0.1);transition:all 0.1s;display:none;'
    document.body.appendChild(this.highlight)

    this.overlay.addEventListener('mousemove', this.onMouseMove)
    this.overlay.addEventListener('click', this.onClick)
    document.addEventListener('keydown', this.onKeyDown)
  }

  deactivate(): void {
    if (!this.active) return
    this.active = false

    this.overlay?.removeEventListener('mousemove', this.onMouseMove)
    this.overlay?.removeEventListener('click', this.onClick)
    document.removeEventListener('keydown', this.onKeyDown)

    this.overlay?.remove()
    this.highlight?.remove()
    this.overlay = null
    this.highlight = null
  }

  private onMouseMove = (e: MouseEvent): void => {
    if (!this.highlight || !this.overlay) return

    this.overlay.style.pointerEvents = 'none'
    const el = document.elementFromPoint(e.clientX, e.clientY)
    this.overlay.style.pointerEvents = 'auto'

    if (el && el !== this.overlay && el !== this.highlight) {
      const rect = el.getBoundingClientRect()
      this.highlight.style.display = 'block'
      this.highlight.style.top = `${rect.top}px`
      this.highlight.style.left = `${rect.left}px`
      this.highlight.style.width = `${rect.width}px`
      this.highlight.style.height = `${rect.height}px`
    }
  }

  private onClick = (e: MouseEvent): void => {
    e.preventDefault()
    e.stopPropagation()

    if (!this.overlay) return

    this.overlay.style.pointerEvents = 'none'
    const el = document.elementFromPoint(e.clientX, e.clientY) as HTMLElement
    this.overlay.style.pointerEvents = 'auto'

    if (el && el !== this.overlay && el !== this.highlight) {
      const info = this.getElementInfo(el)
      // Send selected element info to parent (dashboard iframe)
      window.parent.postMessage(
        { type: 'pixlinks:element-selected', data: info },
        '*'
      )
    }
  }

  private onKeyDown = (e: KeyboardEvent): void => {
    if (e.key === 'Escape') {
      this.deactivate()
      window.parent.postMessage({ type: 'pixlinks:setup-cancelled' }, '*')
    }
  }

  private getElementInfo(el: HTMLElement): ElementInfo {
    const rect = el.getBoundingClientRect()
    return {
      tagName: el.tagName.toLowerCase(),
      id: el.id,
      className: el.className,
      text: (el.textContent || '').trim().slice(0, 100),
      cssSelector: this.getCSSSelector(el),
      xpath: this.getXPath(el),
      rect: {
        top: rect.top,
        left: rect.left,
        width: rect.width,
        height: rect.height,
      },
    }
  }

  private getCSSSelector(el: HTMLElement): string {
    if (el.id) return `#${el.id}`

    const parts: string[] = []
    let current: HTMLElement | null = el

    while (current && current !== document.body) {
      let selector = current.tagName.toLowerCase()
      if (current.id) {
        selector = `#${current.id}`
        parts.unshift(selector)
        break
      }
      if (current.className) {
        const classes = current.className.trim().split(/\s+/).slice(0, 2).join('.')
        selector += `.${classes}`
      }
      const parent: HTMLElement | null = current.parentElement
      if (parent) {
        const siblings = Array.from(parent.children).filter(
          (c: Element) => c.tagName === current!.tagName
        )
        if (siblings.length > 1) {
          const index = siblings.indexOf(current) + 1
          selector += `:nth-of-type(${index})`
        }
      }
      parts.unshift(selector)
      current = parent
    }

    return parts.join(' > ')
  }

  private getXPath(el: HTMLElement): string {
    const parts: string[] = []
    let current: HTMLElement | null = el

    while (current && current !== document.body) {
      let index = 1
      let sibling = current.previousElementSibling

      while (sibling) {
        if (sibling.tagName === current.tagName) index++
        sibling = sibling.previousElementSibling
      }

      parts.unshift(`${current.tagName.toLowerCase()}[${index}]`)
      current = current.parentElement
    }

    return `/body/${parts.join('/')}`
  }
}
