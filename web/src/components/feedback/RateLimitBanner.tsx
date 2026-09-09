import { useEffect, useState } from 'react'
import { Alert, AlertTitle, Collapse } from '@mui/material'
import { useRateLimit } from '@/lib/rateLimit'

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
