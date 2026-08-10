import { useEffect, useState } from 'react'
import { Alert, AlertTitle, Collapse } from '@mui/material'
import { useRateLimit } from '@/lib/rateLimit'

/**
 * Shown when the guard refuses one of our own ledger requests.
 *
 * The /ops console is outside the guarded group, so this only ever fires from
 * ledger traffic, which is the intended lesson rather than a bug.
 */
export function RateLimitBanner() {
  const { until } = useRateLimit()
  const [remaining, setRemaining] = useState(0)

  useEffect(() => {
    if (!until) {
      setRemaining(0)
      return
    }

    const tick = () => setRemaining(Math.max(0, Math.ceil((until - Date.now()) / 1000)))
    tick()

    const id = setInterval(tick, 500)
    return () => clearInterval(id)
  }, [until])

  return (
    <Collapse in={Boolean(until) && remaining > 0}>
      {/*
        error, not warning: the guard refusing our own request is a fault, and
        fault is what carries the RGB split. `warning` is Hot Trace, which is
        spoken for by the vulnerable-mode chip.
      */}
      <Alert
        severity="error"
        sx={{
          '& .MuiAlertTitle-root': {
            textShadow:
              '-1px 0 var(--mui-palette-estral-glitchR), 1px 0 var(--mui-palette-estral-glitchC)',
          },
        }}
      >
        <AlertTitle>Rate limited, {remaining}s remaining</AlertTitle>
        The security guard refused a request from this browser. Further requests during a block
        restart the countdown, so writes are disabled until it clears.
      </Alert>
    </Collapse>
  )
}

/** True while a block is active; used to disable mutation buttons. */
export function useIsRateLimited(): boolean {
  const { until } = useRateLimit()
  return Boolean(until && until > Date.now())
}
