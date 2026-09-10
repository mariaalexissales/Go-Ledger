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

const MAX_LIVE_EVENTS = 500

type StreamStatus = 'connecting' | 'open' | 'error'

interface EventStreamValue {
  events: SecurityEvent[]
  status: StreamStatus
  dropped: number
  clear: () => void
}

const EventStreamContext = createContext<EventStreamValue | null>(null)

export function EventStreamProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [events, setEvents] = useState<SecurityEvent[]>([])
  const [status, setStatus] = useState<StreamStatus>('connecting')
  const [dropped, setDropped] = useState(0)
  const invalidateTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const push = (event: SecurityEvent) =>
      setEvents((current) => [event, ...current].slice(0, MAX_LIVE_EVENTS))

    const scheduleInvalidate = (delayMs: number) => {
      if (invalidateTimer.current) return
      invalidateTimer.current = setTimeout(() => {
        invalidateTimer.current = null
        queryClient.invalidateQueries({ queryKey: qk.security.all })
      }, delayMs)
    }

    const clearInvalidate = () => {
      if (invalidateTimer.current) clearTimeout(invalidateTimer.current)
    }

    // Replay mode has no server to stream from. Recorded events arrive on a
    // local bus at their recorded cadence instead.
    if (REPLAY) {
      setStatus('open')

      const unsubscribe = replayBus.subscribe((event) => {
        push(event)
        scheduleInvalidate(400)
      })

      return () => {
        unsubscribe()
        clearInvalidate()
      }
    }

    const source = new EventSource(securityApi.streamUrl())

    source.onopen = () => setStatus('open')
    source.onerror = () => setStatus('error')

    source.addEventListener('security_event', (message) => {
      let event: SecurityEvent
      try {
        event = JSON.parse((message as MessageEvent<string>).data) as SecurityEvent
      } catch {
        return
      }

      push(event)
      scheduleInvalidate(1200)
    })

    source.addEventListener('lag', (message) => {
      try {
        const payload = JSON.parse((message as MessageEvent<string>).data) as { dropped: number }
        setDropped(payload.dropped)
      } catch {}
    })

    return () => {
      source.close()
      clearInvalidate()
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
