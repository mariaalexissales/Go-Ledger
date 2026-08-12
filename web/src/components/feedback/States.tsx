import { Alert, AlertTitle, Box, Button, Skeleton, Stack, Typography } from '@mui/material'
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { ApiError } from '@/lib/http/errors'

export function LoadingRows({ rows = 5 }: { rows?: number }) {
  return (
    <Stack spacing={1} sx={{ py: 1 }}>
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} variant="rounded" height={40} />
      ))}
    </Stack>
  )
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: ReactNode
}) {
  return (
    <Box sx={{ textAlign: 'center', py: 6, px: 2 }}>
      <Typography variant="h3" gutterBottom>
        {title}
      </Typography>
      {description && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {description}
        </Typography>
      )}
      {action}
    </Box>
  )
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const message = error instanceof ApiError ? error.message : 'Something went wrong.'
  const status = error instanceof ApiError && error.status > 0 ? ` (${error.status})` : ''

  return (
    <Alert
      severity="error"
      action={
        onRetry && (
          <Button color="inherit" size="small" onClick={onRetry}>
            Retry
          </Button>
        )
      }
    >
      <AlertTitle>Request failed{status}</AlertTitle>
      {message}
    </Alert>
  )
}

/**
 * Router-level fallbacks.
 *
 * Without these a throw during render blanks the page, and an unknown path
 * renders the shell around an empty outlet. The second one matters more than it
 * looks on GitHub Pages: 404.html is a copy of index.html, so every mistyped
 * deep link arrives here rather than at a server error page.
 */
export function RouteErrorFallback({ error, reset }: { error: unknown; reset?: () => void }) {
  return (
    <Box sx={{ p: 3 }}>
      <ErrorState error={error} onRetry={reset} />
    </Box>
  )
}

export function RouteNotFound() {
  return (
    <EmptyState
      title="Not found"
      description="That path does not match any page in the console."
      action={
        <Button component={Link} to="/" variant="contained" size="small">
          Back to dashboard
        </Button>
      }
    />
  )
}
