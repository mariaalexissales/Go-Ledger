import type { SecurityEvent } from '@/features/security/security.types'

type Listener = (event: SecurityEvent) => void

/**
 * Recorded runs are far too fast to watch. The burst scenario is 65 requests in
 * 17 ms. Replay keeps the relative spacing between events, so the paced
 * baseline still reads differently from a flat-out burst, but stretches the
 * whole run into a window a person can follow.
 */
const MIN_WINDOW_MS = 2000
const MAX_WINDOW_MS = 5000
const MS_PER_EVENT = 30

interface Pacing {
  /** How long the run actually took on the server. */
  recordedMs: number
  /** How long the replay took here. */
  replayedMs: number
}

/**
 * Stands in for EventSource. The transport publishes recorded events onto it and
 * useEventStream subscribes, so nothing above those two layers knows the
 * difference.
 */
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

  /** Drops any scheduled-but-unsent events, so a new run starts clean. */
  cancel() {
    for (const timer of this.timers) clearTimeout(timer)
    this.timers = []
  }

  /**
   * Emits events on their recorded cadence, rescaled. Resolves once the last
   * one has been published, which keeps the demo mutation pending and the
   * progress bar visible for the length of the animation.
   */
  schedule(events: SecurityEvent[], onEvent?: (event: SecurityEvent) => void): Promise<Pacing> {
    this.cancel()

    if (events.length === 0) {
      return Promise.resolve({ recordedMs: 0, replayedMs: 0 })
    }

    const start = new Date(events[0].timestamp).getTime()
    const offsets = events.map((event) => {
      const offset = new Date(event.timestamp).getTime() - start
      return Number.isFinite(offset) && offset > 0 ? offset : 0
    })

    const recordedMs = offsets[offsets.length - 1]
    const window = clamp(events.length * MS_PER_EVENT, MIN_WINDOW_MS, MAX_WINDOW_MS)

    // When every event shares a timestamp there is no shape to preserve, so
    // spread them evenly instead of firing them all at once.
    const scale = recordedMs > 0 ? window / recordedMs : 0

    return new Promise((resolve) => {
      events.forEach((event, i) => {
        const delay = scale > 0 ? offsets[i] * scale : (window * i) / events.length

        this.timers.push(
          setTimeout(() => {
            this.publish(event)
            onEvent?.(event)

            if (i === events.length - 1) {
              resolve({ recordedMs, replayedMs: Math.round(window) })
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
