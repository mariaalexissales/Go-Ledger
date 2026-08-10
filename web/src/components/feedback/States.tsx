import { Alert, AlertTitle, Box, Button, Skeleton, Stack, Typography } from '@mui/material'
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
