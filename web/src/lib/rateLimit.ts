import { useSyncExternalStore } from 'react'

interface RateLimitState {
  /** Unix ms when the block lifts, or null when not limited. */
  until: number | null
}

type Listener = () => void

/**
 * A tiny external store rather than context, so the fetch client can write to it
 * without being inside the React tree.
 *
 * Worth knowing: the server re-blocks on every request received during a block,
 * which extends it. That is why mutation buttons are disabled while this is
 * active instead of just showing a message.
 */
class RateLimitStore {
  private state: RateLimitState = { until: null }
  private listeners = new Set<Listener>()
  private timer: ReturnType<typeof setTimeout> | null = null

  trip(retryAfterSec: number) {
    const until = Date.now() + retryAfterSec * 1000
    if (this.state.until && this.state.until >= until) return

    this.set({ until })

    if (this.timer) clearTimeout(this.timer)
    this.timer = setTimeout(() => this.clear(), retryAfterSec * 1000)
  }

  clear() {
    if (this.timer) {
      clearTimeout(this.timer)
      this.timer = null
    }
    this.set({ until: null })
  }

  private set(state: RateLimitState) {
    this.state = state
    this.listeners.forEach((listener) => listener())
  }

  subscribe = (listener: Listener) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getSnapshot = () => this.state
}

export const rateLimitStore = new RateLimitStore()

export function useRateLimit() {
  return useSyncExternalStore(rateLimitStore.subscribe, rateLimitStore.getSnapshot)
}
