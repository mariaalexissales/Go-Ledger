import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { qk } from '@/lib/queryKeys'
import { securityApi } from './security.api'
import type { SecurityEvent } from './security.types'
import { REPLAY } from '@/replay/mode'
import { replayBus } from '@/replay/bus'

/** How many live events to keep in memory before dropping the oldest. */
const MAX_LIVE_EVENTS = 500

type StreamStatus = 'connecting' | 'open' | 'error'

interface EventStreamValue {
  events: SecurityEvent[]
  status: StreamStatus
  /** Events the server discarded because this client fell behind. */
  dropped: number
  clear: () => void
}

const EventStreamContext = createContext<EventStreamValue | null>(null)

/**
 * One EventSource for the whole app, mounted in the root layout.
 *
 * The browser sends Last-Event-ID automatically on reconnect and the server
 * replays the gap, so an HMR reload or a sleeping laptop does not silently lose
 * events. Stats queries are invalidated on a debounce rather than per event,
 * because a burst scenario delivers hundreds of events in a second.
 */
export function EventStreamProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [events, setEvents] = useState<SecurityEvent[]>([])
  const [status, setStatus] = useState<StreamStatus>('connecting')
  const [dropped, setDropped] = useState(0)
  const invalidateTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    // Replay mode has no server to stream from. Recorded events arrive on a
    // local bus at their recorded cadence instead.
    if (REPLAY) {
      setStatus('open')

      return replayBus.subscribe((event) => {
        setEvents((current) => [event, ...current].slice(0, MAX_LIVE_EVENTS))

        if (invalidateTimer.current) return
        invalidateTimer.current = setTimeout(() => {
          invalidateTimer.current = null
          queryClient.invalidateQueries({ queryKey: qk.security.all })
        }, 400)
      })
    }

    const source = new EventSource(securityApi.streamUrl())

    source.onopen = () => setStatus('open')
    source.onerror = () => setStatus('error') // EventSource retries on its own.

    source.addEventListener('security_event', (message) => {
      let event: SecurityEvent
      try {
        event = JSON.parse((message as MessageEvent<string>).data) as SecurityEvent
      } catch {
        return
      }

      setEvents((current) => [event, ...current].slice(0, MAX_LIVE_EVENTS))

      if (invalidateTimer.current) return
      invalidateTimer.current = setTimeout(() => {
        invalidateTimer.current = null
        queryClient.invalidateQueries({ queryKey: qk.security.all })
      }, 1200)
    })

    source.addEventListener('lag', (message) => {
      try {
        const payload = JSON.parse((message as MessageEvent<string>).data) as { dropped: number }
        setDropped(payload.dropped)
      } catch {
        // Ignore a malformed lag frame.
      }
    })

    return () => {
      source.close()
      if (invalidateTimer.current) clearTimeout(invalidateTimer.current)
    }
  }, [queryClient])

  const value = useMemo<EventStreamValue>(
    () => ({ events, status, dropped, clear: () => setEvents([]) }),
    [events, status, dropped],
  )

  return <EventStreamContext.Provider value={value}>{children}</EventStreamContext.Provider>
}

export function useEventStream(): EventStreamValue {
  const value = useContext(EventStreamContext)
  if (!value) throw new Error('useEventStream must be used inside EventStreamProvider')
  return value
}
