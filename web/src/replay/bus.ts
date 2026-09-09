import type { SecurityEvent } from '@/features/security/security.types'

type Listener = (event: SecurityEvent) => void

const MIN_WINDOW_MS = 2000
const MAX_WINDOW_MS = 5000
const MS_PER_EVENT = 30

interface Pacing {
  recordedMs: number
}

class ReplayEventBus {
  private listeners = new Set<Listener>()
  private timers: ReturnType<typeof setTimeout>[] = []

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener)
    return () => {
      this.listeners.delete(listener)
    }
  }

  private publish(event: SecurityEvent) {
    for (const listener of this.listeners) listener(event)
  }

  cancel() {
    for (const timer of this.timers) clearTimeout(timer)
    this.timers = []
  }

  schedule(events: SecurityEvent[], onEvent?: (event: SecurityEvent) => void): Promise<Pacing> {
    this.cancel()

    if (events.length === 0) {
      return Promise.resolve({ recordedMs: 0 })
    }

    const start = new Date(events[0].timestamp).getTime()
    const offsets = events.map((event) => {
      const offset = new Date(event.timestamp).getTime() - start
      return Number.isFinite(offset) && offset > 0 ? offset : 0
    })

    const recordedMs = offsets[offsets.length - 1]
    const window = clamp(events.length * MS_PER_EVENT, MIN_WINDOW_MS, MAX_WINDOW_MS)

    const scale = recordedMs > 0 ? window / recordedMs : 0

    return new Promise((resolve) => {
      events.forEach((event, i) => {
        const delay = scale > 0 ? offsets[i] * scale : (window * i) / events.length

        this.timers.push(
          setTimeout(() => {
            this.publish(event)
            onEvent?.(event)

            if (i === events.length - 1) {
              resolve({ recordedMs })
            }
          }, delay),
        )
      })
    })
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

export const replayBus = new ReplayEventBus()
